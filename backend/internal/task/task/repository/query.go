package repository

import "personal_assistant_server/internal/task"

// ListTasks 查询指定用户的全部任务，并按创建时间和 ID 升序排列。
func (r *Repository) ListTasks(userID uint) ([]task.Task, error) {
	items := make([]task.Task, 0)
	err := r.db.
		Where("user_id = ?", userID).
		Order("created_at ASC").
		Order("id ASC").
		Find(&items).
		Error
	return items, err
}
