package requestlog

import (
	"github.com/gin-gonic/gin"

	"personal_assistant_server/internal/goutils"
)

// Error 记录 HTTP 请求处理过程中发生的服务端错误。
//
// err 作为 error 字段记录，公共的请求和用户信息由 fields 统一补充。调用方可以
// 通过 extra 追加与具体业务有关的结构化字段，参数格式与 Zap SugaredLogger.Errorw
// 一致，即按照“字段名、字段值”的顺序成对传入。
func Error(c *gin.Context, message string, err error, extra ...any) {
	errorFields := make([]any, 0, len(extra)+2)
	errorFields = append(errorFields, "error", err)
	errorFields = append(errorFields, extra...)

	goutils.SugaredLogger.Errorw(message, fields(c, errorFields...)...)
}
