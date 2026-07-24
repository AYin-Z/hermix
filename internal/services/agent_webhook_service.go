package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"bbs-go/internal/cache"
	"bbs-go/internal/models/constants"
	"bbs-go/internal/repositories"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
)

var AgentWebhookService = newAgentWebhookService()

func newAgentWebhookService() *agentWebhookService {
	return &agentWebhookService{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

type agentWebhookService struct {
	client *http.Client
}

// WebhookPayload 发送给 Agent 回调地址的载荷
type WebhookPayload struct {
	Event      string `json:"event"`      // 事件类型，如 comment.reply / topic.mention
	AgentId    int64  `json:"agentId"`    // 接收此通知的 Agent
	FromUserId int64  `json:"fromUserId"` // 触发者
	EntityType string `json:"entityType"` // topic / article / comment
	EntityId   int64  `json:"entityId"`   // 实体 ID
	CommentId  int64  `json:"commentId"`  // 相关评论 ID
	Content    string `json:"content"`    // 内容摘要
	Timestamp  int64  `json:"timestamp"`  // 发送时间戳
}

// Notify 向指定 Agent 异步发送 webhook 通知。
// 若该用户不是 Agent 或未配置 webhook URL，则静默跳过。
func (s *agentWebhookService) Notify(agentId int64, payload *WebhookPayload) {
	agent := cache.UserCache.Get(agentId)
	if agent == nil || !agent.IsBot || agent.Status != constants.StatusOk {
		return
	}
	url := agent.HermixWebhook
	if url == "" {
		return
	}
	payload.AgentId = agentId
	payload.Timestamp = dates.NowTimestamp()
	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("webhook marshal failed", slog.Any("err", err))
		return
	}
	secret := agent.HermixWebhookSecret
	// 异步投递，带重试，不阻塞事件处理协程
	go s.deliver(url, secret, body, payload.Event)
}

// deliver 执行实际投递，最多重试 3 次（指数退避）。
func (s *agentWebhookService) deliver(url, secret string, body []byte, eventName string) {
	sig := signPayload(secret, body)
	backoff := []time.Duration{0, 2 * time.Second, 5 * time.Second}
	for attempt, wait := range backoff {
		if wait > 0 {
			time.Sleep(wait)
		}
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			slog.Error("webhook build request failed", slog.Any("err", err))
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Hermix-Webhook/1.0")
		req.Header.Set("X-Hermix-Event", eventName)
		req.Header.Set("X-Hermix-Signature", "sha256="+sig)
		resp, err := s.client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return // 成功
			}
			slog.Warn("webhook non-2xx", slog.Int("status", resp.StatusCode), slog.Int("attempt", attempt+1))
		} else {
			slog.Warn("webhook delivery error", slog.Any("err", err), slog.Int("attempt", attempt+1))
		}
	}
	slog.Error("webhook delivery gave up after retries", slog.String("url", url))
}

// signPayload 用 secret 对 body 做 HMAC-SHA256 签名，返回 hex。secret 为空则返回空串。
func signPayload(secret string, body []byte) string {
	if secret == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// SetWebhook 由 owner 设置某个 Agent 的 webhook URL，并生成签名密钥。
func (s *agentWebhookService) SetWebhook(agentId int64, url string) (string, error) {
	secret := newWebhookSecret()
	err := repositories.UserRepository.Updates(sqls.DB(), agentId, map[string]interface{}{
		"hermix_webhook":        url,
		"hermix_webhook_secret": secret,
	})
	if err != nil {
		return "", err
	}
	cache.UserCache.Invalidate(agentId)
	return secret, nil
}

// newWebhookSecret 生成一个随机 32 字节 hex 密钥。
func newWebhookSecret() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
