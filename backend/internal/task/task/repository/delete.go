package repository

import (
	"errors"

	"personal_assistant_server/internal/task"
)

// ErrTaskHasChildren 表示任务包含下级任务，不能直接删除。
var ErrTaskHasChildren = errors.New("task has children")

// DeleteTask 删除指定用户拥有的任务。
//
// 函数在一个数据库事务中验证任务归属、读取当前用户的任务层级，并根据 ParentID
// 构建父子关系。目标任务没有下级任务时直接删除；目标任务包含下级任务且 cascade
// 为 false 时拒绝删除；cascade 为 true 时使用广度优先遍历收集全部后代任务，
// 最后一次性删除目标任务及其后代。所有查询和删除操作均使用 userID 限定数据范围。
//
// 参数：
//   - userID：当前用户的数据库 ID，用于验证任务归属并隔离其他用户的数据。
//   - id：待删除任务的数据库 ID；任务不存在或不属于 userID 时按不存在处理。
//   - cascade：是否允许级联删除。为 false 且任务包含下级任务时拒绝删除；
//     为 true 时同时删除目标任务的全部后代任务。
//
// 返回值：
//   - error：事务提交成功时返回 nil；任务不存在或不属于当前用户时返回
//     gorm.ErrRecordNotFound；任务包含下级任务但未启用级联删除时返回
//     ErrTaskHasChildren；开启事务、查询任务、删除记录或提交事务失败时返回
//     对应的数据库错误。提交前发生任何错误时，延迟执行的 Rollback 会回滚事务。
func (r *Repository) DeleteTask(userID, id uint, cascade bool) error {
	// 显式开启事务，使任务校验、层级读取和批量删除处于同一事务边界内。
	tx := r.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	// 任一后续步骤提前返回时回滚事务；事务成功提交后再次回滚不会改变已提交结果。
	defer tx.Rollback()

	// 同时使用 userID 和任务 ID 查询目标任务，防止用户删除不属于自己的任务。
	var item task.Task
	if err := tx.Where("user_id = ? AND id = ?", userID, id).First(&item).Error; err != nil {
		return err
	}

	// 仅查询构建层级关系所需的 ID 和 ParentID，避免读取无关的任务详情字段。
	var items []task.Task
	if err := tx.Select("id", "parent_id").Where("user_id = ?", userID).Find(&items).Error; err != nil {
		return err
	}

	// 将任务列表整理为“父任务 ID -> 直属下级任务 ID”映射，并在未确认级联删除时
	// 阻止删除包含下级任务的任务，避免产生失去上级任务的孤立记录。
	children := make(map[uint][]uint)
	for _, current := range items {
		if current.ParentID != nil {
			children[*current.ParentID] = append(children[*current.ParentID], current.ID)
		}
	}
	if len(children[id]) > 0 && !cascade {
		return ErrTaskHasChildren
	}

	// 从目标任务的直属下级任务开始进行广度优先遍历，收集需要删除的全部任务 ID。
	// visited 防止异常的循环层级关系导致重复入队或无限遍历。
	ids := []uint{id}
	visited := map[uint]bool{id: true}
	for queue := append([]uint(nil), children[id]...); len(queue) > 0; {
		currentID := queue[0]
		queue = queue[1:]
		if visited[currentID] {
			continue
		}
		visited[currentID] = true
		ids = append(ids, currentID)
		queue = append(queue, children[currentID]...)
	}

	// 在当前用户范围内批量删除目标任务及其后代；删除成功后提交整个事务。
	if err := tx.Where("user_id = ? AND id IN ?", userID, ids).Delete(&task.Task{}).Error; err != nil {
		return err
	}
	return tx.Commit().Error
}
