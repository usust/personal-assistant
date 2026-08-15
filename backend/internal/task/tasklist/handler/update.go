package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"personal_assistant_server/internal/middleware"
	"personal_assistant_server/internal/requestlog"
	"personal_assistant_server/internal/response"
)

type updateTaskListRequest struct {
	Name   *string `json:"name" binding:"omitempty,max=128"`
	Remark *string `json:"remark" binding:"omitempty,max=2000"`
	Color  *string `json:"color" binding:"omitempty,hexcolor,max=20"`
	Icon   *string `json:"icon" binding:"omitempty,max=100"`
}

func (h *Handler) UpdateTaskList(c *gin.Context) {
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
	var request updateTaskListRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Error(c, http.StatusBadRequest, "任务清单信息无效")
		return
	}
	fields := make(map[string]any, 4)
	if request.Name != nil {
		name := strings.TrimSpace(*request.Name)
		if name == "" {
			response.Error(c, http.StatusBadRequest, "任务清单名称不能为空")
			return
		}
		fields["name"] = name
	}
	if request.Remark != nil {
		fields["remark"] = strings.TrimSpace(*request.Remark)
	}
	if request.Color != nil {
		fields["color"] = strings.TrimSpace(*request.Color)
	}
	if request.Icon != nil {
		icon := strings.TrimSpace(*request.Icon)
		if icon == "" {
			response.Error(c, http.StatusBadRequest, "任务清单图标不能为空")
			return
		}
		fields["icon"] = icon
	}
	if len(fields) == 0 {
		response.Error(c, http.StatusBadRequest, "请提供需要更新的字段")
		return
	}
	item, err := h.service.UpdateTaskList(claims.UserID, id, fields)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Error(c, http.StatusNotFound, "任务清单不存在")
		return
	}
	if err != nil {
		requestlog.Error(c, "更新任务清单失败", err, "task_list_id", id)
		response.Error(c, http.StatusInternalServerError, "更新任务清单失败")
		return
	}
	requestlog.Info(c, "更新任务清单成功", "task_list_id", id)
	response.OK(c, item)
}
