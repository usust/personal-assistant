package service

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"personal_assistant_server/internal/task"
	taskrepository "personal_assistant_server/internal/task/task/repository"
)

const (
	defaultProgressTotal float64 = 100
	defaultProgressStep  float64 = 1
)

var (
	// ErrParentNotFound 表示上级节点不存在、不属于当前用户或不在当前任务清单中。
	ErrParentNotFound = errors.New("上级节点不存在或不属于当前清单")
	// ErrInvalidParent 表示上级节点选择会形成循环或其既有层级关系已经无效。
	ErrInvalidParent = errors.New("上级节点层级关系无效")
)

// Service 实现任务业务操作。
type Service struct {
	repository *taskrepository.Repository
}

// New 创建任务 Service。
func New(db *gorm.DB) *Service {
	return &Service{repository: taskrepository.New(db)}
}

// CreateTaskInput 是创建任务时由 Handler 传递给 Service 的输入参数。
//
// 该结构体集中描述任务的基础信息、层级关系、起止时间、优先级和进度配置，
// 不包含数据库自动生成的 ID、创建时间和更新时间。文本字段的规范化及可选
// 进度值的默认处理由 CreateTask 统一完成。
type CreateTaskInput struct {
	// Title 是任务标题；创建前会去除首尾空白。
	Title string

	// Remark 是任务备注；创建前会去除首尾空白。
	Remark string

	// ListID 是任务所属清单的数据库 ID。
	ListID uint

	// ParentID 是上级任务 ID；nil 表示任务没有上级任务。
	ParentID *uint

	// TaskType 是任务类型；去除首尾空白后为空时默认使用 main。
	TaskType string

	// StartDate 是任务开始日期，由 Service 原样写入数据库。
	StartDate string

	// StartTime 是任务开始时间，由 Service 原样写入数据库。
	StartTime string

	// EndDate 是任务结束日期，由 Service 原样写入数据库。
	EndDate string

	// EndTime 是任务结束时间，由 Service 原样写入数据库。
	EndTime string

	// Priority 是任务优先级；去除首尾空白后为空时默认使用 medium。
	Priority string

	// ProgressTotal 是子任务的进度总量；创建主任务时忽略该字段。
	ProgressTotal *string

	// ProgressCompleted 是子任务的已完成量；创建主任务时忽略该字段。
	ProgressCompleted *string

	// ProgressStep 是子任务的默认进度增量；创建主任务时忽略该字段。
	ProgressStep *string

	// ProgressUnit 是子任务的进度计量单位；创建主任务时忽略该字段。
	ProgressUnit string
}

// CreateTask 组装任务数据并交给 Repository 写入数据库。
//
// 创建成功时仅返回 nil，不查询或组装任务 View；调用方需要展示最新任务信息时，
// 应在创建接口成功后通过任务查询接口重新获取数据。
func (s *Service) CreateTask(userID uint, input CreateTaskInput) error {
	taskType := strings.TrimSpace(input.TaskType)
	if taskType == "" {
		taskType = "main"
	}
	priority := strings.TrimSpace(input.Priority)
	if priority == "" {
		priority = "medium"
	}
	startTime, err := normalizeTaskTime(input.StartTime)
	if err != nil {
		return err
	}
	endTime, err := normalizeTaskTime(input.EndTime)
	if err != nil {
		return err
	}
	if input.ParentID != nil {
		items, err := s.repository.ListTasks(userID)
		if err != nil {
			return err
		}
		if err := validateParentSelection(items, 0, input.ListID, input.ParentID); err != nil {
			return err
		}
	}

	item := &task.Task{
		UserID:    userID,
		ListID:    input.ListID,
		ParentID:  input.ParentID,
		TaskType:  taskType,
		Title:     strings.TrimSpace(input.Title),
		Remark:    strings.TrimSpace(input.Remark),
		StartDate: input.StartDate,
		StartTime: startTime,
		EndDate:   input.EndDate,
		EndTime:   endTime,
		Priority:  priority,
	}
	if taskType == "subtask" {
		total := amount(input.ProgressTotal, defaultProgressTotal)
		completed := amount(input.ProgressCompleted, 0)
		step := amount(input.ProgressStep, defaultProgressStep)
		unit := strings.TrimSpace(input.ProgressUnit)
		item.ProgressTotal = &total
		item.ProgressCompleted = &completed
		item.ProgressStep = &step
		item.ProgressUnit = &unit
	}
	return s.repository.CreateTask(item)
}

// validateParentSelection 校验上级节点属于同一用户查询结果和同一任务清单，并确保
// 从候选上级节点沿祖先链向上遍历时不会回到当前任务，从而阻止自引用和循环层级。
func validateParentSelection(items []task.Task, currentID, listID uint, parentID *uint) error {
	if parentID == nil {
		return nil
	}

	itemsByID := make(map[uint]task.Task, len(items))
	for _, item := range items {
		itemsByID[item.ID] = item
	}

	visited := make(map[uint]bool)
	for currentParentID := *parentID; currentParentID != 0; {
		if currentID != 0 && currentParentID == currentID {
			return ErrInvalidParent
		}
		if visited[currentParentID] {
			return ErrInvalidParent
		}
		visited[currentParentID] = true

		parent, ok := itemsByID[currentParentID]
		if !ok || parent.ListID != listID {
			return ErrParentNotFound
		}
		// 兼容尚未经过启动迁移的历史数据：旧版本曾以 0 表示顶级节点。
		if parent.ParentID == nil || *parent.ParentID == 0 {
			return nil
		}
		currentParentID = *parent.ParentID
	}
	return ErrParentNotFound
}

// normalizeTaskTime 将任务时间统一为只包含小时和分钟的 HH:mm 格式。
//
// 空字符串表示未设置时间；为了兼容数据库中的历史数据和旧客户端，函数同时接受
// HH:mm:ss 输入，但写入前会丢弃秒。其他格式或超出正常时钟范围的值均视为无效。
func normalizeTaskTime(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	for _, layout := range []string{"15:04", "15:04:05"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.Format("15:04"), nil
		}
	}
	return "", ErrInvalidTask
}

// amount 将可选的十进制进度文本转换为数据库使用的浮点数。
//
// 转换前会去除文本首尾空白；当 value 为 nil 或去除空白后为空字符串时，
// 函数直接返回 fallback。对于非空文本，函数将其解析为浮点数并按原值返回，
// 例如文本 "1.25" 会转换为 1.25。当前实现不返回解析错误，无法解析的非空文本
// 按 0 处理。
//
// 参数：
//   - value：待转换的可选进度文本指针；nil 或空白文本表示未提供进度值。
//   - fallback：value 未提供时使用的默认值。
//
// 返回值：
//   - float64：转换后的原始数值；value 未提供时返回 fallback，非空文本无法解析时返回 0。
func amount(value *string, fallback float64) float64 {
	if value == nil || strings.TrimSpace(*value) == "" {
		return fallback
	}
	number, _ := strconv.ParseFloat(strings.TrimSpace(*value), 64)
	return number
}
