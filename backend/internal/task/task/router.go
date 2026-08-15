package task

import (
	"github.com/gin-gonic/gin"

	taskhandler "personal_assistant_server/internal/task/task/handler"
)

// RegisterRoutes 注册任务资源相关的 HTTP 路由。
//
// 参数：
//   - router：任务路由挂载的父路由组。当前由全局路由模块传入已启用
//     JWT 认证中间件的 /api 路由组，因此本函数注册的接口均需要登录。
//   - handler：任务 HTTP 处理器，负责验证请求、调用任务 Service 并写入响应。
//
// 本函数没有返回值。所有路由统一使用 /tasks 资源前缀，并通过认证声明中的
// UserID 限制数据范围，确保用户只能访问和修改自己的任务。
func RegisterRoutes(router *gin.RouterGroup, handler *taskhandler.Handler) {
	// 创建任务资源子路由组。若父路由组的路径为 /api，则下列接口的
	// 完整基础路径为 /api/tasks。
	tasks := router.Group("/tasks")

	// GET /api/tasks
	// 查询当前登录用户的全部层级任务，按创建时间和 ID 升序返回。
	// 响应直接返回任务数据库模型列表，其中进度字段保持数据库中保存的原始数值；
	// 任务树、进度格式化和后代叶子任务汇总均由前端构建。没有任务时返回空数组。
	tasks.GET("", handler.ListTasks)

	// POST /api/tasks
	// 为当前用户创建顶层任务或下级任务。请求体必须提供 title 和 listId，可选
	// 提供备注、起止时间和优先级；taskType 独立指定主任务或子任务，提供 parentId
	// 时把任务挂到指定任务下面，主任务同样可以作为另一个任务的下级任务。
	// 只有子任务保存进度总量、完成量、默认增量和单位；主任务的进度字段保持为空。
	// 上级任务必须属于同一用户和同一清单，
	// 任意层级的任务都可以继续添加下级任务。成功时返回 HTTP 201，响应不包含 data 字段；
	// 客户端随后通过 GET /api/tasks 查询包含层级汇总信息的最新任务列表。
	tasks.POST("", handler.CreateTask)

	// DELETE /api/tasks/:taskId
	// 删除当前用户的指定任务。若任务包含下级任务，默认返回 HTTP 409；
	// 显式传入 cascade=true 时，在同一事务中递归删除任务及其全部后代。
	tasks.DELETE("/:taskId", handler.DeleteTask)

	// PATCH /api/tasks/:taskId
	// 仅更新请求实际提供的任务信息，包括标题、备注、起止日期时间、优先级、
	// 归档状态、上级节点、任务类型和子任务进度配置。parentId 为 null 时把任务
	// 移到顶层；调整层级时不能选择当前任务自身或其任意后代，也不能跨用户或跨
	// 清单选择上级节点。修改为主任务时清空自身进度配置；子任务的单个进度字段
	// 可独立修改，未提供的进度字段保持原值。未知字段或字段值无效时返回 HTTP 400。
	tasks.PATCH("/:taskId", handler.UpdateTask)

	// PATCH /api/tasks/:taskId/progress
	// 修改指定任务的执行进度，taskId 必须属于当前用户。operation 为 increment
	// 时按任务已有 step 增加 completed，为 decrement 时按 step 减少 completed。
	// increment 可通过 allowExceedTotal 指定是否允许 completed 超过 total；
	// 该参数未提供或为 false 时限制为 total。decrement 始终不会低于 0。
	// 接口只更新 completed，不修改其他任务字段。
	// 更新成功后客户端重新查询任务列表，并在前端汇总全部祖先进度。
	tasks.PATCH("/:taskId/progress", handler.UpdateProgress)
}
