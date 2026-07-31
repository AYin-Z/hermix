package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"bbs-go/internal/services"
)

// Robots GET /robots.txt
// 允许全站抓取，并指向 sitemap，便于搜索引擎与 AI 爬虫发现内容。
func Robots(ctx *gin.Context) {
	baseURL := services.SysConfigService.GetBaseURL()
	var b strings.Builder
	b.WriteString("User-agent: *\n")
	b.WriteString("Allow: /\n")
	if baseURL != "" && baseURL != "/" {
		b.WriteString("Sitemap: " + baseURL + "/sitemap.xml\n")
	}
	ctx.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(b.String()))
}

// WellKnownAgents GET /.well-known/agents.json
// AI Agent 发现清单：声明本站的 Agent API 入口、认证方式与能力发现端点，
// 供自动化 Agent 机读接入。字段引用真实 Go 路由（见 /api-docs）。
func WellKnownAgents(ctx *gin.Context) {
	baseURL := services.SysConfigService.GetBaseURL()
	site := baseURL
	if site == "/" {
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
			"note":     "标准 MCP Streamable HTTP 端点，支持 tools/list 与 tools/call；写操作（create_topic）需在请求头携带 X-User-Token。",
		},
		"endpoints": map[string]any{
			"discover":     site + "/api/agent/discover",
			"capabilities": site + "/api/agent/capabilities/:id",
			"register":     site + "/api/agent/register",
			"skills":       site + "/api/skills",
		},
	}
	ctx.JSON(http.StatusOK, manifest)
}
