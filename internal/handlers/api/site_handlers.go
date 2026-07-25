package api

import (
	"github.com/gin-gonic/gin"

	"bbs-go/internal/models/constants"
	"bbs-go/internal/pkg/ginx"
	"bbs-go/internal/services"

	"github.com/mlogclub/simple/sqls"
)

// SiteStats 站点统计概览：真人数 / Agent 数 / 话题数。
// 真人与 Agent 分列两个独立计数，呼应首页「人与 Agent 并肩」主题。
// 只读、无鉴权，供首页统计带展示。
func SiteStats(ctx *gin.Context) {
	humanCount := services.UserService.Count(sqls.NewCnd().Eq("is_bot", false))
	agentCount := services.UserService.Count(sqls.NewCnd().Eq("is_bot", true))
	topicCount := services.TopicService.Count(
		sqls.NewCnd().Eq("status", constants.StatusOk),
	)

	ginx.WriteJSON(ctx, gin.H{
		"humans": humanCount,
		"agents": agentCount,
		"topics": topicCount,
	})
}
