package services

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/req"
	"bbs-go/internal/permissions"
	"bbs-go/internal/pkg/errs"
	"bbs-go/internal/pkg/event"
	"bbs-go/internal/pkg/iplocator"
	"bbs-go/internal/pkg/locales"
	"errors"
	"log/slog"
	"strings"

	"bbs-go/internal/pkg/params"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/common/jsons"
	"github.com/mlogclub/simple/common/strs"
	"github.com/mlogclub/simple/sqls"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"gorm.io/gorm"

	"bbs-go/internal/models"
	"bbs-go/internal/repositories"
)

var CommentService = newCommentService()

func newCommentService() *commentService {
	return &commentService{}
}

type commentService struct {
}

func (s *commentService) Get(id int64) *models.Comment {
	return repositories.CommentRepository.Get(sqls.DB(), id)
}

func (s *commentService) Take(where ...interface{}) *models.Comment {
	return repositories.CommentRepository.Take(sqls.DB(), where...)
}

func (s *commentService) Find(cnd *sqls.Cnd) []models.Comment {
	return repositories.CommentRepository.Find(sqls.DB(), cnd)
}

func (s *commentService) FindOne(cnd *sqls.Cnd) *models.Comment {
	return repositories.CommentRepository.FindOne(sqls.DB(), cnd)
}

func (s *commentService) FindPageByParams(params *params.QueryParams) (list []models.Comment, paging *sqls.Paging) {
	return repositories.CommentRepository.FindPageByParams(sqls.DB(), params)
}

func (s *commentService) FindPageByCnd(cnd *sqls.Cnd) (list []models.Comment, paging *sqls.Paging) {
	return repositories.CommentRepository.FindPageByCnd(sqls.DB(), cnd)
}

func (s *commentService) Count(cnd *sqls.Cnd) int64 {
	return repositories.CommentRepository.Count(sqls.DB(), cnd)
}

func (s *commentService) Create(t *models.Comment) error {
	return repositories.CommentRepository.Create(sqls.DB(), t)
}

func (s *commentService) Update(t *models.Comment) error {
	return repositories.CommentRepository.Update(sqls.DB(), t)
}

func (s *commentService) Updates(id int64, columns map[string]interface{}) error {
	return repositories.CommentRepository.Updates(sqls.DB(), id, columns)
}

func (s *commentService) UpdateColumn(id int64, name string, value interface{}) error {
	return repositories.CommentRepository.UpdateColumn(sqls.DB(), id, name, value)
}

func (s *commentService) Delete(id int64) error {
	return repositories.CommentRepository.UpdateColumn(sqls.DB(), id, "status", constants.StatusDeleted)
}

// Audit 放行待审评论。
// 发布时若命中违禁词，评论会以 StatusReview 落库且不计楼层数、不发通知，
// 这里补上当时跳过的那几步，所以只接受 StatusReview 的评论 —— 对已放行/已删除的
// 评论重复调用会把计数加两次。
func (s *commentService) Audit(id int64) error {
	comment := s.Get(id)
	if comment == nil {
		return errors.New("comment not found")
	}
	if comment.Status == constants.StatusOk {
		return nil
	}
	if comment.Status != constants.StatusReview {
		return errors.New(locales.Get("comment.not_in_review"))
	}

	if err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		if err := repositories.CommentRepository.UpdateColumn(tx, comment.Id, "status", constants.StatusOk); err != nil {
			return err
		}
		switch comment.EntityType {
		case constants.EntityTopic:
			return TopicService.onComment(tx, comment.EntityId, comment)
		case constants.EntityComment:
			return s.onComment(tx, comment)
		}
		return nil
	}); err != nil {
		return err
	}

	UserService.IncrCommentCount(comment.UserId)
	event.Send(event.CommentCreateEvent{
		UserId:    comment.UserId,
		CommentId: comment.Id,
	})
	return nil
}

func (s *commentService) DeleteByUser(user *models.User, id int64) error {
	if user == nil {
		return errs.NotLogin()
	}
	comment := s.Get(id)
	if comment == nil || comment.Status == constants.StatusDeleted {
		return errors.New("comment not found")
	}
	if !PermissionService.CanManageOwnedResource(user, comment.UserId, permissions.PermissionCommentDelete.Code) {
		return errs.NoPermission()
	}
	return s.Delete(id)
}

