package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"personal_assistant_server/internal/config"
	"personal_assistant_server/internal/finance"
	"personal_assistant_server/internal/handler"
	"personal_assistant_server/internal/middleware"
	"personal_assistant_server/internal/response"
	"personal_assistant_server/internal/service"
	taskhandler "personal_assistant_server/internal/task/task/handler"
	taskservice "personal_assistant_server/internal/task/task/service"
	tasklisthandler "personal_assistant_server/internal/task/tasklist/handler"
	tasklistservice "personal_assistant_server/internal/task/tasklist/service"
)

// New 创建并配置应用的 Gin HTTP 路由引擎。
//
// 函数根据应用配置设置 Gin 运行模式、可信代理和全局中间件，随后初始化认证服务，
// 注册健康检查、登录验证码和用户登录接口，并统一处理未匹配的路由。
//
// 参数：
//   - db：已经完成连接和初始化的 GORM 数据库实例。
//   - cfg：已经完成加载和校验的全局应用配置，包含运行模式、跨域和 JWT 参数。
//
// 返回值：
//   - *gin.Engine：已经注册全局中间件、业务接口和未匹配路由处理器的 Gin 引擎。
func New(db *gorm.DB, cfg *config.Config) *gin.Engine {
	// 初始化 Gin 引擎并注册访问日志、异常恢复和跨域处理中间件。
	gin.SetMode(cfg.Mode)
	engine := gin.New()
	_ = engine.SetTrustedProxies(nil)
	engine.Use(
		middleware.Logger(),                 // 记录请求方法、路径、状态码、客户端 IP 和处理耗时
		gin.Recovery(),                      // 捕获请求处理过程中的 panic，避免 HTTP 服务异常退出
		middleware.CORS(cfg.AllowedOrigins), // 根据配置的允许来源处理跨域请求和 OPTIONS 预检请求
	)

	// 创建认证服务，集中处理用户查询、密码校验、JWT 签发和 JWT 解析：
	//   - db：已经初始化的数据库连接，用于查询用户和个人资料；
	//   - cfg.JWTSecret：JWT 签名密钥，用于签发令牌并验证令牌是否被篡改；
	//   - cfg.JWTExpireHours：JWT 有效时长，单位为小时，用于计算令牌过期时间。
	authService := service.NewAuth(db, cfg.JWTSecret, cfg.JWTExpireHours)
	captchaService := service.NewCaptcha()
	taskService := taskservice.New(db)
	taskListService := tasklistservice.New(db)
	financeService := finance.NewService(db)

	// 创建认证 HTTP 处理器，并注入认证服务；处理器负责解析请求参数、调用认证业务逻辑，
	// 再将业务执行结果转换为统一的 HTTP JSON 响应。
	authHandler := handler.NewAuth(authService, captchaService)
	taskHandler := taskhandler.New(taskService)
	taskListHandler := tasklisthandler.New(taskListService)
	financeHandler := finance.NewHandler(financeService)

	// 健康检查不依赖数据库业务逻辑和用户认证，直接注册在基础 API 路由组中。
	api := engine.Group("/api")
	api.GET("/health", func(c *gin.Context) {
		response.OK(c, gin.H{"service": "personal-assistant-api", "status": "ok"})
	})

	// 对整个 HTTP 服务中所有未匹配的请求返回统一的 404 JSON 响应。
	engine.NoRoute(func(c *gin.Context) {
		response.Error(c, http.StatusNotFound, "接口不存在")
	})

	// 集中注册依赖业务服务的公开接口和受保护接口。
	registerRoutes(api, authHandler, authService, taskHandler, taskListHandler, financeHandler)
	return engine
}
