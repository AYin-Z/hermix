package api

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/mlogclub/simple/web"
	"github.com/spf13/cast"

	"bbs-go/internal/handlers/render"
	"bbs-go/internal/models"
	"bbs-go/internal/models/constants"
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/pkg/errs"
	"bbs-go/internal/pkg/ginx"
	"bbs-go/internal/pkg/idcodec"
	"bbs-go/internal/services"
)

// buildSkill 构造技能返回对象（含作者信息与平均分）。
func buildSkill(skill *models.HermixSkill) map[string]any {
	var tags []string
	_ = json.Unmarshal([]byte(skill.Tags), &tags)
	if tags == nil {
		tags = []string{}
	}
	var avg float64
	if skill.RatingCount > 0 {
		avg = float64(skill.RatingSum) / float64(skill.RatingCount)
	}
	return map[string]any{
		"id":             idcodec.Encode(skill.Id),
		"name":           skill.Name,
		"description":    skill.Description,
		"installCommand": skill.InstallCommand,
		"tags":           tags,
		"rating":         avg,
		"ratingCount":    skill.RatingCount,
		"installCount":   skill.InstallCount,
		"author":         render.BuildUserInfoDefaultIfNull(skill.AuthorId),
		"createTime":     skill.CreateTime,
	}
}

// SkillList GET /api/skills 列表（公开）
func SkillList(ctx *gin.Context) {
	tag := ctx.Query("tag")
	keyword := ctx.Query("keyword")
	page := cast.ToInt(ctx.Query("page"))
	limit := cast.ToInt(ctx.Query("limit"))
	skills, paging := services.HermixSkillService.List(tag, keyword, page, limit)
	items := make([]map[string]any, 0, len(skills))
	for i := range skills {
		items = append(items, buildSkill(&skills[i]))
	}
	ginx.WriteJSON(ctx, ginx.PageData(items, paging))
}

// skillPublishReq 发布请求体
type skillPublishReq struct {
	Name           string   `json:"name" form:"name"`
	Description    string   `json:"description" form:"description"`
	InstallCommand string   `json:"installCommand" form:"installCommand"`
	Tags           []string `json:"tags" form:"tags"`
}

// SkillPublish POST /api/skills 发布（需登录）
func SkillPublish(ctx *gin.Context) {
	user := common.GetCurrentUser(ctx)
	if user == nil {
		ginx.WriteJSON(ctx, errs.NotLogin())
		return
	}
	var r skillPublishReq
	if err := ginx.BindJSON(ctx, &r); err != nil {
		ginx.WriteJSON(ctx, err)
		return
	}
	skill, err := services.HermixSkillService.Publish(user.Id, services.SkillPublishInput{
		Name:           r.Name,
		Description:    r.Description,
		InstallCommand: r.InstallCommand,
		Tags:           r.Tags,
	})
	if err != nil {
		ginx.WriteJSON(ctx, ginx.ErrorMessage(err.Error()))
		return
	}
	ginx.WriteJSON(ctx, buildSkill(skill))
}

// skillRateReq 评分请求体
type skillRateReq struct {
	Score int `json:"score" form:"score"`
}

// SkillRate POST /api/skills/rate/:id 评分（需登录）
func SkillRate(ctx *gin.Context) {
	user := common.GetCurrentUser(ctx)
	if user == nil {
		ginx.WriteJSON(ctx, errs.NotLogin())
		return
	}
	skillId := idcodec.Decode(ctx.Param("id"))
	var r skillRateReq
	if err := ginx.BindJSON(ctx, &r); err != nil {
		ginx.WriteJSON(ctx, err)
		return
	}
	if err := services.HermixSkillService.Rate(skillId, user.Id, r.Score); err != nil {
		ginx.WriteJSON(ctx, ginx.ErrorMessage(err.Error()))
		return
	}
	ginx.WriteJSON(ctx, nil)
}

// SkillInstall POST /api/skills/install/:id 记录安装（公开计数）
func SkillInstall(ctx *gin.Context) {
	skillId := idcodec.Decode(ctx.Param("id"))
	skill := services.HermixSkillService.Get(skillId)
	if skill == nil || skill.Status != constants.StatusOk {
		ginx.WriteJSON(ctx, ginx.ErrorMessage("技能不存在"))
		return
	}
	_ = services.HermixSkillService.IncrInstall(skillId)
	ginx.WriteJSON(ctx, web.NewEmptyRspBuilder().
		Put("installCommand", skill.InstallCommand))
}
