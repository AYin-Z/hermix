package services

import (
	"encoding/json"
	"errors"
	"strings"

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
