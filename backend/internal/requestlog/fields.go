// Package requestlog provides structured logging helpers for HTTP requests.
package requestlog

import (
	"github.com/gin-gonic/gin"

	"personal_assistant_server/internal/middleware"
)

// fields 收集各日志等级共同使用的 HTTP 请求上下文字段。
//
// 函数统一收集请求方法、实际路径、路由模板、查询参数、客户端 IP 和 User-Agent；
// 如果请求已经通过认证，还会追加令牌声明中的用户 ID、用户名和角色。extra 用于
// 追加调用位置特有的业务字段，参数应按照“字段名、字段值”的顺序成对传入。
//
// 本函数不会收集请求头、Cookie、请求体或访问令牌，避免敏感信息进入日志。
func fields(c *gin.Context, extra ...any) []any {
	logFields := []any{
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"route", c.FullPath(),
		"query", c.Request.URL.RawQuery,
		"client_ip", c.ClientIP(),
		"user_agent", c.Request.UserAgent(),
	}

	// 认证声明并非所有接口都存在，因此仅在中间件已经写入有效声明时追加用户信息。
	if claims, ok := middleware.CurrentClaims(c); ok {
		logFields = append(logFields,
			"user_id", claims.UserID,
			"username", claims.Username,
			"role", claims.Role,
		)
	}

	return append(logFields, extra...)
}
