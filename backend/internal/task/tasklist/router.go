package tasklist

import (
	"github.com/gin-gonic/gin"

	tasklisthandler "personal_assistant_server/internal/task/tasklist/handler"
)

// RegisterRoutes 注册任务清单资源相关的 HTTP 路由。
//
// 参数：
//   - router：任务清单路由所挂载的父路由组。通常由全局路由模块传入已经注册
//     身份认证中间件的 /api 路由组，因此本函数声明的所有接口都要求用户登录。
//   - handler：任务清单 HTTP Handler，负责解析请求、调用 Service 并写入响应。
//
// 本函数不返回值。所有路由统一使用 /task-lists 前缀，并继承父路由组上的
// 中间件与配置。
func RegisterRoutes(router *gin.RouterGroup, handler *tasklisthandler.Handler) {
	// 创建任务清单子路由组。
	// 若传入的父路由组路径为 /api，则本组的完整基础路径为 /api/task-lists。
	taskLists := router.Group("/task-lists")

	// GET /api/task-lists
	// 查询当前登录用户拥有的全部任务清单。成功时返回任务清单数组；没有数据时
	// 返回空数组。用户身份由父路由组上的认证中间件提供。
	taskLists.GET("", handler.ListTaskLists)

	// POST /api/task-lists
	// 为当前登录用户创建任务清单。请求体包含 name、color 和 icon，
	// 并可选提供 remark 记录任务实施信息。成功时使用 HTTP 201 状态码。
	taskLists.POST("", handler.CreateTaskList)

	// PATCH /api/task-lists/:taskListId
	// 部分更新指定任务清单。请求体可以包含 name、remark、color 或 icon；未提供的字段
	// 保持原值。仅允许当前登录用户更新自己拥有的任务清单。
	taskLists.PATCH("/:taskListId", handler.UpdateTaskList)

	// DELETE /api/task-lists/:taskListId
	// 删除指定任务清单，同时由 Service 在事务中删除该清单关联的任务。
	// 删除成功时返回 HTTP 204，不包含响应体。
	taskLists.DELETE("/:taskListId", handler.DeleteTaskList)
}
