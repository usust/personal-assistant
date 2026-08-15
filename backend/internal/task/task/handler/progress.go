package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"personal_assistant_server/internal/middleware"
	"personal_assistant_server/internal/requestlog"
	"personal_assistant_server/internal/response"
	taskservice "personal_assistant_server/internal/task/task/service"
)

// updateProgressRequest 描述任务进度更新请求。
type updateProgressRequest struct {
	Operation string `json:"operation" binding:"required,oneof=increment decrement"`

	// AllowExceedTotal 表示增加完成量时是否允许结果超过 progress_total。
	AllowExceedTotal bool `json:"allowExceedTotal"`
}

// UpdateProgress 更新当前登录用户指定任务的进度。
func (h *Handler) UpdateProgress(c *gin.Context) {
	claims, ok := middleware.CurrentClaims(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "请先登录")
		return
	}

	id, err := strconv.ParseUint(c.Param("taskId"), 10, strconv.IntSize)
	if err != nil || id == 0 {
		response.Error(c, http.StatusBadRequest, "任务 ID 无效")
		return
	}

	var request updateProgressRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Error(c, http.StatusBadRequest, "进度操作无效")
		return
	}

	err = h.service.UpdateProgress(claims.UserID, uint(id), taskservice.UpdateProgressInput{
		Operation:        request.Operation,
		AllowExceedTotal: request.AllowExceedTotal,
	})
	switch {
	case errors.Is(err, taskservice.ErrInvalidProgress):
		response.Error(c, http.StatusBadRequest, "进度设置无效")
	case errors.Is(err, gorm.ErrRecordNotFound):
		response.Error(c, http.StatusNotFound, "任务不存在")
	case err != nil:
		requestlog.Error(c, "更新任务进度失败", err, "task_id", id)
		response.Error(c, http.StatusInternalServerError, "更新任务进度失败")
	default:
		requestlog.Info(c, "更新任务进度成功", "task_id", id)
		c.JSON(http.StatusOK, gin.H{
			"code":    http.StatusOK,
			"message": "success",
		})
	}
}
