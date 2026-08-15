package task

import "time"

const (
	DefaultTaskListColor = "#6B7280"
	DefaultTaskListIcon  = "Briefcase"
)

// TaskList 表示用户创建的任务清单，用于对具体任务进行分类和组织。
//
// 每个任务清单只属于一个用户，清单之间不存在父子层级。Remark 用于
// 记录任务实施信息，Color 与 Icon 分别描述清单图标的颜色和图形。
type TaskList struct {
	// ID 是任务清单的数据库主键，也是对外暴露的唯一标识。
	ID uint `json:"id" gorm:"primaryKey"`
	// UserID 是任务清单所属用户的 ID，用于数据隔离，不会写入 JSON 响应。
	UserID uint `json:"-" gorm:"not null;index"`
	// Name 是任务清单名称，不能为空，数据库最多保存 128 个字符。
	Name string `json:"name" gorm:"size:128;not null"`
	// Remark 是可选的清单备注，主要用于记录任务实施信息。
	Remark string `json:"remark" gorm:"size:2000;not null;default:''"`
	// Color 保存图标的颜色值。
	Color string `json:"color" gorm:"size:20;not null;default:#6B7280"`
	// Icon 保存前端可识别的图标名称或编码。
	Icon string `json:"icon" gorm:"size:100;not null;default:Briefcase"`
	// CreatedAt 记录任务清单首次写入数据库的时间，由 GORM 自动维护。
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt 记录任务清单最近一次更新的时间，由 GORM 自动维护。
	UpdatedAt time.Time `json:"updatedAt"`
}

// Task 表示任务清单中的具体待办事项。
//
// 每个任务同时属于一个用户和一个任务清单。任务可以通过 ParentID 建立父子关系，
// 形成任意深度的任务层级；TaskType 独立描述任务是主任务还是子任务，因此主任务
// 也可以挂在另一个任务下面。Service 层负责确保父任务与下级任务属于同一用户和
// 同一任务清单。Priority 用于描述任务的优先级。
type Task struct {
	// ID 是任务的数据库主键，也是对外暴露的唯一标识。
	ID uint `json:"id" gorm:"primaryKey"`
	// UserID 是任务所属用户的 ID，用于数据隔离，不会写入 JSON 响应。
	UserID uint `json:"-" gorm:"not null;index"`
	// ListID 是任务所属任务清单的 ID，创建任务时必须提供。
	ListID uint `json:"listId" gorm:"not null;index"`
	// ParentID 是可选的父任务 ID；仅描述展示层级，不决定任务类型。
	ParentID *uint `json:"parentId" gorm:"index"`
	// TaskType 表示任务类型，可用值为 main 或 subtask；该字段与 ParentID 相互独立。
	TaskType string `json:"taskType" gorm:"size:20;not null;default:main"`
	// Title 是任务标题，不能为空，数据库最多保存 200 个字符。
	Title string `json:"title" gorm:"size:200;not null"`
	// Remark 是任务的补充备注，数据库最多保存 1000 个字符。
	Remark    string `json:"remark" gorm:"size:1000"`
	StartDate string `json:"startDate" gorm:"size:10"`
	StartTime string `json:"startTime" gorm:"size:5"`
	EndDate   string `json:"endDate" gorm:"size:10"`
	EndTime   string `json:"endTime" gorm:"size:5"`
	// Priority 表示任务优先级，可用值为 high、medium 或 low，默认为 medium。
	Priority string `json:"priority" gorm:"size:20;not null;default:medium"`
	// Archived 表示任务是否已归档；归档和恢复操作不会改变任务的原有进度。
	Archived bool `json:"archived" gorm:"not null;default:false;index"`
	// ProgressTotal、ProgressCompleted、ProgressStep 与 ProgressUnit 仅用于子任务。
	// 主任务不保存自身进度配置，因此这些字段保持 NULL，JSON 响应中也会省略。
	ProgressTotal     *float64 `json:"progressTotal,omitempty"`
	ProgressCompleted *float64 `json:"progressCompleted,omitempty"`
	ProgressStep      *float64 `json:"progressStep,omitempty"`
	ProgressUnit      *string  `json:"progressUnit,omitempty" gorm:"size:20"`
	// CreatedAt 记录任务首次写入数据库的时间，由 GORM 自动维护。
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt 记录任务最近一次更新的时间，由 GORM 自动维护。
	UpdatedAt time.Time `json:"updatedAt"`
}
