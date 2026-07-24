package repositories

import (
	"gorm.io/gorm"

	"bbs-go/internal/models"
)

var HermixSkillRatingRepository = newHermixSkillRatingRepository()

func newHermixSkillRatingRepository() *hermixSkillRatingRepository {
	return &hermixSkillRatingRepository{}
}

type hermixSkillRatingRepository struct{}

func (r *hermixSkillRatingRepository) FindBySkillAndUser(db *gorm.DB, skillId, userId int64) *models.HermixSkillRating {
	ret := &models.HermixSkillRating{}
	if err := db.Where("skill_id = ? AND user_id = ?", skillId, userId).First(ret).Error; err != nil {
		return nil
	}
	return ret
}

func (r *hermixSkillRatingRepository) Create(db *gorm.DB, t *models.HermixSkillRating) error {
	return db.Create(t).Error
}
