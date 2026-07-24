package services

import (
	"path/filepath"
	"testing"

	"bbs-go/internal/models"
	"bbs-go/internal/models/constants"
	"bbs-go/internal/pkg/config"
	"bbs-go/internal/pkg/search"
	"bbs-go/internal/repositories"

	"github.com/mlogclub/simple/sqls"
)

// setupAgentServiceTestDB 建 in-memory 库并初始化 search 索引，
// 因为 RegisterAgent 成功路径会调用 search.UpdateUserIndex。
func setupAgentServiceTestDB(t *testing.T) {
	t.Helper()
	config.Instance = &config.Config{Language: config.DefaultLanguage}
	config.Instance.Search.IndexPath = filepath.Join(t.TempDir(), "idx")
	search.Init()
	db := setupTestDB(t)
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("auto migrate user: %v", err)
	}
}

func TestRegisterAgent_ValidationRejects(t *testing.T) {
	setupAgentServiceTestDB(t)

	cases := []struct {
		name     string
		ownerId  int64
		username string
		nickname string
	}{
		{"no owner", 0, "bot1", "Bot One"},
		{"blank nickname", 1, "bot1", "  "},
		{"blank username", 1, "  ", "Bot One"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := AgentService.RegisterAgent(c.ownerId, c.username, c.nickname, "gpt", nil); err == nil {
				t.Fatalf("expected error for %s", c.name)
			}
		})
	}
}

func TestRegisterAgent_DuplicateUsername(t *testing.T) {
	setupAgentServiceTestDB(t)

	if _, err := AgentService.RegisterAgent(1, "dupbot", "First", "gpt", nil); err != nil {
		t.Fatalf("first register: %v", err)
	}
	if _, err := AgentService.RegisterAgent(1, "dupbot", "Second", "gpt", nil); err == nil {
		t.Fatal("expected duplicate username to be rejected")
	}
}

func TestRegisterAgent_PersistsBotFields(t *testing.T) {
	setupAgentServiceTestDB(t)

	agent, err := AgentService.RegisterAgent(7, "capbot", "Cap Bot", "claude", []string{"code-review", "qa"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if !agent.IsBot {
		t.Error("agent should have IsBot=true")
	}
	if agent.BotOwner != 7 {
		t.Errorf("BotOwner=%d want 7", agent.BotOwner)
	}
	if agent.BotModel != "claude" {
		t.Errorf("BotModel=%q want claude", agent.BotModel)
	}
	if agent.HermixReputation != 0 {
		t.Errorf("new agent reputation=%d want 0", agent.HermixReputation)
	}
	if agent.HermixCapabilities != `["code-review","qa"]` {
		t.Errorf("capabilities=%q want JSON array", agent.HermixCapabilities)
	}
	// 随机密码不落空
	if agent.Password == "" {
		t.Error("agent password should not be empty")
	}
}

func TestListByOwner_FiltersByOwnerAndBot(t *testing.T) {
	setupAgentServiceTestDB(t)

	if _, err := AgentService.RegisterAgent(1, "o1a", "Owner1 A", "gpt", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := AgentService.RegisterAgent(1, "o1b", "Owner1 B", "gpt", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := AgentService.RegisterAgent(2, "o2a", "Owner2 A", "gpt", nil); err != nil {
		t.Fatal(err)
	}

	got := AgentService.ListByOwner(1)
	if len(got) != 2 {
		t.Fatalf("owner 1 agents=%d want 2", len(got))
	}
	if AgentService.ListByOwner(0) != nil {
		t.Error("ListByOwner(0) should return nil")
	}
}

func TestDiscoverAgents_CapabilityFilterAndReputationSort(t *testing.T) {
	setupAgentServiceTestDB(t)

	a, _ := AgentService.RegisterAgent(1, "rev1", "Reviewer 1", "gpt", []string{"code-review"})
	b, _ := AgentService.RegisterAgent(1, "rev2", "Reviewer 2", "gpt", []string{"code-review"})
	if _, err := AgentService.RegisterAgent(1, "qaonly", "QA Only", "gpt", []string{"qa"}); err != nil {
		t.Fatal(err)
	}
	// 直接改库设不同信誉，验证排序（不经缓存，避免 cache 依赖）
	if err := repositories.UserRepository.AdjustReputation(sqls.DB(), a.Id, 5); err != nil {
		t.Fatal(err)
	}
	if err := repositories.UserRepository.AdjustReputation(sqls.DB(), b.Id, 20); err != nil {
		t.Fatal(err)
	}

	got := AgentService.DiscoverAgents("code-review", 0)
	if len(got) != 2 {
		t.Fatalf("code-review agents=%d want 2", len(got))
	}
	// 按信誉倒序：b(20) 在 a(5) 前
	if got[0].Id != b.Id {
		t.Errorf("first result id=%d want %d (higher reputation)", got[0].Id, b.Id)
	}
	// 无能力过滤时返回全部 3 个
	if all := AgentService.DiscoverAgents("", 0); len(all) != 3 {
		t.Errorf("all agents=%d want 3", len(all))
	}
}

func TestDiscoverAgents_ExcludesDeleted(t *testing.T) {
	setupAgentServiceTestDB(t)

	a, _ := AgentService.RegisterAgent(1, "live", "Live", "gpt", []string{"x"})
	del, _ := AgentService.RegisterAgent(1, "dead", "Dead", "gpt", []string{"x"})
	if err := sqls.DB().Model(&models.User{}).Where("id = ?", del.Id).
		UpdateColumn("status", constants.StatusDeleted).Error; err != nil {
		t.Fatal(err)
	}

	got := AgentService.DiscoverAgents("x", 0)
	if len(got) != 1 || got[0].Id != a.Id {
		t.Fatalf("discover should exclude deleted agent, got %d results", len(got))
	}
}
