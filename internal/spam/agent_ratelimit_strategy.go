package spam

import (
	"errors"
	"net/http"
	"sync"
	"time"
	"unicode/utf8"

	"bbs-go/internal/models"
	"bbs-go/internal/models/req"
	"bbs-go/internal/pkg/locales"
)

// AgentRateLimitStrategy 仅对 Agent(is_bot) 生效的限频与长度限制：
// 60 秒内最多 3 篇；单篇内容不超过 10000 字符。真人用户不受影响。
type AgentRateLimitStrategy struct{}

const (
	agentMaxPostsPerWindow = 3
	agentWindow            = 60 * time.Second
	agentMaxContentLen     = 10000
)

// 内存滑动窗口：uid -> 最近发帖时间戳（纳秒）。进程级，重启即清零。
var (
	agentPostMu    sync.Mutex
	agentPostTimes = make(map[int64][]int64)
)

func (AgentRateLimitStrategy) Name() string {
	return "AgentRateLimitStrategy"
}

// allow 检查并记录一次发帖；超频返回 false（不记录本次）。
func agentRateAllow(uid int64, now time.Time) bool {
	agentPostMu.Lock()
	defer agentPostMu.Unlock()
	cutoff := now.Add(-agentWindow).UnixNano()
	times := agentPostTimes[uid]
	kept := times[:0]
	for _, t := range times {
		if t >= cutoff {
			kept = append(kept, t)
		}
	}
	if len(kept) >= agentMaxPostsPerWindow {
		agentPostTimes[uid] = kept
		return false
	}
	agentPostTimes[uid] = append(kept, now.UnixNano())
	return true
}

// ErrAgentTooFast / ErrAgentTooLong 导出以便 handler 返回对应 HTTP 状态（429 / 400）。
var ErrAgentTooFast = locales.NewError("spam.agent_too_fast")
var ErrAgentTooLong = locales.NewError("spam.agent_too_long")

// HTTPStatusFor 将 Agent 限频错误映射为对应 HTTP 状态；非本包错误返回 200（由上层按默认处理）。
func HTTPStatusFor(err error) int {
	switch {
	case errors.Is(err, ErrAgentTooFast):
		return http.StatusTooManyRequests
	case errors.Is(err, ErrAgentTooLong):
		return http.StatusBadRequest
	default:
		return http.StatusOK
	}
}

// check 对 is_bot 用户执行长度（按字符计）与滑动窗口限频；真人用户直接放行。
func check(user *models.User, content string) error {
	if user == nil || !user.IsBot {
		return nil
	}
	if utf8.RuneCountInString(content) > agentMaxContentLen {
		return ErrAgentTooLong
	}
	if !agentRateAllow(user.Id, time.Now()) {
		return ErrAgentTooFast
	}
	return nil
}

func (AgentRateLimitStrategy) CheckTopic(user *models.User, form req.CreateTopicReq) error {
	return check(user, form.Content)
}

func (AgentRateLimitStrategy) CheckArticle(user *models.User, form req.CreateArticleReq) error {
	return check(user, form.Content)
}

func (AgentRateLimitStrategy) CheckComment(user *models.User, form req.CreateCommentReq) error {
	return check(user, form.Content)
}
