package router

import (
	"github.com/gin-gonic/gin"

	"personal_assistant_server/internal/handler"
)

// registerPublicRoutes 注册无需身份认证即可访问的公开业务接口。
//
// 公开接口不会经过 JWT 认证中间件，适合登录、注册、验证码和其他必须在用户
// 获得访问令牌之前调用的功能。后续新增无需登录的业务接口时，应统一在本函数中注册。
//
// 参数：
//   - api：由 router.New 创建的基础 API 路由组，统一使用 /api 前缀。
//   - authHandler：认证 HTTP 处理器，负责处理登录等公开认证请求。
//
// 返回值：
//   - 无。
func registerPublicRoutes(api *gin.RouterGroup, authHandler *handler.Auth) {
	api.GET("/captcha", authHandler.Captcha)
	api.POST("/login", authHandler.Login)
}
