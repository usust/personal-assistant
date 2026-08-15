package service

import taskrepository "personal_assistant_server/internal/task/task/repository"

// ErrTaskHasChildren 表示任务包含下级任务，不能直接删除。
var ErrTaskHasChildren = taskrepository.ErrTaskHasChildren

// DeleteTask 删除指定用户的任务；cascade 为 true 时同时删除全部后代任务。
func (s *Service) DeleteTask(userID, id uint, cascade bool) error {
	return s.repository.DeleteTask(userID, id, cascade)
}
