package services

import (
	"context"
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"bbs-go/internal/cache"
	"bbs-go/internal/models/constants"
	"bbs-go/internal/repositories"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
)

var AgentWebhookService = newAgentWebhookService()

func newAgentWebhookService() *agentWebhookService {
	// 自定义 dialer：在实际建立 TCP 连接前校验目标 IP，防御 DNS rebinding
	// （set 时解析到公网、投递时被重绑到内网的情况）。
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	safeDial := func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		// 自行解析并校验，再直接连到验证过的 IP —— 关闭 DNS rebinding 缺口
		ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
		if err != nil || len(ips) == 0 {
			return nil, errors.New("无法解析 webhook 主机名")
		}
		for _, ip := range ips {
			if !isPublicIP(ip) {
				return nil, errors.New("webhook 目标解析到非公网地址，已拒绝")
			}
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
	}
	return &agentWebhookService{
		client: &http.Client{
			Timeout:   10 * time.Second,
			Transport: &http.Transport{DialContext: safeDial},
		},
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
// 允许清空（url==""）以停用回调；否则做 SSRF 校验。
func (s *agentWebhookService) SetWebhook(agentId int64, webhookUrl string) (string, error) {
	webhookUrl = strings.TrimSpace(webhookUrl)
	if webhookUrl != "" {
		if err := ValidateWebhookURL(webhookUrl); err != nil {
			return "", err
		}
	}
	secret := newWebhookSecret()
	err := repositories.UserRepository.Updates(sqls.DB(), agentId, map[string]interface{}{
		"hermix_webhook":        webhookUrl,
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

// ValidateWebhookURL 对 webhook URL 做 SSRF 防护：
// 仅允许 http/https，拒绝 loopback / 私网 / 链路本地 / 未指定 / 多播地址，
// 且必须能解析到至少一个公网 IP。
func ValidateWebhookURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return errors.New("非法的 webhook URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("webhook URL 必须以 http:// 或 https:// 开头")
	}
	host := u.Hostname()
	if host == "" {
		return errors.New("webhook URL 缺少主机名")
	}
	// 解析主机名到 IP（域名可能解析到内网，一并校验）
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return errors.New("无法解析 webhook 主机名")
	}
	for _, ip := range ips {
		if !isPublicIP(ip) {
			return errors.New("webhook URL 不允许指向内网 / 本地地址")
		}
	}
	return nil
}

// isPublicIP 判断 IP 是否为可对外访问的公网地址。
func isPublicIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return false
	}
	// 显式拒绝云元数据地址 169.254.169.254（已被 LinkLocal 覆盖，双保险）
	if ip.Equal(net.IPv4(169, 254, 169, 254)) {
		return false
	}
	return true
}
