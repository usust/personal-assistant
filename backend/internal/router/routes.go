package router

import (
	"github.com/gin-gonic/gin"

	"personal_assistant_server/internal/handler"
	"personal_assistant_server/internal/middleware"
	"personal_assistant_server/internal/service"
	taskroutes "personal_assistant_server/internal/task/task"
	taskhandler "personal_assistant_server/internal/task/task/handler"
	tasklistroutes "personal_assistant_server/internal/task/tasklist"
	tasklisthandler "personal_assistant_server/internal/task/tasklist/handler"
)

// registerRoutes 组织并注册应用的全部业务 HTTP 路由。
//
// 当前业务接口均为无需身份认证的公开接口，由 registerPublicRoutes 统一注册。
// 后续增加其他类型的业务接口时，可以继续在本函数中组织对应的路由注册函数。
//
// 参数：
//   - api：由 router.New 创建的基础 API 路由组，统一使用 /api 前缀。
//   - authHandler：认证 HTTP 处理器，负责处理登录等公开认证请求。
//
// 返回值：
//   - 无。
func registerRoutes(
	api *gin.RouterGroup,
	authHandler *handler.Auth,
	authService *service.Auth,
	taskHandler *taskhandler.Handler,
	taskListHandler *tasklisthandler.Handler,
) {
	// 注册无需身份认证即可访问的公开业务接口。
	registerPublicRoutes(api, authHandler)

	protected := api.Group("")
	protected.Use(middleware.Auth(authService))
	tasklistroutes.RegisterRoutes(protected, taskListHandler)
	taskroutes.RegisterRoutes(protected, taskHandler)
}
