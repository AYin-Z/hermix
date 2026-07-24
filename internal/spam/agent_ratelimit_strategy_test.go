package spam

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"bbs-go/internal/models"
	"bbs-go/internal/models/req"
)

func botUser(id int64) *models.User {
	return &models.User{Model: models.Model{Id: id}, IsBot: true}
}

func humanUser(id int64) *models.User {
	return &models.User{Model: models.Model{Id: id}, IsBot: false}
}

func TestAgentRateLimit_HumanUnaffected(t *testing.T) {
	s := AgentRateLimitStrategy{}
	long := strings.Repeat("x", agentMaxContentLen+100)
	// 真人：超长、连发都不受限
	for i := 0; i < 10; i++ {
		if err := s.CheckTopic(humanUser(1001), req.CreateTopicReq{Content: long}); err != nil {
			t.Fatalf("human should never be limited, got %v", err)
		}
	}
}

func TestAgentRateLimit_NilUserUnaffected(t *testing.T) {
	s := AgentRateLimitStrategy{}
	if err := s.CheckTopic(nil, req.CreateTopicReq{}); err != nil {
		t.Fatalf("nil user should pass, got %v", err)
	}
}

func TestAgentRateLimit_ContentLengthByRune(t *testing.T) {
	s := AgentRateLimitStrategy{}
	// 恰好上限：通过（CJK 每字算 1 rune 而非 3 字节）
	exact := strings.Repeat("中", agentMaxContentLen)
	if err := s.CheckTopic(botUser(2001), req.CreateTopicReq{Content: exact}); err != nil {
		t.Fatalf("content at exact rune limit should pass, got %v", err)
	}
	// 超一个字符：拒绝
	over := strings.Repeat("中", agentMaxContentLen+1)
	if err := s.CheckTopic(botUser(2002), req.CreateTopicReq{Content: over}); err != ErrAgentTooLong {
		t.Fatalf("over-limit content err=%v want ErrAgentTooLong", err)
	}
}

func TestAgentRateAllow_SlidingWindow(t *testing.T) {
	uid := int64(3001)
	base := time.Unix(1_700_000_000, 0)

	// 窗口内前 3 次允许，第 4 次拒绝
	for i := 0; i < agentMaxPostsPerWindow; i++ {
		if !agentRateAllow(uid, base) {
			t.Fatalf("post %d within window should be allowed", i+1)
		}
	}
	if agentRateAllow(uid, base) {
		t.Fatal("post over window limit should be denied")
	}

	// 窗口滑过后重新允许
	later := base.Add(agentWindow + time.Second)
	if !agentRateAllow(uid, later) {
		t.Fatal("post after window elapsed should be allowed again")
	}
}

func TestAgentRateAllow_PerUserIsolation(t *testing.T) {
	base := time.Unix(1_700_001_000, 0)
	// uid A 打满窗口
	for i := 0; i < agentMaxPostsPerWindow; i++ {
		agentRateAllow(4001, base)
	}
	if agentRateAllow(4001, base) {
		t.Fatal("uid A should be limited")
	}
	// uid B 不受 A 影响
	if !agentRateAllow(4002, base) {
		t.Fatal("uid B should be independent of uid A")
	}
}

func TestHTTPStatusFor(t *testing.T) {
	if got := HTTPStatusFor(ErrAgentTooFast); got != http.StatusTooManyRequests {
		t.Errorf("ErrAgentTooFast -> %d want 429", got)
	}
	if got := HTTPStatusFor(ErrAgentTooLong); got != http.StatusBadRequest {
		t.Errorf("ErrAgentTooLong -> %d want 400", got)
	}
	if got := HTTPStatusFor(nil); got != http.StatusOK {
		t.Errorf("nil -> %d want 200", got)
	}
}
