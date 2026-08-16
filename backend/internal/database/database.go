package database

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"personal_assistant_server/internal/config"
	"personal_assistant_server/internal/model"
	"personal_assistant_server/internal/task"
)

// Connect 根据配置创建 MySQL 数据库连接。
// 参数 cfg 包含 MySQL 连接信息；返回已建立的 GORM 数据库连接和连接错误。
func Connect(cfg config.DatabaseConnectionConfig) (*gorm.DB, error) {
	return connectMySQL(cfg)
}

// MigrateAndSeed 创建数据库表结构，并在管理员账户不存在时写入初始账户。
// 参数 db 是已连接的 GORM 实例；返回迁移、查询、密码生成或写入过程中的错误。
func MigrateAndSeed(db *gorm.DB) error {
	hadFixedPointTaskProgress, err := hasFixedPointTaskProgress(db)
	if err != nil {
		return err
	}
	hadTaskTypeColumn := db.Migrator().HasTable(&task.Task{}) && db.Migrator().HasColumn(&task.Task{}, "task_type")
	if err := db.AutoMigrate(&model.User{}, &task.TaskList{}, &task.Task{}); err != nil {
		return err
	}
	// 历史接口曾使用 parent_id = 0 表示顶级任务，而当前模型使用 NULL。
	// 统一为 NULL，避免选择这类历史顶级任务作为上级节点时沿祖先链走到 0，
	// 被误判为上级节点不存在。该清理可在每次启动时安全重复执行。
	if err := db.Exec("UPDATE tasks SET parent_id = NULL WHERE parent_id = 0").Error; err != nil {
		return fmt.Errorf("normalize zero task parent id: %w", err)
	}
	// 旧客户端和历史接口可能将任务时间保存为 HH:mm:ss。任务时间现在统一只保留
	// 小时和分钟；该语句仅处理超过 5 个字符的非空值，并且可在每次启动时安全重复执行。
	if err := db.Exec(`
		UPDATE tasks
		SET start_time = CASE
		        WHEN LENGTH(start_time) > 5 THEN SUBSTR(start_time, 1, 5)
		        ELSE start_time
		    END,
		    end_time = CASE
		        WHEN LENGTH(end_time) > 5 THEN SUBSTR(end_time, 1, 5)
		        ELSE end_time
		    END
		WHERE LENGTH(start_time) > 5 OR LENGTH(end_time) > 5
	`).Error; err != nil {
		return fmt.Errorf("normalize task times to minute precision: %w", err)
	}
	// 旧版本通过 parent_id 推导任务类型。首次增加独立的 task_type 字段时，
	// 按原规则回填已有数据，避免历史下级任务被数据库默认值误标为主任务。
	taskTypeScope := db.Model(&task.Task{})
	if hadTaskTypeColumn {
		taskTypeScope = taskTypeScope.Where("task_type = '' OR task_type IS NULL")
	} else {
		taskTypeScope = taskTypeScope.Where("1 = 1")
	}
	if err := taskTypeScope.Update("task_type", gorm.Expr("CASE WHEN parent_id IS NULL THEN ? ELSE ? END", "main", "subtask")).Error; err != nil {
		return fmt.Errorf("migrate task type: %w", err)
	}
	// 将任务的旧 description 字段迁移为 remark。AutoMigrate 会先创建新的
	// remark 列；仅在旧列仍存在时复制数据并删除旧列，以支持重复启动。
	taskColumns, err := db.Migrator().ColumnTypes(&task.Task{})
	if err != nil {
		return fmt.Errorf("inspect task columns: %w", err)
	}
	hasTaskDescription := false
	hasTaskDue := false
	hasTaskDeadlineType := false
	for _, column := range taskColumns {
		switch {
		case strings.EqualFold(column.Name(), "description"):
			hasTaskDescription = true
		case strings.EqualFold(column.Name(), "due"):
			hasTaskDue = true
		case strings.EqualFold(column.Name(), "due_type"):
			hasTaskDeadlineType = true
		}
	}
	if hasTaskDescription {
		if err := db.Model(&task.Task{}).
			Where("remark = '' OR remark IS NULL").
			Update("remark", gorm.Expr("description")).Error; err != nil {
			return fmt.Errorf("migrate task description to remark: %w", err)
		}
		if err := db.Exec("ALTER TABLE tasks DROP COLUMN description").Error; err != nil {
			return fmt.Errorf("drop obsolete task column description: %w", err)
		}
	}
	// 截止日期和时间已由 end_date、end_time 分别保存，删除不再使用的汇总字段
	// 以及原先由客户端提供的截止状态字段。
	if hasTaskDue {
		if err := db.Exec("ALTER TABLE tasks DROP COLUMN due").Error; err != nil {
			return fmt.Errorf("drop obsolete task column due: %w", err)
		}
	}
	if hasTaskDeadlineType {
		if err := db.Exec("ALTER TABLE tasks DROP COLUMN due_type").Error; err != nil {
			return fmt.Errorf("drop obsolete task column due_type: %w", err)
		}
	}
	// 旧版本将进度放大 100 倍后保存为整数。仅当迁移前的进度列是整数类型时，
	// 才将已有数据恢复为用户输入的原始值；AutoMigrate 会把列类型改为浮点数，
	// 因此后续启动不会重复执行除法。
	if hadFixedPointTaskProgress {
		if err := db.Exec(`
			UPDATE tasks
			SET progress_total = progress_total / ?,
			    progress_completed = progress_completed / ?,
			    progress_step = progress_step / ?
		`, 100.0, 100.0, 100.0).Error; err != nil {
			return fmt.Errorf("migrate fixed-point task progress: %w", err)
		}
	}
	// 子任务使用独立进度配置；仅初始化尚未设置进度总量的子任务。
	if err := db.Model(&task.Task{}).
		Where("task_type = ? AND (progress_total IS NULL OR progress_total = 0)", "subtask").
		Updates(map[string]any{
			"progress_total":     100,
			"progress_step":      1,
			"progress_completed": 0,
		}).Error; err != nil {
		return fmt.Errorf("migrate task progress: %w", err)
	}
	// 主任务的进度由前端根据后代任务构建，不保存自身进度配置。该清理同时
	// 修复旧版本已经写入 1/0/1/空单位的主任务记录。
	if err := db.Exec(`
		UPDATE tasks
		SET progress_total = NULL,
		    progress_completed = NULL,
		    progress_step = NULL,
		    progress_unit = NULL
		WHERE task_type = ?
	`, "main").Error; err != nil {
		return fmt.Errorf("clear main task progress: %w", err)
	}
	// Convert the former mutually exclusive icon_type/icon representation into
	// independent color and icon fields before removing icon_type.
	if db.Migrator().HasColumn(&task.TaskList{}, "icon_type") {
		if err := db.Exec("UPDATE task_lists SET color = icon, icon = ? WHERE icon_type = ?", task.DefaultTaskListIcon, "color").Error; err != nil {
			return fmt.Errorf("migrate task list visual fields: %w", err)
		}
		if err := db.Migrator().DropColumn(&task.TaskList{}, "icon_type"); err != nil {
			return fmt.Errorf("drop obsolete task list column icon_type: %w", err)
		}
	}
	// TaskList 已改为扁平结构，清理旧版本遗留的层级与描述字段。
	for _, column := range []string{"parent_id", "description"} {
		if db.Migrator().HasColumn(&task.TaskList{}, column) {
			if err := db.Migrator().DropColumn(&task.TaskList{}, column); err != nil {
				return fmt.Errorf("drop obsolete task list column %s: %w", column, err)
			}
		}
	}

	var user model.User
	err = db.Where("username = ?", "admin").First(&user).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return db.Create(&model.User{Username: "admin", Password: string(hash), Nickname: "管理员", Role: "admin"}).Error
}

// hasFixedPointTaskProgress 判断现有任务进度是否仍使用放大 100 倍的整数列。
func hasFixedPointTaskProgress(db *gorm.DB) (bool, error) {
	if !db.Migrator().HasTable(&task.Task{}) {
		return false, nil
	}
	columns, err := db.Migrator().ColumnTypes(&task.Task{})
	if err != nil {
		return false, fmt.Errorf("inspect task progress columns: %w", err)
	}
	for _, column := range columns {
		if strings.EqualFold(column.Name(), "progress_total") {
			return strings.Contains(strings.ToUpper(column.DatabaseTypeName()), "INT"), nil
		}
	}
	return false, nil
}
