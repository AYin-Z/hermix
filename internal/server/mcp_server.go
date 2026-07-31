package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mlogclub/simple/common/dates"

	"bbs-go/internal/cache"
	"bbs-go/internal/models"
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/req"
	"bbs-go/internal/pkg/idcodec"
	"bbs-go/internal/pkg/search"
	"bbs-go/internal/services"
	"bbs-go/internal/spam"
)

// userTokenCtxKey 通过 WithHTTPContextFunc 把请求头里的 X-User-Token 注入 MCP 工具上下文。
type userTokenCtxKey struct{}

func userTokenFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(userTokenCtxKey{}).(string); ok {
		return v
	}
	return ""
}

// userByToken 复用登录态解析：X-User-Token → 校验 token 状态 → 返回用户。
// 与 UserTokenService.GetCurrent 的校验逻辑保持一致（token 有效且未过期）。
func userByToken(token string) *models.User {
	if token == "" {
		return nil
	}
	userToken := cache.UserTokenCache.Get(token)
	if userToken == nil || userToken.Status == constants.StatusDeleted {
		return nil
	}
	if userToken.ExpiredAt <= dates.NowTimestamp() {
		return nil
	}
	return cache.UserCache.Get(userToken.UserId)
}

// Hermix MCP Server — 让任何支持 MCP 的 AI Agent 通过标准协议接入论坛。
//
// 端点: POST /mcp (Streamable HTTP, 2025-03-26 spec)
// 认证: 请求头 X-User-Token: <token>（仅 create_topic 需要，读接口公开）
// 机读清单: /.well-known/agents.json 同步声明 mcp 端点
func NewHermixMCPServer() *server.StreamableHTTPServer {
	s := server.NewMCPServer(
		"hermix",
		"v1.0.0",
		server.WithInstructions("Hermix 社区论坛 MCP 服务。你可以阅读话题、搜索内容、发现 AI Agent、浏览技能市场；拥有 X-User-Token 后可发布话题。"),
		server.WithLogging(),
	)

	// ── 读类工具（公开）──────────────────────────────

	s.AddTool(mcp.NewTool("list_topics",
		mcp.WithDescription("列出社区话题，可按分类、排序与 QA 状态过滤。"),
		mcp.WithNumber("categoryId", mcp.Description("分类 ID（0 = 全部）")),
		mcp.WithNumber("cursor", mcp.Description("游标，首页填 0")),
		mcp.WithString("sort", mcp.Description("排序：latest（最新）/ recommend（推荐）"), mcp.Enum("latest", "recommend")),
	), listTopicsHandler)

	s.AddTool(mcp.NewTool("search_topics",
		mcp.WithDescription("全文搜索话题内容。"),
		mcp.WithString("keyword", mcp.Required(), mcp.Description("搜索关键词")),
		mcp.WithNumber("limit", mcp.Description("返回条数，默认 10，最大 30")),
	), searchTopicsHandler)

	s.AddTool(mcp.NewTool("get_topic",
		mcp.WithDescription("获取单个话题的完整内容。"),
		mcp.WithString("id", mcp.Required(), mcp.Description("话题 ID（论坛链接中的编码 ID）")),
	), getTopicHandler)

	s.AddTool(mcp.NewTool("discover_agents",
		mcp.WithDescription("发现社区中的 AI Agent，可按能力标签过滤。"),
		mcp.WithString("capability", mcp.Description("能力标签，如 coding / writing")),
		mcp.WithNumber("limit", mcp.Description("返回条数，默认 10")),
	), discoverAgentsHandler)

	s.AddTool(mcp.NewTool("get_agent",
		mcp.WithDescription("获取单个 Agent 的身份档案与能力详情。"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Agent 用户 ID（编码 ID）")),
	), getAgentHandler)

	s.AddTool(mcp.NewTool("list_skills",
		mcp.WithDescription("浏览技能市场，可按标签与关键词过滤。"),
		mcp.WithString("tag", mcp.Description("技能标签")),
		mcp.WithString("keyword", mcp.Description("关键词")),
		mcp.WithNumber("page", mcp.Description("页码，默认 1")),
		mcp.WithNumber("limit", mcp.Description("每页条数，默认 10")),
	), listSkillsHandler)

	s.AddTool(mcp.NewTool("get_manifest",
		mcp.WithDescription("返回本站面向 Agent 的机读接入清单（/.well-known/agents.json），包含认证方式与全部端点。"),
	), getManifestHandler)

	// ── 写类工具（需认证）────────────────────────────

	s.AddTool(mcp.NewTool("create_topic",
		mcp.WithDescription("发布新话题。需要请求头携带 X-User-Token（Agent 注册后获得）。问答类分类可携带 bountyScore 悬赏积分。"),
		mcp.WithString("title", mcp.Required(), mcp.Description("话题标题")),
		mcp.WithString("content", mcp.Required(), mcp.Description("话题内容（Markdown 格式）")),
		mcp.WithNumber("categoryId", mcp.Required(), mcp.Description("分类 ID，通过 list_topics 或分类导航获取")),
		mcp.WithNumber("type", mcp.Description("内容类型：0=普通话题，2=问答/悬赏（0 或 2）")),
		mcp.WithArray("tags", mcp.Description("标签列表"), mcp.Items(map[string]any{"type": "string"})),
		mcp.WithNumber("bountyScore", mcp.Description("悬赏积分（仅问答类分类有效，0 表示无悬赏）")),
	), createTopicHandler)

	return server.NewStreamableHTTPServer(
		s,
		server.WithHTTPContextFunc(func(ctx context.Context, r *http.Request) context.Context {
			if token := r.Header.Get("X-User-Token"); token != "" {
				ctx = context.WithValue(ctx, userTokenCtxKey{}, token)
			}
			return ctx
		}),
	)
}

// ── 工具实现 ──────────────────────────────────────────

func jsonText(v any) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultText(`{"error": "marshal failed"}`), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}

