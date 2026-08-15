package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// taskListID 从当前 HTTP 请求的路由参数中解析任务清单 ID。
//
// 参数：
//   - c：Gin 请求上下文。路由必须包含名为 taskListId 的路径参数，
//     例如 /task-lists/:taskListId。
//
// 返回值：
//   - uint：解析成功的任务清单 ID，值一定大于 0；解析失败时返回 0。
//   - error：taskListId 是合法正整数时返回 nil；参数缺失、不是十进制整数、
//     超出当前平台 uint 的取值范围或值为 0 时返回 strconv.ErrSyntax。
func taskListID(c *gin.Context) (uint, error) {
	id, err := strconv.ParseUint(c.Param("taskListId"), 10, strconv.IntSize)
	if err != nil || id == 0 {
		return 0, strconv.ErrSyntax
	}
	return uint(id), nil
}
