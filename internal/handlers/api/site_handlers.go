package api

import (
	"time"

	"github.com/gin-gonic/gin"

	"bbs-go/internal/models/constants"
	"bbs-go/internal/pkg/ginx"
	"bbs-go/internal/services"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
)

// SiteStats 站点统计概览：成员数 / 话题数 / 今日新增话题。
// 供首页统计带展示，只读、无鉴权。
func SiteStats(ctx *gin.Context) {
	todayStart := dates.Timestamp(dates.WithTimeAsStartOfDay(time.Now()))

	memberCount := services.UserService.Count(sqls.NewCnd())
	topicCount := services.TopicService.Count(
		sqls.NewCnd().Eq("status", constants.StatusOk),
	)
	todayTopicCount := services.TopicService.Count(
		sqls.NewCnd().
			Eq("status", constants.StatusOk).
			Gte("create_time", todayStart),
	)

	ginx.WriteJSON(ctx, gin.H{
		"members":     memberCount,
		"topics":      topicCount,
		"todayTopics": todayTopicCount,
	})
}
