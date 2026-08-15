package middleware

import (
	"time"

	go_utils "personal_assistant_server/internal/goutils"

	"github.com/gin-gonic/gin"
)

// Logger 创建用于记录 Gin HTTP 请求访问日志的中间件。
//
// 中间件在请求进入业务处理链前记录开始时间、请求路径和查询参数，随后通过
// c.Next 继续执行后续中间件及路由处理函数。请求处理完成后，中间件收集响应
// 状态码、请求方法、客户端 IP、处理耗时和 Gin 上下文错误，并通过 goUtils
// 初始化的 Zap SugaredLogger 输出结构化访问日志。
//
// 日志级别根据 HTTP 响应状态码确定：500 及以上使用 Error，400～499 使用 Warn，
// 其余状态码使用 Info，便于按照请求处理结果筛选和检索日志。
//
// 参数：
//   - 无。
//
// 返回值：
//   - gin.HandlerFunc：记录每个 HTTP 请求处理结果和耗时的 Gin 中间件函数。
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 在执行后续处理函数前保存开始时间，用于计算请求的完整处理耗时。
		startedAt := time.Now()

		// 提前保存原始请求路径和查询参数，避免后续处理过程修改请求对象后影响日志内容。
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery

		// 继续执行中间件链和最终路由处理函数；返回后再记录响应结果。
		c.Next()

		// 组织结构化日志字段，完整描述本次请求、响应结果和处理过程中产生的错误。
		fields := []any{
			"status", c.Writer.Status(), // HTTP 响应状态码
			"method", c.Request.Method, // HTTP 请求方法
			"path", path, // 不包含查询参数的请求路径
			"query", rawQuery, // 原始 URL 查询参数
			"client_ip", c.ClientIP(), // 根据 Gin 代理信任配置解析的客户端 IP
			"latency", time.Since(startedAt), // 请求进入中间件至响应完成的总耗时
			"errors", c.Errors.String(), // 请求处理链写入 Gin 上下文的错误集合
		}

		// 根据响应状态码选择日志级别，使服务端错误、客户端错误和正常请求易于区分。
		switch status := c.Writer.Status(); {
		case status >= 500:
			go_utils.SugaredLogger.Errorw("http request", fields...)
		case status >= 400:
			go_utils.SugaredLogger.Warnw("http request", fields...)
		default:
			go_utils.SugaredLogger.Infow("http request", fields...)
		}
	}
}
