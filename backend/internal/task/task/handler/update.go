package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"personal_assistant_server/internal/middleware"
	"personal_assistant_server/internal/requestlog"
	"personal_assistant_server/internal/response"
	taskservice "personal_assistant_server/internal/task/task/service"
)

// optionalParentID 记录 PATCH 请求是否显式提供了 parentId，并区分 null 与具体 ID。
// Present 为 false 表示保持原上级节点；Present 为 true 且 Value 为 nil 表示移到顶层。
type optionalParentID struct {
	Present bool
	Value   *uint
}

// UnmarshalJSON 解码 parentId，允许正整数或 null。
func (value *optionalParentID) UnmarshalJSON(data []byte) error {
	value.Present = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		value.Value = nil
		return nil
	}

	var id uint
	if err := json.Unmarshal(data, &id); err != nil {
		return err
	}
	if id == 0 {
		return errors.New("上级节点 ID 无效")
	}
	value.Value = &id
	return nil
}

// updateTaskRequest 描述任务信息的部分更新请求。
// 普通字段使用指针区分未提供字段和显式提交的零值；ParentID 还需要区分 null。
type updateTaskRequest struct {
	Title             *string          `json:"title"`
	Remark            *string          `json:"remark"`
	StartDate         *string          `json:"startDate"`
	StartTime         *string          `json:"startTime"`
	EndDate           *string          `json:"endDate"`
	EndTime           *string          `json:"endTime"`
	Priority          *string          `json:"priority"`
	Archived          *bool            `json:"archived"`
	ParentID          optionalParentID `json:"parentId"`
	TaskType          *string          `json:"taskType"`
	ProgressTotal     *string          `json:"progressTotal"`
	ProgressCompleted *string          `json:"progressCompleted"`
	ProgressStep      *string          `json:"progressStep"`
	ProgressUnit      *string          `json:"progressUnit"`
}

// UpdateTask 部分更新当前登录用户指定任务的信息。
//
// 函数从认证上下文中取得当前用户 ID，从路由参数 taskId 中解析任务 ID，并严格
// 解码请求体中的单个 JSON 对象。请求允许包含标题、备注、起止日期时间、优先级、
// 归档状态、上级节点、任务类型，以及子任务的进度总量、完成量、默认增量和单位；
// 未提供的字段保持原值。请求通过基础格式检查后，函数将字段指针传给 Service
// 完成具体校验和持久化。
//
// 参数：
//   - h：任务 HTTP 处理器，内部持有执行任务信息更新的任务 Service。
//   - c：Gin 请求上下文，包含认证声明、taskId 路由参数、JSON 请求体，并用于写入
//     最终 HTTP 响应。
//
// 返回值：
//   - 本函数没有 Go 返回值，处理结果直接写入 HTTP 响应。更新成功时返回 HTTP 200，
//     响应仅包含 code 和 message；未登录时返回 HTTP 401；任务 ID、JSON、字段名称
//     或任务信息、进度配置无效时返回 HTTP 400；任务不存在或不属于当前用户时返回 HTTP 404；
//     其他 Service 或数据库错误返回 HTTP 500。
func (h *Handler) UpdateTask(c *gin.Context) {
	// 从认证中间件写入的上下文中读取当前用户，避免客户端自行指定数据所属用户。
	claims, ok := middleware.CurrentClaims(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "请先登录")
		return
	}

	// taskId 必须是当前平台 uint 范围内的正十进制整数。
	id, err := strconv.ParseUint(c.Param("taskId"), 10, strconv.IntSize)
	if err != nil || id == 0 {
		response.Error(c, http.StatusBadRequest, "任务 ID 无效")
		return
	}

	// 严格解码任务信息：DisallowUnknownFields 会拒绝白名单外字段。
	var request updateTaskRequest
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		response.Error(c, http.StatusBadRequest, "请求数据格式错误或包含不支持的字段")
		return
	}
	// 首个 JSON 对象之后必须直接到达请求体末尾，防止一次请求携带多个 JSON 值。
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		response.Error(c, http.StatusBadRequest, "请求数据格式错误")
		return
	}

	// 保留各字段的指针状态传入 Service，使其能够区分“未提供”与显式提交空值或 false。
	err = h.service.UpdateTask(claims.UserID, uint(id), taskservice.UpdateTaskInput{
		Title:             request.Title,
		Remark:            request.Remark,
		StartDate:         request.StartDate,
		StartTime:         request.StartTime,
		EndDate:           request.EndDate,
		EndTime:           request.EndTime,
		Priority:          request.Priority,
		Archived:          request.Archived,
		ParentID:          request.ParentID.Value,
		ParentIDSet:       request.ParentID.Present,
		TaskType:          request.TaskType,
		ProgressTotal:     request.ProgressTotal,
		ProgressCompleted: request.ProgressCompleted,
		ProgressStep:      request.ProgressStep,
		ProgressUnit:      request.ProgressUnit,
	})
	// 将 Service 返回的业务错误和数据库错误转换为稳定的 HTTP 状态码与中文提示。
	switch {
	case errors.Is(err, taskservice.ErrInvalidTask):
		response.Error(c, http.StatusBadRequest, "任务基础信息无效")
	case errors.Is(err, taskservice.ErrInvalidProgress):
		response.Error(c, http.StatusBadRequest, "进度设置无效")
	case errors.Is(err, taskservice.ErrParentNotFound):
		response.Error(c, http.StatusBadRequest, "上级节点不存在或不属于当前清单")
	case errors.Is(err, taskservice.ErrInvalidParent):
		response.Error(c, http.StatusBadRequest, "不能将任务自身或其下级节点设为上级节点")
	case errors.Is(err, gorm.ErrRecordNotFound):
		response.Error(c, http.StatusNotFound, "任务不存在")
	case err != nil:
		requestlog.Error(c, "更新任务失败", err, "task_id", id)
		response.Error(c, http.StatusInternalServerError, "更新任务失败")
	default:
		// 更新成功只返回结果码和消息，不返回 data，前端随后重新查询最新任务列表。
		requestlog.Info(c, "更新任务成功", "task_id", id)
		c.JSON(http.StatusOK, gin.H{
			"code":    http.StatusOK,
			"message": "success",
		})
	}
}
