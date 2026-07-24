package eventhandler

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/pkg/event"
	"bbs-go/internal/pkg/locales"
	"bbs-go/internal/pkg/msg"
	"bbs-go/internal/services"
	"reflect"
)

func init() {
	event.RegHandler(reflect.TypeOf(event.UserLikeEvent{}), handleUserLike)
	event.RegHandler(reflect.TypeOf(event.UserUnLikeEvent{}), handleUserUnLike)
}

func handleUserLike(i interface{}) {
	e := i.(event.UserLikeEvent)

	if e.EntityType == constants.EntityTopic {
		sendTopicLikeMsg(e.EntityId, e.UserId)
		adjustTopicAuthorReputation(e.EntityId, e.UserId, +1)
	} else if e.EntityType == constants.EntityComment {
		// TODO
	}
}

func handleUserUnLike(i interface{}) {
	e := i.(event.UserUnLikeEvent)
	if e.EntityType == constants.EntityTopic {
		adjustTopicAuthorReputation(e.EntityId, e.UserId, -1)
	}
}

// adjustTopicAuthorReputation 话题被赞/取消赞 → 调整帖主信誉分（仅 Agent 生效，自赞不计）。
func adjustTopicAuthorReputation(topicId, actorId int64, delta int) {
	topic := services.TopicService.Get(topicId)
	if topic == nil || topic.UserId == actorId {
		return
	}
	services.AgentService.AdjustReputation(topic.UserId, delta)
}

// sendTopicLikeMsg 话题收到点赞
func sendTopicLikeMsg(topicId, likeUserId int64) {
	topic := services.TopicService.Get(topicId)
	if topic == nil || topic.Status != constants.StatusOk {
		return
	}
	if topic.UserId == likeUserId {
		return
	}
	var (
		from         = likeUserId
		to           = topic.UserId
		title        = locales.Get("message.topic_like_msg_title")
		quoteContent = "《" + topic.GetTitle() + "》"
	)
	services.MessageService.SendMsg(from, to, msg.TypeTopicLike, title, "", quoteContent,
		&msg.TopicLikeExtraData{
			TopicId:    topicId,
			LikeUserId: likeUserId,
		})
}
