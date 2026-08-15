package service

import (
	"gorm.io/gorm"

	"personal_assistant_server/internal/task"
)

func (s *Service) DeleteTaskList(userID, id uint) error {
	var count int64
	if err := s.db.Model(&task.TaskList{}).Where("id = ? AND user_id = ?", id, userID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return gorm.ErrRecordNotFound
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if tx.Migrator().HasTable(&task.Task{}) {
			if err := tx.Where("user_id = ? AND list_id = ?", userID, id).Delete(&task.Task{}).Error; err != nil {
				return err
			}
		}
		return tx.Where("user_id = ? AND id = ?", userID, id).Delete(&task.TaskList{}).Error
	})
}
