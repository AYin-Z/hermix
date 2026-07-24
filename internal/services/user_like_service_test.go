package services

import (
	"testing"

	"bbs-go/internal/models"
	"bbs-go/internal/models/constants"
	"bbs-go/internal/pkg/config"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
)

func setupLikeServiceTestDB(t *testing.T) {
	t.Helper()
	config.Instance = &config.Config{Language: config.DefaultLanguage}
	db := setupTestDB(t)
	if err := db.AutoMigrate(&models.Topic{}, &models.UserLike{}); err != nil {
		t.Fatalf("auto migrate like: %v", err)
	}
}

func mustCreateTopic(t *testing.T, authorId int64) *models.Topic {
	t.Helper()
	now := dates.NowTimestamp()
	topic := &models.Topic{
		UserId:     authorId,
		Title:      "t",
		Status:     constants.StatusOk,
		CreateTime: now,
	}
	if err := sqls.DB().Create(topic).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}
	return topic
}

func topicLikeCount(t *testing.T, topicId int64) int64 {
	t.Helper()
	var topic models.Topic
	if err := sqls.DB().Select("like_count").First(&topic, topicId).Error; err != nil {
		t.Fatalf("reload topic: %v", err)
	}
	return topic.LikeCount
}

// TestTopicUnLike_IdempotentGuardsAgainstGriefing 锁住 HIGH 修复：
// 未点赞过时 unlike 必须报错且不递减 like_count，否则可把作者的
// like_count / hermix_reputation 反复刷成负数。
func TestTopicUnLike_IdempotentGuardsAgainstGriefing(t *testing.T) {
	setupLikeServiceTestDB(t)
	topic := mustCreateTopic(t, 100)
	const liker = int64(200)

	// 点赞：+1
	if err := UserLikeService.TopicLike(liker, topic.Id); err != nil {
		t.Fatalf("like: %v", err)
	}
	if got := topicLikeCount(t, topic.Id); got != 1 {
		t.Fatalf("after like like_count=%d want 1", got)
	}

	// 首次取消赞：-1 回到 0
	if err := UserLikeService.TopicUnLike(liker, topic.Id); err != nil {
		t.Fatalf("first unlike: %v", err)
	}
	if got := topicLikeCount(t, topic.Id); got != 0 {
		t.Fatalf("after unlike like_count=%d want 0", got)
	}

	// 重复取消赞：必须报错，like_count 不得再降
	for i := 0; i < 3; i++ {
		if err := UserLikeService.TopicUnLike(liker, topic.Id); err == nil {
			t.Fatalf("repeat unlike #%d should error", i+1)
		}
		if got := topicLikeCount(t, topic.Id); got != 0 {
			t.Fatalf("repeat unlike #%d drove like_count to %d, want 0 (griefing regression)", i+1, got)
		}
	}
}

func TestTopicLike_RejectsDoubleLike(t *testing.T) {
	setupLikeServiceTestDB(t)
	topic := mustCreateTopic(t, 100)
	const liker = int64(200)

	if err := UserLikeService.TopicLike(liker, topic.Id); err != nil {
		t.Fatalf("first like: %v", err)
	}
	if err := UserLikeService.TopicLike(liker, topic.Id); err == nil {
		t.Fatal("double like should be rejected")
	}
	// like_count 不得因重复点赞被刷高
	if got := topicLikeCount(t, topic.Id); got != 1 {
		t.Fatalf("like_count=%d want 1 after rejected double-like", got)
	}
}
