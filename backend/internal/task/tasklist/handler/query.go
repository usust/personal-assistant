package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"personal_assistant_server/internal/middleware"
	"personal_assistant_server/internal/requestlog"
	"personal_assistant_server/internal/response"
)

// ListTaskLists 查询当前登录用户拥有的全部任务清单。
//
// 参数：
//   - c：Gin 请求上下文。认证中间件应当提前将当前用户的身份声明写入上下文；
//     Handler 使用声明中的 UserID 限制查询范围，防止读取其他用户的任务清单。
//
// 返回值：
//   - 该方法不返回 Go 函数值，而是直接通过 c 写入 HTTP 响应。
//   - 200 OK：查询成功，响应 data 为任务清单数组；没有数据时返回空数组。
//   - 401 Unauthorized：上下文中不存在有效的用户身份声明。
//   - 500 Internal Server Error：数据库查询失败。
func (h *Handler) ListTaskLists(c *gin.Context) {
	claims, ok := middleware.CurrentClaims(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "请先登录")
		return
	}
	items, err := h.service.ListTaskLists(claims.UserID)
	if err != nil {
		requestlog.Error(c, "查询任务清单失败", err)
		response.Error(c, http.StatusInternalServerError, "查询任务清单失败")
		return
	}
	requestlog.Info(c, "查询任务清单成功", "list_count", len(items))
	response.OK(c, items)
}

// GetTaskList 查询当前登录用户拥有的指定任务清单。
//
// 参数：
//   - c：Gin 请求上下文。路由必须包含名为 taskListId 的路径参数，认证中间件
//     还应当提前将当前用户的身份声明写入上下文。查询同时使用任务清单 ID 和
//     UserID 作为条件，以确保用户只能访问自己的任务清单。
//
// 返回值：
//   - 该方法不返回 Go 函数值，而是直接通过 c 写入 HTTP 响应。
//   - 200 OK：查询成功，响应 data 为对应的任务清单对象。
//   - 400 Bad Request：taskListId 缺失、不是合法正整数或超出 uint 取值范围。
//   - 401 Unauthorized：上下文中不存在有效的用户身份声明。
//   - 404 Not Found：任务清单不存在或不属于当前用户。
//   - 500 Internal Server Error：数据库查询过程中发生其他错误。
func (h *Handler) GetTaskList(c *gin.Context) {
	claims, ok := middleware.CurrentClaims(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "请先登录")
		return
	}
	id, err := taskListID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "任务清单 ID 无效")
		return
	}
	item, err := h.service.GetTaskList(claims.UserID, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Error(c, http.StatusNotFound, "任务清单不存在")
		return
	}
	if err != nil {
		requestlog.Error(c, "查询任务清单详情失败", err, "task_list_id", id)
		response.Error(c, http.StatusInternalServerError, "查询任务清单详情失败")
		return
	}
	response.OK(c, item)
}
