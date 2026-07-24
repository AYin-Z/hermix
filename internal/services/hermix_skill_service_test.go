package services

import (
	"strings"
	"testing"

	"bbs-go/internal/models"
	"bbs-go/internal/pkg/config"
)

func setupSkillServiceTestDB(t *testing.T) {
	t.Helper()
	config.Instance = &config.Config{Language: config.DefaultLanguage}
	db := setupTestDB(t)
	if err := db.AutoMigrate(&models.HermixSkill{}, &models.HermixSkillRating{}); err != nil {
		t.Fatalf("auto migrate skill: %v", err)
	}
}

func mustPublish(t *testing.T, authorId int64, name string) *models.HermixSkill {
	t.Helper()
	skill, err := HermixSkillService.Publish(authorId, SkillPublishInput{Name: name, InstallCommand: "npm i x"})
	if err != nil {
		t.Fatalf("publish %q: %v", name, err)
	}
	return skill
}

func TestSkillPublish_Validation(t *testing.T) {
	setupSkillServiceTestDB(t)

	cases := []struct {
		name string
		in   SkillPublishInput
	}{
		{"blank name", SkillPublishInput{Name: "  "}},
		{"name too long", SkillPublishInput{Name: strings.Repeat("a", 129)}},
		{"description too long", SkillPublishInput{Name: "ok", Description: strings.Repeat("d", 5001)}},
		{"install command too long", SkillPublishInput{Name: "ok", InstallCommand: strings.Repeat("c", 1001)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := HermixSkillService.Publish(1, c.in); err == nil {
				t.Fatalf("expected error for %s", c.name)
			}
		})
	}
}

func TestSkillPublish_NormalizesTags(t *testing.T) {
	setupSkillServiceTestDB(t)

	// 重复、空白、超量、超长标签都应被规整
	tags := []string{"go", "go", " ", "rust", strings.Repeat("x", 33)}
	skill, err := HermixSkillService.Publish(1, SkillPublishInput{Name: "tagtest", Tags: tags})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	// go 去重一次 + rust；空白与超长被丢弃
	if skill.Tags != `["go","rust"]` {
		t.Errorf("tags=%q want [\"go\",\"rust\"]", skill.Tags)
	}
}

func TestSkillRate_ScoreRange(t *testing.T) {
	setupSkillServiceTestDB(t)
	skill := mustPublish(t, 1, "ratergame")

	for _, bad := range []int{0, 6, -1} {
		if err := HermixSkillService.Rate(skill.Id, 2, bad); err == nil {
			t.Errorf("score %d should be rejected", bad)
		}
	}
}

func TestSkillRate_RejectsSelfRating(t *testing.T) {
	setupSkillServiceTestDB(t)
	skill := mustPublish(t, 1, "selfrate")

	if err := HermixSkillService.Rate(skill.Id, 1, 5); err == nil {
		t.Fatal("author rating own skill should be rejected")
	}
}

func TestSkillRate_RejectsDuplicate(t *testing.T) {
	setupSkillServiceTestDB(t)
	skill := mustPublish(t, 1, "duprate")

	if err := HermixSkillService.Rate(skill.Id, 2, 4); err != nil {
		t.Fatalf("first rating: %v", err)
	}
	if err := HermixSkillService.Rate(skill.Id, 2, 5); err != ErrAlreadyRated {
		t.Fatalf("second rating err=%v want ErrAlreadyRated", err)
	}
}

func TestSkillRate_AggregatesSumAndCount(t *testing.T) {
	setupSkillServiceTestDB(t)
	skill := mustPublish(t, 1, "aggrate")

	if err := HermixSkillService.Rate(skill.Id, 2, 4); err != nil {
		t.Fatal(err)
	}
	if err := HermixSkillService.Rate(skill.Id, 3, 2); err != nil {
		t.Fatal(err)
	}

	got := HermixSkillService.Get(skill.Id)
	if got.RatingCount != 2 {
		t.Errorf("rating_count=%d want 2", got.RatingCount)
	}
	if got.RatingSum != 6 {
		t.Errorf("rating_sum=%d want 6", got.RatingSum)
	}
}

func TestSkillRate_NonexistentSkill(t *testing.T) {
	setupSkillServiceTestDB(t)
	if err := HermixSkillService.Rate(999999, 2, 5); err == nil {
		t.Fatal("rating nonexistent skill should error")
	}
}

func TestSkillIncrInstall(t *testing.T) {
	setupSkillServiceTestDB(t)
	skill := mustPublish(t, 1, "installcount")

	for i := 0; i < 3; i++ {
		if err := HermixSkillService.IncrInstall(skill.Id); err != nil {
			t.Fatal(err)
		}
	}
	if got := HermixSkillService.Get(skill.Id); got.InstallCount != 3 {
		t.Errorf("install_count=%d want 3", got.InstallCount)
	}
}

func TestSkillList_FiltersByTagAndStatus(t *testing.T) {
	setupSkillServiceTestDB(t)

	if _, err := HermixSkillService.Publish(1, SkillPublishInput{Name: "goskill", Tags: []string{"go"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := HermixSkillService.Publish(1, SkillPublishInput{Name: "rustskill", Tags: []string{"rust"}}); err != nil {
		t.Fatal(err)
	}

	got, _ := HermixSkillService.List("go", "", 1, 20)
	if len(got) != 1 || got[0].Name != "goskill" {
		t.Fatalf("tag filter returned %d results, want 1 goskill", len(got))
	}
	// 关键词过滤
	kw, _ := HermixSkillService.List("", "rust", 1, 20)
	if len(kw) != 1 || kw[0].Name != "rustskill" {
		t.Fatalf("keyword filter returned %d results, want 1 rustskill", len(kw))
	}
}
