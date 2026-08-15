package service

import "personal_assistant_server/internal/task"

func (s *Service) UpdateTaskList(userID, id uint, fields map[string]any) (*task.TaskList, error) {
	result := s.db.Model(&task.TaskList{}).Where("id = ? AND user_id = ?", id, userID).Updates(fields)
	if result.Error != nil {
		return nil, result.Error
	}
	return s.GetTaskList(userID, id)
}
