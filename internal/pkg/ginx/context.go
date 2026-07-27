package ginx

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func Bind(ctx *gin.Context, obj any) error {
	return ctx.ShouldBind(obj)
}

func BindJSON(ctx *gin.Context, obj any) error {
	return ctx.ShouldBindJSON(obj)
}

func BindQuery(ctx *gin.Context, obj any) error {
	return ctx.ShouldBindQuery(obj)
}

func BindForm(ctx *gin.Context, obj any) error {
	return ctx.ShouldBind(obj)
}

func FormValues(ctx *gin.Context) url.Values {
	_ = ctx.Request.ParseMultipartForm(32 << 20)
	_ = ctx.Request.ParseForm()
	values := url.Values{}
	for k, v := range ctx.Request.URL.Query() {
		values[k] = append(values[k], v...)
	}
	for k, v := range ctx.Request.PostForm {
		values[k] = append(values[k], v...)
	}
	return values
}

func FormValue(ctx *gin.Context, name string) string {
	if value := ctx.Query(name); value != "" {
		return value
	}
	return ctx.PostForm(name)
}

func FormValueDefault(ctx *gin.Context, name, def string) string {
	if value := FormValue(ctx, name); value != "" {
		return value
	}
	return def
}

func GetCookie(ctx *gin.Context, name string) string {
	value, _ := ctx.Cookie(name)
	return value
}

func SetCookieKV(ctx *gin.Context, name, value string, opts ...CookieOption) {
	options := cookieOptions{}
	for _, opt := range opts {
		opt(&options)
	}
	maxAge := 0
	if options.expires > 0 {
		maxAge = int(options.expires.Seconds())
	}
	setSameSiteLax(ctx)
	ctx.SetCookie(name, value, maxAge, "/", "", isHTTPS(ctx), options.httpOnly)
}

func RemoveCookie(ctx *gin.Context, name string) {
	setSameSiteLax(ctx)
	ctx.SetCookie(name, "", -1, "/", "", isHTTPS(ctx), true)
}

// setSameSiteLax 给后续 SetCookie 打上 SameSite=Lax。
// 会话 cookie 缺少 SameSite 时浏览器按 None 处理，跨站表单/图片请求都会带上它，
// 站内所有状态变更接口（改资料、发帖、采纳答案…）因此可被 CSRF 触发。
// Lax 保留正常的站外链接跳转（GET 顶层导航仍携带 cookie），只挡掉跨站的
// POST 与子资源请求，对现有前端无影响。
func setSameSiteLax(ctx *gin.Context) {
	ctx.SetSameSite(http.SameSiteLaxMode)
}

// isHTTPS 判断当前请求是否走 TLS，用于决定 Secure 标记。
// 生产在 traefik 后面，TLS 在反代终止，因此需同时看 X-Forwarded-Proto。
func isHTTPS(ctx *gin.Context) bool {
	if ctx.Request == nil {
		return false
	}
	if ctx.Request.TLS != nil {
		return true
	}
	return strings.EqualFold(ctx.GetHeader("X-Forwarded-Proto"), "https")
}

type CookieOption func(*cookieOptions)

type cookieOptions struct {
	httpOnly bool
	expires  time.Duration
}

func CookieHTTPOnly(enabled bool) CookieOption {
	return func(opts *cookieOptions) {
		opts.httpOnly = enabled
	}
}

func CookieExpires(duration time.Duration) CookieOption {
	return func(opts *cookieOptions) {
		opts.expires = duration
	}
}
