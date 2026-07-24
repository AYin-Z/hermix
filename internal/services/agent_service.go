package services

import (
	"encoding/json"
	"errors"
	"strings"

	"bbs-go/internal/cache"
	"bbs-go/internal/models"
	"bbs-go/internal/models/constants"
	"bbs-go/internal/pkg/search"
	"bbs-go/internal/repositories"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/common/passwd"
	"github.com/mlogclub/simple/common/strs"
	"github.com/mlogclub/simple/sqls"
)

var AgentService = newAgentService()

func newAgentService() *agentService {
	return &agentService{}
}

type agentService struct{}

// RegisterAgent 由 owner（真人用户）注册一个 AI Agent 账号。
// 返回新建的 Agent 用户；调用方负责为其签发 token。
func (s *agentService) RegisterAgent(ownerId int64, username, nickname, botModel string, capabilities []string) (*models.User, error) {
	username = strings.TrimSpace(username)
	nickname = strings.TrimSpace(nickname)
	botModel = strings.TrimSpace(botModel)

	if ownerId <= 0 {
		return nil, errors.New("必须由登录用户注册 Agent")
	}
	if len(nickname) == 0 {
		return nil, errors.New("Agent 昵称不能为空")
	}
	if len(username) == 0 {
		return nil, errors.New("Agent 用户名不能为空")
	}
	if UserService.GetByUsername(username) != nil {
		return nil, errors.New("用户名已被占用：" + username)
	}

	capJSON := "[]"
	if len(capabilities) > 0 {
		if b, err := json.Marshal(capabilities); err == nil {
			capJSON = string(b)
		}
	}

	now := dates.NowTimestamp()
	agent := &models.User{
		Username:           sqls.SqlNullString(username),
		Nickname:           nickname,
		Password:           passwd.EncodePassword(strs.UUID()), // 随机密码，Agent 用 token 登录
		Status:             constants.StatusOk,
		Roles:              "user",
		IsBot:              true,
		BotOwner:           ownerId,
		BotModel:           botModel,
		HermixReputation:   0,
		HermixCapabilities: capJSON,
		CreateTime:         now,
		UpdateTime:         now,
	}
	if err := repositories.UserRepository.Create(sqls.DB(), agent); err != nil {
		return nil, err
	}
	search.UpdateUserIndex(agent)
	return agent, nil
}

// ListByOwner 返回某个 owner 拥有的全部 Agent（按创建时间倒序）。
func (s *agentService) ListByOwner(ownerId int64) []models.User {
	if ownerId <= 0 {
		return nil
	}
	return repositories.UserRepository.Find(sqls.DB(), sqls.NewCnd().
		Eq("is_bot", true).
		Eq("bot_owner", ownerId).
		Desc("id"))
}

// DiscoverAgents 公开发现 Agent。capability 非空时按能力标签过滤（JSON 数组 LIKE 匹配）。
// 仅返回状态正常的 Agent，按信誉分倒序。limit<=0 时默认 50，上限 200。
func (s *agentService) DiscoverAgents(capability string, limit int) []models.User {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	cnd := sqls.NewCnd().
		Eq("is_bot", true).
		Eq("status", constants.StatusOk)
	capability = strings.TrimSpace(capability)
	if capability != "" {
		// hermix_capabilities 存的是 JSON 数组字符串，如 ["code-review","qa"]
		cnd.Like("hermix_capabilities", "%\""+capability+"\"%")
	}
	cnd.Desc("hermix_reputation").Limit(limit)
	return repositories.UserRepository.Find(sqls.DB(), cnd)
}

// GetAgent 按 id 返回 Agent（仅当确为 bot 且状态正常）。
func (s *agentService) GetAgent(id int64) *models.User {
	if id <= 0 {
		return nil
	}
	u := cache.UserCache.Get(id)
	if u == nil || !u.IsBot || u.Status != constants.StatusOk {
		return nil
	}
	return u
}

// AdjustReputation 调整 Agent 信誉分（仅对 is_bot 用户生效）。非 Agent 用户静默跳过。
func (s *agentService) AdjustReputation(userId int64, delta int) {
	u := cache.UserCache.Get(userId)
	if u == nil || !u.IsBot {
		return
	}
	if err := repositories.UserRepository.AdjustReputation(sqls.DB(), userId, delta); err != nil {
		return
	}
	cache.UserCache.Invalidate(userId)
}
