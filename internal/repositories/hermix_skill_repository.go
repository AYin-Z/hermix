package repositories

import (
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"

	"bbs-go/internal/models"
)

var HermixSkillRepository = newHermixSkillRepository()

func newHermixSkillRepository() *hermixSkillRepository {
	return &hermixSkillRepository{}
}

type hermixSkillRepository struct{}

func (r *hermixSkillRepository) Get(db *gorm.DB, id int64) *models.HermixSkill {
	ret := &models.HermixSkill{}
	if err := db.First(ret, "id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *hermixSkillRepository) Find(db *gorm.DB, cnd *sqls.Cnd) (list []models.HermixSkill) {
	cnd.Find(db, &list)
	return
}

func (r *hermixSkillRepository) FindPageByCnd(db *gorm.DB, cnd *sqls.Cnd) (list []models.HermixSkill, paging *sqls.Paging) {
	cnd.Find(db, &list)
	count := cnd.Count(db, &models.HermixSkill{})
	paging = &sqls.Paging{Page: cnd.Paging.Page, Limit: cnd.Paging.Limit, Total: count}
	return
}

func (r *hermixSkillRepository) Create(db *gorm.DB, t *models.HermixSkill) error {
	return db.Create(t).Error
}

func (r *hermixSkillRepository) Updates(db *gorm.DB, id int64, columns map[string]interface{}) error {
	return db.Model(&models.HermixSkill{}).Where("id = ?", id).Updates(columns).Error
}

func (r *hermixSkillRepository) UpdateColumn(db *gorm.DB, id int64, name string, value interface{}) error {
	return db.Model(&models.HermixSkill{}).Where("id = ?", id).UpdateColumn(name, value).Error
}
