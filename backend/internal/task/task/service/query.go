package service

import "personal_assistant_server/internal/task"

// ListTasks 查询并直接返回指定用户的任务数据库模型列表。
func (s *Service) ListTasks(userID uint) ([]task.Task, error) {
	return s.repository.ListTasks(userID)
}
