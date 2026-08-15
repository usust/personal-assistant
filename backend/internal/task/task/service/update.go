package service

import (
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"personal_assistant_server/internal/task"
)

// ErrInvalidTask 表示任务基础信息为空、格式错误或超出允许范围。
var ErrInvalidTask = errors.New("任务基础信息无效")

var progressAmountPattern = regexp.MustCompile(`^\d+(?:\.\d{1,2})?$`)

// UpdateTaskInput 是任务基础信息部分更新时由 Handler 传递给 Service 的输入参数。
type UpdateTaskInput struct {
	Title             *string
	Remark            *string
	StartDate         *string
	StartTime         *string
	EndDate           *string
	EndTime           *string
	Priority          *string
	Archived          *bool
	ParentID          *uint
	ParentIDSet       bool
	TaskType          *string
	ProgressTotal     *string
	ProgressCompleted *string
	ProgressStep      *string
	ProgressUnit      *string
}

// UpdateTask 校验并部分更新指定用户的任务信息。
func (s *Service) UpdateTask(userID, id uint, input UpdateTaskInput) error {
	item := &task.Task{}
	fields := make([]string, 0, 14)

	if input.Title != nil {
		value := strings.TrimSpace(*input.Title)
		if value == "" || utf8.RuneCountInString(value) > 200 {
			return ErrInvalidTask
		}
		item.Title = value
		fields = append(fields, "Title")
	}
	if input.Remark != nil {
		value := strings.TrimSpace(*input.Remark)
		if utf8.RuneCountInString(value) > 1000 {
			return ErrInvalidTask
		}
		item.Remark = value
		fields = append(fields, "Remark")
	}
	if input.StartDate != nil {
		value := strings.TrimSpace(*input.StartDate)
		if value != "" {
			if _, err := time.Parse("2006-01-02", value); err != nil {
				return ErrInvalidTask
			}
		}
		item.StartDate = value
		fields = append(fields, "StartDate")
	}
	if input.StartTime != nil {
		value, err := normalizeTaskTime(*input.StartTime)
		if err != nil {
			return err
		}
		item.StartTime = value
		fields = append(fields, "StartTime")
	}
	if input.EndDate != nil {
		value := strings.TrimSpace(*input.EndDate)
		if value != "" {
			if _, err := time.Parse("2006-01-02", value); err != nil {
				return ErrInvalidTask
			}
		}
		item.EndDate = value
		fields = append(fields, "EndDate")
	}
	if input.EndTime != nil {
		value, err := normalizeTaskTime(*input.EndTime)
		if err != nil {
			return err
		}
		item.EndTime = value
		fields = append(fields, "EndTime")
	}
	if input.Priority != nil {
		value := strings.TrimSpace(*input.Priority)
		if value != "high" && value != "medium" && value != "low" {
			return ErrInvalidTask
		}
		item.Priority = value
		fields = append(fields, "Priority")
	}
	if input.Archived != nil {
		item.Archived = *input.Archived
		fields = append(fields, "Archived")
	}

	hasProgressInput := input.ProgressTotal != nil ||
		input.ProgressCompleted != nil ||
		input.ProgressStep != nil ||
		input.ProgressUnit != nil
	needsCurrentTask := input.ParentIDSet || input.TaskType != nil || hasProgressInput
	var current *task.Task
	var items []task.Task
	if needsCurrentTask {
		var err error
		items, err = s.repository.ListTasks(userID)
		if err != nil {
			return err
		}
		for index := range items {
			if items[index].ID == id {
				current = &items[index]
				break
			}
		}
		if current == nil {
			return gorm.ErrRecordNotFound
		}
	}

	if input.ParentIDSet {
		if err := validateParentSelection(items, id, current.ListID, input.ParentID); err != nil {
			return err
		}
		item.ParentID = input.ParentID
		fields = append(fields, "ParentID")
	}

	targetTaskType := ""
	if current != nil {
		targetTaskType = current.TaskType
	}
	if input.TaskType != nil {
		targetTaskType = strings.TrimSpace(*input.TaskType)
		if targetTaskType != "main" && targetTaskType != "subtask" {
			return ErrInvalidTask
		}
		item.TaskType = targetTaskType
		fields = append(fields, "TaskType")
	}

	if input.TaskType != nil || hasProgressInput {
		if targetTaskType == "main" {
			if hasProgressInput {
				return ErrInvalidProgress
			}
			item.ProgressTotal = nil
			item.ProgressCompleted = nil
			item.ProgressStep = nil
			item.ProgressUnit = nil
			fields = append(fields, "ProgressTotal", "ProgressCompleted", "ProgressStep", "ProgressUnit")
		} else if targetTaskType == "subtask" {
			total, err := updatedProgressAmount(input.ProgressTotal, current.ProgressTotal, defaultProgressTotal)
			if err != nil {
				return err
			}
			completed, err := updatedProgressAmount(input.ProgressCompleted, current.ProgressCompleted, 0)
			if err != nil {
				return err
			}
			step, err := updatedProgressAmount(input.ProgressStep, current.ProgressStep, defaultProgressStep)
			if err != nil {
				return err
			}
			unit := ""
			if current.ProgressUnit != nil {
				unit = strings.TrimSpace(*current.ProgressUnit)
			}
			if input.ProgressUnit != nil {
				unit = strings.TrimSpace(*input.ProgressUnit)
			}
			if total <= 0 || completed < 0 || completed > total ||
				step <= 0 || step > total || utf8.RuneCountInString(unit) > 20 {
				return ErrInvalidProgress
			}
			item.ProgressTotal = &total
			item.ProgressCompleted = &completed
			item.ProgressStep = &step
			item.ProgressUnit = &unit
			fields = append(fields, "ProgressTotal", "ProgressCompleted", "ProgressStep", "ProgressUnit")
		} else {
			return ErrInvalidTask
		}
	}

	if len(fields) == 0 {
		return ErrInvalidTask
	}

	return s.repository.UpdateTask(userID, id, item, fields)
}

// updatedProgressAmount 返回 PATCH 后的单个进度数值；未提供字段时保留数据库值，
// 原值为空时使用 fallback。显式提供的值必须是最多两位小数的非负十进制数。
func updatedProgressAmount(value *string, current *float64, fallback float64) (float64, error) {
	if value == nil {
		if current != nil {
			return *current, nil
		}
		return fallback, nil
	}

	text := strings.TrimSpace(*value)
	if !progressAmountPattern.MatchString(text) {
		return 0, ErrInvalidProgress
	}
	number, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsInf(number, 0) || math.IsNaN(number) {
		return 0, ErrInvalidProgress
	}
	return number, nil
}
