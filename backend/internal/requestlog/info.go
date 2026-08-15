package requestlog

import (
	"github.com/gin-gonic/gin"

	"personal_assistant_server/internal/goutils"
)

// Info 记录 HTTP 请求处理过程中产生的一般业务信息。
//
// 公共的请求和用户信息由 fields 统一补充。调用方可以通过 extra 追加与具体业务
// 有关的结构化字段，参数格式与 Zap SugaredLogger.Infow 一致，即按照
// “字段名、字段值”的顺序成对传入。
//
// 参数：
//   - c：当前请求的 Gin 上下文，用于提取请求路径、客户端信息和认证用户信息。
//   - message：描述本次业务操作的简短日志消息。
//   - extra：可选的业务字段，按照“字段名、字段值”的顺序成对传入。
//
// 返回值：
//   - 无。
func Info(c *gin.Context, message string, extra ...any) {
	goutils.SugaredLogger.Infow(message, fields(c, extra...)...)
}
