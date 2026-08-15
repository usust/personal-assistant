package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"personal_assistant_server/internal/middleware"
	"personal_assistant_server/internal/requestlog"
	"personal_assistant_server/internal/response"
	taskservice "personal_assistant_server/internal/task/task/service"
)

// Handler 负责处理任务 HTTP 请求。
type Handler struct {
	service *taskservice.Service
}

// New 创建任务 Handler。
func New(service *taskservice.Service) *Handler {
	return &Handler{service: service}
}

type createTaskRequest struct {
	Title             string  `json:"title"`
	Remark            string  `json:"remark"`
	ListID            uint    `json:"listId"`
	ParentID          *uint   `json:"parentId"`
	TaskType          string  `json:"taskType"`
	StartDate         string  `json:"startDate"`
	StartTime         string  `json:"startTime"`
	EndDate           string  `json:"endDate"`
	EndTime           string  `json:"endTime"`
	Priority          string  `json:"priority"`
	ProgressTotal     *string `json:"progressTotal"`
	ProgressCompleted *string `json:"progressCompleted"`
	ProgressStep      *string `json:"progressStep"`
	ProgressUnit      string  `json:"progressUnit"`
}

// CreateTask 创建当前登录用户的任务。创建成功时返回 HTTP 201，响应不包含 data 字段。
func (h *Handler) CreateTask(c *gin.Context) {
	claims, ok := middleware.CurrentClaims(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "请先登录")
		return
	}

	var request createTaskRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Error(c, http.StatusBadRequest, "请求数据格式错误")
		return
	}

	err := h.service.CreateTask(claims.UserID, taskservice.CreateTaskInput{
		Title:             request.Title,
		Remark:            request.Remark,
		ListID:            request.ListID,
		ParentID:          request.ParentID,
		TaskType:          request.TaskType,
		StartDate:         request.StartDate,
		StartTime:         request.StartTime,
		EndDate:           request.EndDate,
		EndTime:           request.EndTime,
		Priority:          request.Priority,
		ProgressTotal:     request.ProgressTotal,
		ProgressCompleted: request.ProgressCompleted,
		ProgressStep:      request.ProgressStep,
		ProgressUnit:      request.ProgressUnit,
	})
	switch {
	case errors.Is(err, taskservice.ErrInvalidTask):
		response.Error(c, http.StatusBadRequest, "任务基础信息无效")
	case errors.Is(err, taskservice.ErrParentNotFound):
		response.Error(c, http.StatusBadRequest, "上级节点不存在或不属于当前清单")
	case errors.Is(err, taskservice.ErrInvalidParent):
		response.Error(c, http.StatusBadRequest, "上级节点的层级关系无效")
	case err != nil:
		requestlog.Error(c, "创建任务失败", err)
		response.Error(c, http.StatusInternalServerError, "创建任务失败")
	default:
		requestlog.Info(c, "创建任务成功")
		c.JSON(http.StatusCreated, gin.H{
			"code":    http.StatusCreated,
			"message": "success",
		})
	}
}
