package services

import (
	"bbs-go/internal/cache"
	"bbs-go/internal/models"
	"bbs-go/internal/models/constants"
	"bbs-go/internal/pkg/locales"
	"bbs-go/internal/repositories"
	"errors"
	"strings"

	"bbs-go/internal/pkg/params"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/common/strs"
	"github.com/mlogclub/simple/sqls"
)

var UserReportService = newUserReportService()

func newUserReportService() *userReportService {
	return &userReportService{}
}

type userReportService struct {
}

func (s *userReportService) Get(id int64) *models.UserReport {
	return repositories.UserReportRepository.Get(sqls.DB(), id)
}

func (s *userReportService) Take(where ...interface{}) *models.UserReport {
	return repositories.UserReportRepository.Take(sqls.DB(), where...)
}

func (s *userReportService) Find(cnd *sqls.Cnd) []models.UserReport {
	return repositories.UserReportRepository.Find(sqls.DB(), cnd)
}

func (s *userReportService) FindOne(cnd *sqls.Cnd) *models.UserReport {
	return repositories.UserReportRepository.FindOne(sqls.DB(), cnd)
}

func (s *userReportService) FindPageByParams(params *params.QueryParams) (list []models.UserReport, paging *sqls.Paging) {
	return repositories.UserReportRepository.FindPageByParams(sqls.DB(), params)
}

func (s *userReportService) FindPageByCnd(cnd *sqls.Cnd) (list []models.UserReport, paging *sqls.Paging) {
	return repositories.UserReportRepository.FindPageByCnd(sqls.DB(), cnd)
}

func (s *userReportService) Count(cnd *sqls.Cnd) int64 {
	return repositories.UserReportRepository.Count(sqls.DB(), cnd)
}

func (s *userReportService) Create(t *models.UserReport) error {
	return repositories.UserReportRepository.Create(sqls.DB(), t)
}

// Submit 用户提交举报。
// 校验三件事：举报对象必须是已知实体类型且真实存在（否则队列里全是指向不存在 id 的垃圾）、
// 理由长度受限（reason 列 1024，不截会被数据库拒绝）、同一用户对同一对象不重复挂待审举报
// （否则一个人刷 1000 条就把审核队列冲掉）。
func (s *userReportService) Submit(userId int64, dataType string, dataId int64, reason string) error {
	reason = strings.TrimSpace(reason)
	if strs.IsBlank(reason) {
		return errors.New(locales.Get("user_report.reason_required"))
	}
	if strs.RuneLen(reason) > 500 {
		return errors.New(locales.Get("user_report.reason_too_long"))
	}
	if dataId <= 0 || !s.dataExists(dataType, dataId) {
		return errors.New(locales.Get("user_report.target_not_found"))
	}

	exists := s.FindOne(sqls.NewCnd().
		Eq("user_id", userId).
		Eq("data_type", dataType).
		Eq("data_id", dataId).
		Eq("audit_status", 0)) // 0 = 未处理，与后台列表/概览口径一致
	if exists != nil {
		return errors.New(locales.Get("user_report.already_reported"))
	}

	return s.Create(&models.UserReport{
		DataId:     dataId,
		DataType:   dataType,
		UserId:     userId,
		Reason:     reason,
		CreateTime: dates.NowTimestamp(),
	})
}

// dataExists 举报对象是否存在。只认这四种实体，其余 dataType 直接拒。
func (s *userReportService) dataExists(dataType string, dataId int64) bool {
	switch dataType {
	case constants.EntityTopic:
		return TopicService.Get(dataId) != nil
	case constants.EntityArticle:
		return ArticleService.Get(dataId) != nil
	case constants.EntityComment:
		return CommentService.Get(dataId) != nil
	case constants.EntityUser:
		return cache.UserCache.Get(dataId) != nil
	}
	return false
}

func (s *userReportService) Update(t *models.UserReport) error {
	return repositories.UserReportRepository.Update(sqls.DB(), t)
}

func (s *userReportService) Updates(id int64, columns map[string]interface{}) error {
	return repositories.UserReportRepository.Updates(sqls.DB(), id, columns)
}

func (s *userReportService) UpdateColumn(id int64, name string, value interface{}) error {
	return repositories.UserReportRepository.UpdateColumn(sqls.DB(), id, name, value)
}

func (s *userReportService) Delete(id int64) {
	repositories.UserReportRepository.Delete(sqls.DB(), id)
}
