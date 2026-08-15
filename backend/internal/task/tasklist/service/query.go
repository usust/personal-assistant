package service

import "personal_assistant_server/internal/task"

// ListTaskLists 查询指定用户拥有的全部任务清单。
//
// 参数：
//   - userID：任务清单所属用户的唯一标识。该值用于限定数据库查询范围，
//     确保不会返回其他用户的数据。
//
// 返回值：
//   - []task.TaskList：查询到的任务清单集合，按创建时间倒序排列；创建时间
//     相同时按 ID 倒序排列。没有匹配记录时返回已初始化的空切片，而不是 nil。
//   - error：查询成功时返回 nil；数据库执行查询失败时返回底层数据库错误。
func (s *Service) ListTaskLists(userID uint) ([]task.TaskList, error) {
	items := make([]task.TaskList, 0)
	err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Order("id DESC").Find(&items).Error
	return items, err
}

// GetTaskList 查询指定用户拥有的单个任务清单。
//
// 参数：
//   - userID：任务清单所属用户的唯一标识，用于执行数据归属校验。
//   - id：待查询任务清单的唯一标识。
//
// 返回值：
//   - *task.TaskList：查询成功时返回任务清单对象；查询失败时返回 nil。
//   - error：查询成功时返回 nil；记录不存在或不属于指定用户时返回
//     gorm.ErrRecordNotFound；其他查询失败场景返回底层数据库错误。
func (s *Service) GetTaskList(userID, id uint) (*task.TaskList, error) {
	var item task.TaskList
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}