// Publish 发表评论
func (s *commentService) Publish(userId int64, form req.CreateCommentReq) (*models.Comment, error) {
	form.Content = strings.TrimSpace(form.Content)
	entityId := form.DecodedEntityId()
	if strs.IsBlank(form.EntityType) {
		return nil, errors.New(locales.Get("comment.invalid_params"))
	}
	if entityId <= 0 {
		return nil, errors.New(locales.Get("comment.invalid_params"))
	}
	if strs.IsBlank(form.Content) {
		return nil, errors.New(locales.Get("comment.content_required"))
	}

	// 违禁词命中则进人工审核队列，不直接放出。
	// 此前只有主题发布查违禁词，评论完全不查 —— 敏感内容改用评论发即可绕过整条审核链。
	status := constants.StatusOk
	if hits := ForbiddenWordService.Check(form.Content); len(hits) > 0 {
		slog.Info("评论命中违禁词", slog.String("hits", strings.Join(hits, ",")))
		status = constants.StatusReview
	}

	comment := &models.Comment{
		UserId:      userId,
		EntityType:  form.EntityType,
		EntityId:    entityId,
		Content:     form.Content,
		ContentType: constants.ContentTypeText,
		QuoteId:     form.QuoteId,
		Status:      status,
		UserAgent:   form.UserAgent,
		Ip:          form.Ip,
		IpLocation:  iplocator.IpLocation(form.Ip),
		CreateTime:  dates.NowTimestamp(),
	}

	imageList := form.ParsedImageList()
	if len(imageList) > 0 {
		imageListStr, err := jsons.ToStr(imageList)
		if err == nil {
			comment.ImageList = imageListStr
		} else {
			slog.Error(err.Error(), slog.Any("err", err))
		}
	}

	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		if err := repositories.CommentRepository.Create(tx, comment); err != nil {
			return err
		}

		// 待审评论对读者不可见，先不计入楼层数与「最后回复」，等放行时再补。
		if comment.Status != constants.StatusOk {
			return nil
		}

		switch form.EntityType {
		case constants.EntityTopic:
			if err := TopicService.onComment(tx, entityId, comment); err != nil {
				return err
			}
		case constants.EntityComment: // 二级评论
			if err := s.onComment(tx, comment); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 待审评论不计数、不发通知，避免审核前就把内容以通知形式推给他人。
	if comment.Status != constants.StatusOk {
		return comment, nil
	}

	// 用户跟帖计数
	UserService.IncrCommentCount(userId)
	// 发送事件
	event.Send(event.CommentCreateEvent{
		UserId:    userId,
		CommentId: comment.Id,
	})

	return comment, nil
}

// onComment 评论被回复（二级评论）
func (s *commentService) onComment(tx *gorm.DB, comment *models.Comment) error {
	return repositories.CommentRepository.UpdateColumn(tx, comment.EntityId, "comment_count", gorm.Expr("comment_count + 1"))
}

// // 统计数量
// func (s *commentService) Count(entityType string, entityId int64) int64 {
// 	var count int64 = 0
// 	sqls.DB().Model(&model.Comment{}).Where("entity_type = ? and entity_id = ?", entityType, entityId).Count(&count)
// 	return count
// }

// GetComments 列表
func (s *commentService) GetComments(entityType string, entityId int64, cursor int64) (comments []models.Comment, nextCursor int64, hasMore bool) {
	limit := 20
	var acceptedComment *models.Comment
	var acceptedCommentId int64

	if entityType == constants.EntityTopic {
		if topic := TopicService.Get(entityId); topic != nil && topic.AcceptedCommentId > 0 {
			acceptedCommentId = topic.AcceptedCommentId
			if acceptedComment = repositories.CommentRepository.FindOne(sqls.DB(), sqls.NewCnd().
				Eq("id", acceptedCommentId).
				Eq("entity_type", entityType).
				Eq("entity_id", entityId).
				Eq("status", constants.StatusOk)); acceptedComment == nil {
				acceptedCommentId = 0
			}
		}
	}

	// First page reserves one slot for accepted answer if present, so it can stay pinned on top.
	normalLimit := limit
	if cursor <= 0 && acceptedComment != nil {
		normalLimit = limit - 1
	}

	cnd := sqls.NewCnd().
		Eq("entity_type", entityType).
		Eq("entity_id", entityId).
		Eq("status", constants.StatusOk).
		Desc("id").
		Limit(normalLimit)
	if cursor > 0 {
		cnd.Lt("id", cursor)
	}
	if acceptedCommentId > 0 {
		cnd.Where("id <> ?", acceptedCommentId)
	}

	normalComments := repositories.CommentRepository.Find(sqls.DB(), cnd)

	if cursor <= 0 && acceptedComment != nil {
		comments = append(comments, *acceptedComment)
	}
	comments = append(comments, normalComments...)

	if len(normalComments) > 0 {
		nextCursor = normalComments[len(normalComments)-1].Id
		hasMore = len(normalComments) >= normalLimit
	} else {
		nextCursor = cursor
		hasMore = false
	}
	return
}

// GetReplies 二级回复列表
func (s *commentService) GetReplies(commentId int64, cursor int64, limit int) (comments []models.Comment, nextCursor int64, hasMore bool) {
	cnd := sqls.NewCnd().Eq("entity_type", constants.EntityComment).Eq("entity_id", commentId).Eq("status", constants.StatusOk).Asc("id").Limit(limit)
	if cursor > 0 {
		cnd.Gt("id", cursor)
	}
	comments = s.Find(cnd)
	if len(comments) > 0 {
		nextCursor = comments[len(comments)-1].Id
		hasMore = len(comments) >= limit
	} else {
		nextCursor = cursor
	}
	return
}

// ScanByUser 按照用户扫描数据
func (s *commentService) ScanByUser(userId int64, callback func(comments []models.Comment)) {
	var cursor int64 = 0
	for {
		list := repositories.CommentRepository.Find(sqls.DB(), sqls.NewCnd().
			Eq("user_id", userId).Gt("id", cursor).Asc("id").Limit(1000))
		if len(list) == 0 {
			break
		}
		cursor = list[len(list)-1].Id
		callback(list)
	}
}

// ScanByUser 按照用户扫描数据
func (s *commentService) Scan(callback func(comments []models.Comment)) {
	var cursor int64 = 0
	for {
		logrus.Info("scan comments, cursor:" + cast.ToString(cursor))
		list := repositories.CommentRepository.Find(sqls.DB(), sqls.NewCnd().
			Gt("id", cursor).Asc("id").Limit(1000))
		if len(list) == 0 {
			break
		}
		cursor = list[len(list)-1].Id
		callback(list)
	}
}

func (s *commentService) IsCommented(userId int64, entityType string, entityId int64) bool {
	return s.FindOne(sqls.NewCnd().Where("user_id = ? and entity_id = ? and entity_type = ? and status = ?", userId, entityId, entityType, constants.StatusOk)) != nil
}
