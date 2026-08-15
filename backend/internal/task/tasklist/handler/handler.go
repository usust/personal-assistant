package handler

import tasklistservice "personal_assistant_server/internal/task/tasklist/service"

type Handler struct{ service *tasklistservice.Service }

func New(service *tasklistservice.Service) *Handler { return &Handler{service: service} }
