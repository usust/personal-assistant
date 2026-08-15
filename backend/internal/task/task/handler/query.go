package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"personal_assistant_server/internal/middleware"
	"personal_assistant_server/internal/requestlog"
	"personal_assistant_server/internal/response"
)

// ListTasks 查询当前登录用户拥有的全部任务。
func (h *Handler) ListTasks(c *gin.Context) {
	claims, ok := middleware.CurrentClaims(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "请先登录")
		return
	}

	items, err := h.service.ListTasks(claims.UserID)
	if err != nil {
		requestlog.Error(c, "查询任务失败", err)
		response.Error(c, http.StatusInternalServerError, "查询任务失败")
		return
	}

	requestlog.Info(c, "查询任务成功", "task_count", len(items))
	response.OK(c, items)
}
