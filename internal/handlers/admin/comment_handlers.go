package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mlogclub/simple/web"

	"bbs-go/internal/handlers/render"
	"bbs-go/internal/models"
	"bbs-go/internal/models/constants"
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/pkg/errs"
	"bbs-go/internal/pkg/ginx"
	"bbs-go/internal/pkg/idcodec"
	"bbs-go/internal/pkg/params"
	"bbs-go/internal/services"
)

func CommentDetail(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		ginx.WriteJSON(ctx, err)
		return
	}
	t := services.CommentService.Get(id)
	if t == nil {
		ginx.WriteJSON(ctx, ginx.ErrorMessage("Not found, id="+strconv.FormatInt(id, 10)))
		return
	}
	ginx.WriteJSON(ctx, buildAdminComment(t))
}

func CommentList(ctx *gin.Context) {
	list, paging := services.CommentService.FindPageByCnd(params.NewPagedSqlCnd(ctx,
		params.QueryFilter{
			ParamName: "id",
			Op:        params.Eq,
		},
		params.QueryFilter{
			ParamName: "userId",
			Op:        params.Eq,
			ValueWrapper: func(origin string) string {
				if id := idcodec.Decode(origin); id > 0 {
					return strconv.FormatInt(id, 10)
				}
				return ""
			},
		},
		params.QueryFilter{
			ParamName: "entityType",
			Op:        params.Eq,
		},
		params.QueryFilter{
			ParamName: "entityId",
			Op:        params.Eq,
		},
		params.QueryFilter{
			ParamName: "status",
			Op:        params.Eq,
		},
		params.QueryFilter{
			ParamName: "content",
			Op:        params.Like,
		},
	).Desc("id"))

	var results []map[string]interface{}
	for _, comment := range list {
		results = append(results, buildAdminComment(&comment))
	}
	ginx.WriteJSON(ctx, &web.PageResult{Results: results, Page: paging})
}

// CommentAudit 放行待审评论
func CommentAudit(ctx *gin.Context) {
	id, _ := params.GetInt64(ctx, "id")
	if id <= 0 {
		ginx.WriteJSON(ctx, ginx.ErrorMessage("id is required"))
		return
	}
	if err := services.CommentService.Audit(id); err != nil {
		ginx.WriteJSON(ctx, err)
		return
	}
	ginx.WriteJSON(ctx, nil)
}

func CommentRemove(ctx *gin.Context) {
	id, err := params.FormValueInt64(ctx, "id")
	if err != nil {
		ginx.WriteJSON(ctx, err)
		return
	}
	if user := common.GetCurrentUser(ctx); user == nil {
		ginx.WriteJSON(ctx, errs.NotLogin())
		return
	}
	if err := services.CommentService.Delete(id); err != nil {
		ginx.WriteJSON(ctx, err)
		return
	}
	ginx.WriteJSON(ctx, nil)
}

// buildAdminComment 后台列表用的评论数据。
// 与前台不同：明文 id、带上 status、附上宿主帖链接，方便审核时点进去看上下文。
func buildAdminComment(comment *models.Comment) map[string]interface{} {
	builder := web.NewRspBuilder(comment)
	builder.Put("user", render.BuildUserInfoDefaultIfNull(comment.UserId))
	builder.Put("status", comment.Status)
	if comment.EntityType == constants.EntityTopic {
		builder.Put("entityUrl", "/topic/"+idcodec.Encode(comment.EntityId))
	}
	return builder.Build()
}
