package api

import (
	"bbs-go/internal/cache"
	"bbs-go/internal/handlers/render"
	"bbs-go/internal/models/constants"
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/pkg/errs"
	"bbs-go/internal/pkg/ginx"
	"bbs-go/internal/pkg/idcodec"
	"bbs-go/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/mlogclub/simple/web"
	"github.com/spf13/cast"
)

// agentRegisterReq Agent 注册请求
type agentRegisterReq struct {
	Username     string   `json:"username" form:"username"`
	Nickname     string   `json:"nickname" form:"nickname"`
	BotModel     string   `json:"botModel" form:"botModel"`
	Capabilities []string `json:"capabilities" form:"capabilities"`
}

// AgentRegister 由登录的真人用户（owner）注册一个 AI Agent，返回 Agent 的访问 token。
// POST /api/agent/register  (需 owner 登录)
func AgentRegister(ctx *gin.Context) {
	owner := common.GetCurrentUser(ctx)
	if owner == nil {
		ginx.WriteJSON(ctx, errs.NotLogin())
		return
	}
	if owner.IsBot {
		ginx.WriteJSON(ctx, ginx.ErrorMessage("Agent 不能再注册子 Agent"))
		return
	}

	var r agentRegisterReq
	if err := ginx.Bind(ctx, &r); err != nil {
		ginx.WriteJSON(ctx, err)
		return
	}

	agent, err := services.AgentService.RegisterAgent(owner.Id, r.Username, r.Nickname, r.BotModel, r.Capabilities)
	if err != nil {
		ginx.WriteJSON(ctx, ginx.ErrorMessage(err.Error()))
		return
	}

	token, err := services.UserTokenService.Generate(agent.Id)
	if err != nil {
		ginx.WriteJSON(ctx, web.JsonError(err))
		return
	}

	ginx.WriteJSON(ctx, web.NewEmptyRspBuilder().
		Put("token", token).
		Put("agentId", idcodec.Encode(agent.Id)).
		Put("agent", render.BuildUserDetail(agent)).
		JsonResult())
}

// AgentList 列出当前登录用户拥有的 Agent。
// GET /api/agent/list  (需 owner 登录)
func AgentList(ctx *gin.Context) {
	owner := common.GetCurrentUser(ctx)
	if owner == nil {
		ginx.WriteJSON(ctx, errs.NotLogin())
		return
	}
	agents := services.AgentService.ListByOwner(owner.Id)
	items := make([]interface{}, 0, len(agents))
	for i := range agents {
		if agents[i].Status == constants.StatusDeleted {
			continue
		}
		items = append(items, render.BuildUserDetail(&agents[i]))
	}
	ginx.WriteJSON(ctx, web.NewEmptyRspBuilder().Put("agents", items).JsonResult())
}

// AgentDiscover 公开发现 Agent，可按能力标签过滤。
// GET /api/agent/discover?capability=code-review&limit=50  (公开)
func AgentDiscover(ctx *gin.Context) {
	capability := ctx.Query("capability")
	limit := cast.ToInt(ctx.Query("limit"))
	agents := services.AgentService.DiscoverAgents(capability, limit)
	items := make([]interface{}, 0, len(agents))
	for i := range agents {
		items = append(items, render.BuildUserDetail(&agents[i]))
	}
	ginx.WriteJSON(ctx, web.NewEmptyRspBuilder().
		Put("agents", items).
		Put("total", len(items)).
		JsonResult())
}

// AgentCapabilities 返回单个 Agent 的能力详情。
// GET /api/agent/capabilities/:id  (公开)
func AgentCapabilities(ctx *gin.Context) {
	agentId := idcodec.Decode(ctx.Param("id"))
	agent := services.AgentService.GetAgent(agentId)
	if agent == nil {
		ginx.WriteJSON(ctx, ginx.ErrorMessage("Agent 不存在"))
		return
	}
	ginx.WriteJSON(ctx, render.BuildUserDetail(agent))
}

// AgentRegenerateToken 为指定 Agent 重新签发 token（仅 owner 可操作）。
// POST /api/agent/regenerate_token/:id
func AgentRegenerateToken(ctx *gin.Context) {
	owner := common.GetCurrentUser(ctx)
	if owner == nil {
		ginx.WriteJSON(ctx, errs.NotLogin())
		return
	}
	agentId := idcodec.Decode(ctx.Param("id"))
	agent := cache.UserCache.Get(agentId)
	if agent == nil || !agent.IsBot || agent.BotOwner != owner.Id {
		ginx.WriteJSON(ctx, ginx.ErrorMessage("Agent 不存在或无权限"))
		return
	}
	token, err := services.UserTokenService.Generate(agent.Id)
	if err != nil {
		ginx.WriteJSON(ctx, web.JsonError(err))
		return
	}
	ginx.WriteJSON(ctx, web.NewEmptyRspBuilder().Put("token", token).JsonResult())
}
