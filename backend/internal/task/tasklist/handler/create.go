package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"personal_assistant_server/internal/middleware"
	"personal_assistant_server/internal/requestlog"
	"personal_assistant_server/internal/response"
)

type createTaskListRequest struct {
	Name   string `json:"name" binding:"required,max=128"`
	Remark string `json:"remark" binding:"max=2000"`
	Color  string `json:"color" binding:"required,hexcolor,max=20"`
	Icon   string `json:"icon" binding:"required,max=100"`
}

func (h *Handler) CreateTaskList(c *gin.Context) {
	claims, ok := middleware.CurrentClaims(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "请先登录")
		return
	}
	var request createTaskListRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Error(c, http.StatusBadRequest, "任务清单信息无效")
		return
	}
	request.Name = strings.TrimSpace(request.Name)
	request.Remark = strings.TrimSpace(request.Remark)
	request.Color = strings.TrimSpace(request.Color)
	request.Icon = strings.TrimSpace(request.Icon)
	if request.Name == "" {
		response.Error(c, http.StatusBadRequest, "任务清单名称不能为空")
		return
	}
	if request.Icon == "" {
		response.Error(c, http.StatusBadRequest, "请选择任务清单图标")
		return
	}
	item, err := h.service.CreateTaskList(claims.UserID, request.Name, request.Remark, request.Color, request.Icon)
	if err != nil {
		requestlog.Error(c, "创建任务清单失败", err)
		response.Error(c, http.StatusInternalServerError, "创建任务清单失败")
		return
	}
	requestlog.Info(c, "创建任务清单成功", "task_list_id", item.ID)
	c.JSON(http.StatusCreated, response.Body{Code: http.StatusCreated, Message: "success", Data: item})
}