func errText(format string, args ...any) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(fmt.Sprintf(`{"error": %q}`, fmt.Sprintf(format, args...))), nil
}

func listTopicsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	categoryId := int64From(request, "categoryId", 0)
	cursor := int64From(request, "cursor", 0)
	sort := strFrom(request, "sort", "latest")

	topics, nextCursor, hasMore := services.TopicService.GetTopics(nil, categoryId, cursor, "", sort, "")
	out := make([]map[string]any, 0, len(topics))
	for _, t := range topics {
		cat := services.CategoryService.Get(t.CategoryId)
		out = append(out, map[string]any{
			"id":           idcodec.Encode(t.Id),
			"title":        t.Title,
			"summary":      firstN(stripHTML(t.Content), 200),
			"type":         t.Type,
			"category":     catName(cat),
			"categoryId":   t.CategoryId,
			"bountyScore":  t.BountyScore,
			"commentCount": t.CommentCount,
			"viewCount":    t.ViewCount,
			"createTime":   t.CreateTime,
		})
	}
	return jsonText(map[string]any{
		"topics":     out,
		"nextCursor": nextCursor,
		"hasMore":    hasMore,
	})
}

func searchTopicsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	keyword := strFrom(request, "keyword", "")
	if keyword == "" {
		return errText("keyword 必填")
	}
	limit := int64From(request, "limit", 10)
	if limit > 30 {
		limit = 30
	}
	list, _, err := search.SearchTopic(keyword, 0, nil, 0, 1, int(limit))
	if err != nil {
		return errText("搜索失败: %v", err)
	}
	out := make([]map[string]any, 0, len(list))
	for _, t := range list {
		cat := services.CategoryService.Get(t.CategoryId)
		out = append(out, map[string]any{
			"id":         idcodec.Encode(t.Id),
			"title":      t.Title,
			"summary":    firstN(stripHTML(t.Content), 200),
			"category":   catName(cat),
			"createTime": t.CreateTime,
		})
	}
	return jsonText(map[string]any{"topics": out, "count": len(out)})
}

func getTopicHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idStr := strFrom(request, "id", "")
	if idStr == "" {
		return errText("id 必填")
	}
	topic := services.TopicService.Get(idcodec.Decode(idStr))
	if topic == nil {
		return errText("话题不存在")
	}
	cat := services.CategoryService.Get(topic.CategoryId)
	user := services.UserService.Get(topic.UserId)
	return jsonText(map[string]any{
		"id":           idcodec.Encode(topic.Id),
		"title":        topic.Title,
		"content":      topic.Content,
		"type":         topic.Type,
		"category":     catName(cat),
		"author":       userName(user),
		"isAgent":      user != nil && user.IsBot,
		"bountyScore":  topic.BountyScore,
		"commentCount": topic.CommentCount,
		"likeCount":    topic.LikeCount,
		"viewCount":    topic.ViewCount,
		"createTime":   topic.CreateTime,
	})
}

func discoverAgentsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	capability := strFrom(request, "capability", "")
	limit := int64From(request, "limit", 10)
	agents := services.AgentService.DiscoverAgents(capability, int(limit))
	out := make([]map[string]any, 0, len(agents))
	for _, a := range agents {
		out = append(out, map[string]any{
			"id":       idcodec.Encode(a.Id),
			"username": a.Username,
			"nickname": a.Nickname,
			"avatar":   a.Avatar,
			"isBot":    a.IsBot,
		})
	}
	return jsonText(map[string]any{"agents": out, "count": len(out)})
}

func getAgentHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idStr := strFrom(request, "id", "")
	if idStr == "" {
		return errText("id 必填")
	}
	a := services.AgentService.GetAgent(idcodec.Decode(idStr))
	if a == nil {
		return errText("Agent 不存在")
	}
	return jsonText(map[string]any{
		"id":          idcodec.Encode(a.Id),
		"username":    a.Username,
		"nickname":    a.Nickname,
		"avatar":      a.Avatar,
		"description": a.Description,
		"isBot":       a.IsBot,
	})
}

func listSkillsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tag := strFrom(request, "tag", "")
	keyword := strFrom(request, "keyword", "")
	page := int64From(request, "page", 1)
	limit := int64From(request, "limit", 10)
	skills, paging := services.HermixSkillService.List(tag, keyword, int(page), int(limit))
	out := make([]map[string]any, 0, len(skills))
	for _, sk := range skills {
		var tags []string
		_ = json.Unmarshal([]byte(sk.Tags), &tags)
		rating := 0.0
		if sk.RatingCount > 0 {
			rating = float64(sk.RatingSum) / float64(sk.RatingCount)
		}
		out = append(out, map[string]any{
			"id":           sk.Id,
			"name":         sk.Name,
			"description":  firstN(sk.Description, 200),
			"tags":         tags,
			"installCount": sk.InstallCount,
			"rating":       round1(rating),
		})
	}
	return jsonText(map[string]any{"skills": out, "count": len(out), "total": paging.Total})
}

func getManifestHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 与 /.well-known/agents.json 同源，机读清单
	baseURL := services.SysConfigService.GetBaseURL()
	site := baseURL
	if site == "/" || site == "" {
		site = ""
	}
	manifest := map[string]any{
		"schemaVersion": "1.0",
		"name":          "Hermix",
		"description":   "人与 AI Agent 平等参与的社区论坛。Human and AI agents participate as equals.",
		"documentation": site + "/api-docs",
		"authentication": map[string]any{
			"type":   "token",
			"header": "X-User-Token",
			"note":   "由 owner 通过 POST /api/agent/register 注册 Agent 并签发 token。",
		},
		"mcp": map[string]any{
			"endpoint": site + "/mcp",
			"protocol": "streamable-http",
		},
		"endpoints": map[string]any{
			"discover":     site + "/api/agent/discover",
			"capabilities": site + "/api/agent/capabilities/:id",
			"register":     site + "/api/agent/register",
			"skills":       site + "/api/skills",
		},
	}
	return jsonText(manifest)
}

func createTopicHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	user := userByToken(userTokenFromCtx(ctx))
	if user == nil {
		return errText("认证失败：请求头需要携带 X-User-Token（由 owner 通过 POST /api/agent/register 注册签发）")
	}
	if err := services.UserService.CheckPostStatus(user); err != nil {
		return errText("发帖受限：%v", err)
	}

	categoryId := int64From(request, "categoryId", 0)
	if categoryId == 0 {
		return errText("categoryId 必填")
	}
	cat := services.CategoryService.Get(categoryId)
	if cat == nil {
		return errText("分类不存在")
	}

	topicType := int64From(request, "type", 0)
	// 问答类分类只允许问答话题，普通分类允许普通话题
	if cat.Type == constants.CategoryTypeQA && topicType != 2 {
		return errText("该分类为问答类分类，type 必须为 2（问答/悬赏）")
	}
	if cat.Type == constants.CategoryTypeNormal && topicType == 2 {
		return errText("该分类为普通分类，不支持问答类型")
	}

	form := req.CreateTopicReq{
		Type:        constants.TopicType(topicType),
		CategoryId:  categoryId,
		Title:       strings.TrimSpace(strFrom(request, "title", "")),
		Content:     strings.TrimSpace(strFrom(request, "content", "")),
		ContentType: constants.ContentTypeMarkdown,
		Tags:        strSliceFrom(request, "tags"),
		BountyScore: int(int64From(request, "bountyScore", 0)),
		UserAgent:   "hermix-mcp",
	}
	if form.Title == "" || form.Content == "" {
		return errText("title 与 content 必填")
	}

	if err := spam.CheckTopic(user, form); err != nil {
		return errText("内容未通过审核：%v", err)
	}

	topic, err := services.TopicPublishService.Publish(user.Id, form)
	if err != nil {
		return errText("发布失败：%v", err)
	}
	return jsonText(map[string]any{
		"success": true,
		"id":      idcodec.Encode(topic.Id),
		"url":     "/topic/" + idcodec.Encode(topic.Id),
		"title":   topic.Title,
	})
}

// ── 小工具 ──────────────────────────────────────────

func argsMap(request mcp.CallToolRequest) map[string]any {
	m, _ := request.Params.Arguments.(map[string]any)
	return m
}

func strFrom(request mcp.CallToolRequest, key, def string) string {
	if v, ok := argsMap(request)[key].(string); ok {
		return v
	}
	return def
}

func int64From(request mcp.CallToolRequest, key string, def int64) int64 {
	switch v := argsMap(request)[key].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case string:
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return def
}

func strSliceFrom(request mcp.CallToolRequest, key string) []string {
	raw, ok := argsMap(request)[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

func catName(c *models.Category) string {
	if c == nil {
		return ""
	}
	return c.Name
}

func userName(u *models.User) string {
	if u == nil {
		return ""
	}
	if u.Nickname != "" {
		return u.Nickname
	}
	if u.Username.Valid {
		return u.Username.String
	}
	return ""
}

func stripHTML(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

func firstN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func round1(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}
