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
	"bbs-go/internal/pkg/locales"
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
		return nil, errors.New(locales.Get("skill.name_required"))
	}
	if len(name) > 128 {
		return nil, errors.New(locales.Get("skill.name_too_long"))
	}
	description := strings.TrimSpace(in.Description)
	if len(description) > 5000 {
		return nil, errors.New(locales.Get("skill.description_too_long"))
	}
	installCommand := strings.TrimSpace(in.InstallCommand)
	if len(installCommand) > 1000 {
		return nil, errors.New(locales.Get("skill.install_command_too_long"))
	}
	// 规整标签：去空、去重、限制数量与单个长度
	tags := normalizeTags(in.Tags)
	tagsJSON := "[]"
	if len(tags) > 0 {
		if b, err := json.Marshal(tags); err == nil {
			tagsJSON = string(b)
		}
	}
	now := dates.NowTimestamp()
	skill := &models.HermixSkill{
		AuthorId:       authorId,
		Name:           name,
		Description:    description,
		InstallCommand: installCommand,
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

// ErrAlreadyRated 表示该用户已对此技能评过分（含并发命中唯一索引的情况）。
var ErrAlreadyRated = locales.NewError("skill.already_rated")

// Rate 给技能评分（1-5），每个用户每个技能仅记一次，防重复刷分。
// 评分记录与聚合累加放在同一事务，避免崩溃导致计数漂移；
// 并发下依赖 (skill_id,user_id) 唯一索引兜底，重复插入映射为 ErrAlreadyRated。
func (s *hermixSkillService) Rate(skillId, userId int64, score int) error {
	if score < 1 || score > 5 {
		return errors.New(locales.Get("skill.score_out_of_range"))
	}
	skill := repositories.HermixSkillRepository.Get(sqls.DB(), skillId)
	if skill == nil || skill.Status != constants.StatusOk {
		return errors.New(locales.Get("skill.not_found"))
	}
	if skill.AuthorId == userId {
		return errors.New(locales.Get("skill.self_rating_forbidden"))
	}
	if existing := repositories.HermixSkillRatingRepository.FindBySkillAndUser(sqls.DB(), skillId, userId); existing != nil {
		return ErrAlreadyRated
	}
	return sqls.DB().Transaction(func(tx *gorm.DB) error {
		if err := repositories.HermixSkillRatingRepository.Create(tx, &models.HermixSkillRating{
			SkillId:    skillId,
			UserId:     userId,
			Score:      score,
			CreateTime: dates.NowTimestamp(),
		}); err != nil {
			if isDuplicateKeyErr(err) {
				return ErrAlreadyRated
			}
			return err
		}
		return repositories.HermixSkillRepository.Updates(tx, skillId, map[string]interface{}{
			"rating_sum":   gorm.Expr("rating_sum + ?", score),
			"rating_count": gorm.Expr("rating_count + 1"),
		})
	})
}

// IncrInstall 记录一次安装。
func (s *hermixSkillService) IncrInstall(skillId int64) error {
	return repositories.HermixSkillRepository.Updates(sqls.DB(), skillId, map[string]interface{}{
		"install_count": gorm.Expr("install_count + 1"),
	})
}

// normalizeTags 去空白、去重、限制标签数量(≤10)与单标签长度(≤32)。
func normalizeTags(in []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(in))
	for _, t := range in {
		t = strings.TrimSpace(t)
		if t == "" || len(t) > 32 {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
		if len(out) >= 10 {
			break
		}
	}
	return out
}

// isDuplicateKeyErr 判断是否为唯一索引冲突（跨 MySQL/Postgres）。
func isDuplicateKeyErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "1062")
}
