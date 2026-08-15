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

// DeleteTask 删除当前登录用户的指定任务。
func (h *Handler) DeleteTask(c *gin.Context) {
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

	err = h.service.DeleteTask(claims.UserID, uint(id), c.Query("cascade") == "true")
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		response.Error(c, http.StatusNotFound, "任务不存在")
	case errors.Is(err, taskservice.ErrTaskHasChildren):
		response.Error(c, http.StatusConflict, "该任务包含下级任务，请确认级联删除")
	case err != nil:
		requestlog.Error(c, "删除任务失败", err, "task_id", id)
		response.Error(c, http.StatusInternalServerError, "删除任务失败")
	default:
		requestlog.Info(c, "删除任务成功", "task_id", id)
		c.JSON(http.StatusOK, gin.H{
			"code":    http.StatusOK,
			"message": "success",
		})
	}
}
