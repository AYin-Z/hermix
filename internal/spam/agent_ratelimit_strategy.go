package spam

import (
	"errors"
	"sync"
	"time"

	"bbs-go/internal/models"
	"bbs-go/internal/models/req"
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
var ErrAgentTooFast = errors.New("Agent 发帖过于频繁（限 3 篇/60 秒），请稍后再试")
var ErrAgentTooLong = errors.New("内容超出长度上限（10000 字符）")

func (AgentRateLimitStrategy) CheckTopic(user *models.User, form req.CreateTopicReq) error {
	if user == nil || !user.IsBot {
		return nil
	}
	if len(form.Content) > agentMaxContentLen {
		return ErrAgentTooLong
	}
	if !agentRateAllow(user.Id, time.Now()) {
		return ErrAgentTooFast
	}
	return nil
}

func (AgentRateLimitStrategy) CheckArticle(user *models.User, form req.CreateArticleReq) error {
	if user == nil || !user.IsBot {
		return nil
	}
	if len(form.Content) > agentMaxContentLen {
		return ErrAgentTooLong
	}
	if !agentRateAllow(user.Id, time.Now()) {
		return ErrAgentTooFast
	}
	return nil
}

func (AgentRateLimitStrategy) CheckComment(user *models.User, form req.CreateCommentReq) error {
	if user == nil || !user.IsBot {
		return nil
	}
	if len(form.Content) > agentMaxContentLen {
		return ErrAgentTooLong
	}
	if !agentRateAllow(user.Id, time.Now()) {
		return ErrAgentTooFast
	}
	return nil
}
