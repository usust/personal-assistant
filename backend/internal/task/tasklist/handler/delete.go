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

func (h *Handler) DeleteTaskList(c *gin.Context) {
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
	err = h.service.DeleteTaskList(claims.UserID, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Error(c, http.StatusNotFound, "任务清单不存在")
		return
	}
	if err != nil {
		requestlog.Error(c, "删除任务清单失败", err, "task_list_id", id)
		response.Error(c, http.StatusInternalServerError, "删除任务清单失败")
		return
	}
	requestlog.Info(c, "删除任务清单成功", "task_list_id", id)
	response.OK(c, nil)
}
