package service

import (
	"errors"
)

// ErrInvalidProgress 表示进度操作不符合约束。
var ErrInvalidProgress = errors.New("任务进度操作无效")

// UpdateProgressInput 描述任务的进度更新操作。
type UpdateProgressInput struct {
	// Operation 指定进度调整方向；increment 增加一个 step，decrement 减少一个 step。
	Operation string

	// AllowExceedTotal 表示增加完成量时是否允许结果超过 progress_total。
	AllowExceedTotal bool
}

// UpdateProgress 按任务自身的 step 增减 completed 字段。
//
// 函数只根据 Operation 选择增加或减少操作，具体数值由 Repository 直接使用
// 数据库中的 progress_step 原子更新 progress_completed。AllowExceedTotal 为
// false 时，增加后的结果最多为 progress_total；为 true 时允许超过总量。
// 减少后的结果始终不会低于 0。
//
// 参数：
//   - s：任务 Service，内部持有更新任务进度所需的 Repository。
//   - userID：当前登录用户的数据库 ID，用于限制更新的数据归属。
//   - id：需要调整进度的任务数据库 ID。
//   - input：进度操作参数，Operation 仅支持 increment 或 decrement；
//     AllowExceedTotal 控制增加操作是否允许超过 progress_total。
//
// 返回值：
//   - nil：progress_completed 已成功增加或减少一个 progress_step。
//   - ErrInvalidProgress：Operation 不是受支持的操作。
//   - 其他错误：Repository 更新数据库时返回的原始错误。
func (s *Service) UpdateProgress(userID, id uint, input UpdateProgressInput) error {
	// Service 只分发操作类型，不读取任务或计算进度数值。
	switch input.Operation {
	case "increment":
		return s.repository.IncrementProgressCompleted(userID, id, input.AllowExceedTotal)
	case "decrement":
		return s.repository.DecrementProgressCompleted(userID, id)
	default:
		return ErrInvalidProgress
	}
}
