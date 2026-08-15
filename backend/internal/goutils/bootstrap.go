// Derived from github.com/usust/goUtils and modified for local integration.
package goutils

import (
	"fmt"
	"log"
	"os"
)

// bootstrapLogger 在正式日志系统初始化前输出启动过程。
// 使用标准日志可以确保配置加载失败或 Zap 初始化失败时仍有日志可供排查。
var bootstrapLogger = log.New(os.Stdout, "[bootstrap] ", log.LstdFlags)

// Hooks 定义应用启动过程中可以替换的初始化阶段。
//
// 各阶段按照 LoadConfig、LoadLog 的顺序执行。调用方可以替换这些函数，
// 以便为测试注入桩函数，或者为不同运行环境提供自定义初始化逻辑。
type Hooks struct {
	// InitConfigFunc 将配置数据加载到 dst 指向的对象中。
	// 该函数为必填项；InitWith 会在执行前检查其是否为 nil。
	InitConfigFunc func(dst any) error

	// InitLogFunc 根据已经完成加载的 cfg 初始化日志系统。
	// 该函数为可选项；值为 nil 时，InitWith 将跳过日志初始化阶段。
	InitLogFunc func(cfg any) error
}

// Bootstrap 使用默认启动流程初始化应用。
//
// 函数首先调用 LoadConfigFileInto 将配置文件内容写入 cfg，随后调用
// LoadLog，根据加载完成的配置初始化日志系统。
//
// 参数：
//   - cfg：配置对象，通常应为指向配置结构体的非 nil 指针。
//
// 返回值：
//   - error：配置加载或日志初始化失败时返回包含阶段信息的错误；全部成功时返回 nil。
func Bootstrap(cfg any) error {
	return InitWith(cfg, Hooks{
		InitConfigFunc: LoadConfigFileInto,
		InitLogFunc:    LoadLog,
	})
}

// InitWith 使用指定的 Hooks 按固定顺序执行应用初始化阶段。
//
// 函数先执行 hooks.InitConfigFunc；只有配置加载成功后，才会执行可选的 hooks.InitLogFunc。
// 任一阶段失败都会立即终止后续流程，并在原始错误外包装当前阶段信息，
// 调用方仍可通过 errors.Is 或 errors.As 检查底层错误。
//
// 参数：
//   - cfg：传递给各初始化阶段的配置对象；具体类型和有效性由相应 Hook 校验。
//   - hooks：启动阶段函数集合，其中 InitConfigFunc 必须提供，InitLogFunc 可以为 nil。
//
// 返回值：
//   - error：Hook 缺失或某个初始化阶段执行失败时返回错误；全部成功时返回 nil。
func InitWith(cfg any, hooks Hooks) error {
	// 检查 InitConfigFunc 是否为 nil，确保配置加载阶段可执行
	if hooks.InitConfigFunc == nil {
		err := fmt.Errorf("bootstrap hook init config function pointer is nil")
		bootstrapLogger.Printf("配置初始化失败：%v", err)
		return err
	}

	// 执行配置加载阶段
	bootstrapLogger.Printf("配置初始化开始")
	if err := hooks.InitConfigFunc(cfg); err != nil {
		wrappedErr := fmt.Errorf("init config failed: %w", err)
		bootstrapLogger.Printf("配置初始化失败：%v", wrappedErr)
		return wrappedErr
	}
	bootstrapLogger.Printf("配置初始化完成")

	// 执行日志初始化阶段
	if hooks.InitLogFunc != nil {
		// 仅在 InitLogFunc 不为 nil 时才执行日志初始化
		bootstrapLogger.Printf("日志初始化开始")
		if err := hooks.InitLogFunc(cfg); err != nil {
			wrappedErr := fmt.Errorf("init log failed: %w", err)
			bootstrapLogger.Printf("日志初始化失败：%v", wrappedErr)
			return wrappedErr
		}
		bootstrapLogger.Printf("日志初始化完成")
	} else {
		bootstrapLogger.Printf("已跳过日志初始化：初始化函数为空")
	}

	bootstrapLogger.Printf("应用启动初始化完成")
	return nil
}
