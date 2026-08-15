package repository

import (
	"gorm.io/gorm"

	"personal_assistant_server/internal/task"
)

// Repository 封装任务数据库操作。
type Repository struct {
	db *gorm.DB
}

// New 创建任务 Repository。
func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// CreateTask 将任务写入数据库。
func (r *Repository) CreateTask(item *task.Task) error {
	return r.db.Create(item).Error
}
