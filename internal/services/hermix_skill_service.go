package services

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"

	"bbs-go/internal/models"
	"bbs-go/internal/models/constants"
	"bbs-go/internal/repositories"
)

var HermixSkillService = newHermixSkillService()

func newHermixSkillService() *hermixSkillService {
	return &hermixSkillService{}
}

type hermixSkillService struct{}

// SkillPublishInput 发布技能入参
type SkillPublishInput struct {
	Name           string
	Description    string
	InstallCommand string
	Tags           []string
}

func (s *hermixSkillService) Get(id int64) *models.HermixSkill {
	return repositories.HermixSkillRepository.Get(sqls.DB(), id)
}

// Publish 发布一个新技能。
func (s *hermixSkillService) Publish(authorId int64, in SkillPublishInput) (*models.HermixSkill, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, errors.New("技能名称不能为空")
	}
	if len(name) > 128 {
		return nil, errors.New("技能名称过长")
	}
	tagsJSON := "[]"
	if len(in.Tags) > 0 {
		if b, err := json.Marshal(in.Tags); err == nil {
			tagsJSON = string(b)
		}
	}
	now := dates.NowTimestamp()
	skill := &models.HermixSkill{
		AuthorId:       authorId,
		Name:           name,
		Description:    strings.TrimSpace(in.Description),
		InstallCommand: strings.TrimSpace(in.InstallCommand),
		Tags:           tagsJSON,
		Status:         constants.StatusOk,
		CreateTime:     now,
		UpdateTime:     now,
	}
	if err := repositories.HermixSkillRepository.Create(sqls.DB(), skill); err != nil {
		return nil, err
	}
	return skill, nil
}

// List 列出技能，可选按 tag 过滤、关键词搜索，按安装量/评分排序。
func (s *hermixSkillService) List(tag, keyword string, page, limit int) ([]models.HermixSkill, *sqls.Paging) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	cnd := sqls.NewCnd().Eq("status", constants.StatusOk)
	if tag = strings.TrimSpace(tag); tag != "" {
		cnd.Like("tags", "%\""+tag+"\"%")
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		cnd.Like("name", "%"+keyword+"%")
	}
	cnd.Desc("install_count").Desc("id").Page(page, limit)
	return repositories.HermixSkillRepository.FindPageByCnd(sqls.DB(), cnd)
}

// Rate 给技能评分（1-5），每个用户每个技能仅记一次，防重复刷分。
func (s *hermixSkillService) Rate(skillId, userId int64, score int) error {
	if score < 1 || score > 5 {
		return errors.New("评分必须在 1-5 之间")
	}
	skill := repositories.HermixSkillRepository.Get(sqls.DB(), skillId)
	if skill == nil || skill.Status != constants.StatusOk {
		return errors.New("技能不存在")
	}
	if existing := repositories.HermixSkillRatingRepository.FindBySkillAndUser(sqls.DB(), skillId, userId); existing != nil {
		return errors.New("你已经评过分了")
	}
	err := repositories.HermixSkillRatingRepository.Create(sqls.DB(), &models.HermixSkillRating{
		SkillId:    skillId,
		UserId:     userId,
		Score:      score,
		CreateTime: dates.NowTimestamp(),
	})
	if err != nil {
		return err
	}
	// 原子累加评分总和/人数
	return repositories.HermixSkillRepository.Updates(sqls.DB(), skillId, map[string]interface{}{
		"rating_sum":   gorm.Expr("rating_sum + ?", score),
		"rating_count": gorm.Expr("rating_count + 1"),
	})
}

// IncrInstall 记录一次安装。
func (s *hermixSkillService) IncrInstall(skillId int64) error {
	return repositories.HermixSkillRepository.Updates(sqls.DB(), skillId, map[string]interface{}{
		"install_count": gorm.Expr("install_count + 1"),
	})
}
