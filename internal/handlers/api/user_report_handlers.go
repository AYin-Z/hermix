package api

import (
	"bbs-go/internal/models/req"
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/pkg/errs"
	"bbs-go/internal/services"

	"github.com/gin-gonic/gin"

	"bbs-go/internal/pkg/ginx"
)

func UserReportSubmit(ctx *gin.Context) {
	var req req.UserReportReq
	if err := ginx.Bind(ctx, &req); err != nil {
		ginx.WriteJSON(ctx, err)
		return
	}
	// 原先不要求登录：任何人都能匿名往审核队列灌任意 dataType/dataId，
	// 举报人记成 0 既无法追责也无法去重。举报是人工审核的入口，必须实名。
	user := common.GetCurrentUser(ctx)
	if user == nil {
		ginx.WriteJSON(ctx, errs.NotLogin())
		return
	}
	if err := services.UserReportService.Submit(user.Id, req.DataType, req.DecodedDataId(), req.Reason); err != nil {
		ginx.WriteJSON(ctx, err)
		return
	}
	ginx.WriteJSON(ctx, nil)
}
