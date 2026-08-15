package repository

import (
	"gorm.io/gorm"

	"personal_assistant_server/internal/task"
)

// IncrementProgressCompleted 将指定任务的 completed 增加一个 step。
//
// 该方法使用一条 UPDATE 语句，直接读取任务当前保存的 progress_completed、
// progress_step 和 progress_total，并将 progress_completed 增加一个 step。
// allowExceedTotal 为 false 时，如果增加后的数值达到或超过 total，则把 completed
// 设置为 total；为 true 时直接增加一个 step，允许 completed 超过 total。
// 更新条件同时限制任务所属用户、任务 ID，并要求三个进度字段均不为 NULL。
//
// 参数：
//   - r：任务 Repository，内部持有执行更新操作的数据库连接。
//   - userID：任务所属用户的数据库 ID，用于防止更新其他用户的任务。
//   - id：需要增加完成量的任务数据库 ID。
//   - allowExceedTotal：是否允许增加后的 completed 超过 progress_total。
//
// 返回值：
//   - nil：progress_completed 已成功更新。
//   - gorm.ErrRecordNotFound：没有匹配指定用户、任务 ID 和进度字段条件的任务。
//   - 其他错误：数据库执行 UPDATE 时返回的原始错误。
func (r *Repository) IncrementProgressCompleted(userID, id uint, allowExceedTotal bool) error {
	// 默认使用 CASE 将增加后的 completed 限制在 progress_total 以内。
	completedExpression := gorm.Expr(`
		CASE
			WHEN progress_completed + progress_step >= progress_total THEN progress_total
			ELSE progress_completed + progress_step
		END
	`)
	if allowExceedTotal {
		// 允许超过总量时直接增加 progress_step，不执行 progress_total 上限判断。
		completedExpression = gorm.Expr("progress_completed + progress_step")
	}

	// 只匹配当前用户的指定任务，并排除没有完整进度字段的任务。
	result := r.db.Model(&task.Task{}).
		Where("user_id = ? AND id = ?", userID, id).
		Where("progress_total IS NOT NULL AND progress_completed IS NOT NULL AND progress_step IS NOT NULL").
		// 只更新 progress_completed，不修改其他任务字段。
		UpdateColumn("progress_completed", completedExpression)
	return progressUpdateError(result)
}

// DecrementProgressCompleted 将指定任务的 completed 减少一个 step。
//
// 该方法使用一条 UPDATE 语句，直接读取任务当前保存的 progress_completed 和
// progress_step，并将 progress_completed 减少一个 step。如果减少后的数值小于
// 或等于 0，则把 completed 设置为 0。更新条件同时限制任务所属用户、任务 ID，
// 并要求三个进度字段均不为 NULL。
//
// 参数：
//   - r：任务 Repository，内部持有执行更新操作的数据库连接。
//   - userID：任务所属用户的数据库 ID，用于防止更新其他用户的任务。
//   - id：需要减少完成量的任务数据库 ID。
//
// 返回值：
//   - nil：progress_completed 已成功更新。
//   - gorm.ErrRecordNotFound：没有匹配指定用户、任务 ID 和进度字段条件的任务。
//   - 其他错误：数据库执行 UPDATE 时返回的原始错误。
func (r *Repository) DecrementProgressCompleted(userID, id uint) error {
	// 只匹配当前用户的指定任务，并排除没有完整进度字段的任务。
	result := r.db.Model(&task.Task{}).
		Where("user_id = ? AND id = ?", userID, id).
		Where("progress_total IS NOT NULL AND progress_completed IS NOT NULL AND progress_step IS NOT NULL").
		// 只更新 progress_completed；CASE 保证减少后的数值不会低于 0。
		UpdateColumn("progress_completed", gorm.Expr(`
			CASE
				WHEN progress_completed - progress_step <= 0 THEN 0
				ELSE progress_completed - progress_step
			END
		`))
	return progressUpdateError(result)
}

// progressUpdateError 返回进度更新结果中的数据库错误，并将未匹配任务转换为记录不存在。
//
// 参数：
//   - result：GORM 执行进度 UPDATE 后返回的数据库结果，包含错误和受影响行数。
//
// 返回值：
//   - nil：UPDATE 执行成功且至少更新了一条记录。
//   - gorm.ErrRecordNotFound：UPDATE 执行成功但没有记录符合更新条件。
//   - 其他错误：result 中保存的数据库原始错误。
func progressUpdateError(result *gorm.DB) error {
	// 数据库执行失败时优先返回原始错误，供上层记录和转换响应。
	if result.Error != nil {
		return result.Error
	}

	// 没有记录满足用户、任务 ID 或进度字段条件时，统一按记录不存在处理。
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
