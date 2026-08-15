package repository

import (
	"gorm.io/gorm"

	"personal_assistant_server/internal/task"
)

// UpdateTask 更新指定用户任务中由 fields 明确列出的信息字段。
func (r *Repository) UpdateTask(userID, id uint, item *task.Task, fields []string) error {
	var current task.Task
	if err := r.db.Select("id").Where("user_id = ? AND id = ?", userID, id).First(&current).Error; err != nil {
		return err
	}

	result := r.db.Model(&task.Task{}).
		Where("user_id = ? AND id = ?", userID, id).
		Select(fields).
		Updates(item)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		var count int64
		if err := r.db.Model(&task.Task{}).Where("user_id = ? AND id = ?", userID, id).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return gorm.ErrRecordNotFound
		}
	}
	return nil
}
