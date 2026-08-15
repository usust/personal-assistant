package service

import "personal_assistant_server/internal/task"

func (s *Service) CreateTaskList(userID uint, name, remark, color, icon string) (*task.TaskList, error) {
	item := &task.TaskList{UserID: userID, Name: name, Remark: remark, Color: color, Icon: icon}
	if err := s.db.Create(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}
