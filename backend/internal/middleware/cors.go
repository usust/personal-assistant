package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS 创建用于处理浏览器跨域资源共享请求的 Gin 中间件。
//
// 函数首先将允许访问服务的来源列表转换为查询效率更高的集合。每次收到请求时，
// 中间件读取 Origin 请求头并执行精确匹配；来源在白名单中时，响应将携带允许来源、
// 缓存区分和凭证访问相关的 CORS 响应头。无论来源是否匹配，中间件都会声明服务允许的
// 请求头和 HTTP 方法。
//
// 对于浏览器发送的 OPTIONS 预检请求，中间件直接返回 204 No Content 并终止后续
// 处理链；其他请求则通过 c.Next 继续交给后续中间件和路由处理函数。来源不在白名单
// 中时，中间件不会主动返回错误，但不会设置 Access-Control-Allow-Origin，浏览器因此
// 不会向前端页面开放该跨域响应。
//
// 参数：
//   - allowedOrigins：允许跨域访问服务的来源列表，例如 http://localhost:10080；
//     每个来源在加入白名单前都会去除首尾空白，并按照完整字符串进行精确匹配。
//
// 返回值：
//   - gin.HandlerFunc：负责设置 CORS 响应头和处理 OPTIONS 预检请求的 Gin 中间件函数。
func CORS(allowedOrigins []string) gin.HandlerFunc {
	// 将来源列表转换为集合，使每次请求可以通过常数时间查询判断来源是否被允许。
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		// 去除配置项首尾空白，避免格式问题造成合法来源匹配失败。
		allowed[strings.TrimSpace(origin)] = true
	}

	return func(c *gin.Context) {
		// Origin 表示发起跨域请求的页面来源，通常由协议、域名和端口组成。
		origin := c.GetHeader("Origin")

		// 仅对白名单中的来源返回允许跨域访问和携带身份凭证的响应头。
		if allowed[origin] {
			c.Header("Access-Control-Allow-Origin", origin)      // 将当前合法来源返回给浏览器
			c.Header("Vary", "Origin")                           // 提醒缓存按照 Origin 分别保存响应
			c.Header("Access-Control-Allow-Credentials", "true") // 允许跨域请求携带 Cookie 等凭证
		}

		// 声明跨域请求可以携带的请求头以及可以使用的 HTTP 方法。
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		// OPTIONS 请求属于浏览器预检请求，只需返回 CORS 响应头，无需进入业务路由。
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		// 非预检请求继续执行后续中间件和最终路由处理函数。
		c.Next()
	}
}
