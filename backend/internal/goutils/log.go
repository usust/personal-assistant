// Derived from github.com/usust/goUtils and modified for local integration.
package goutils

import (
	"fmt"
	"reflect"

	logger "personal_assistant_server/internal/goutils/zap"
)

// ZapLogConfig 是底层 Zap 日志配置类型的别名。
//
// 使用类型别名可以让 goutils 包的调用方直接声明日志配置，而不需要依赖
// 内部的 zap 子包，同时保证配置可以直接传递给底层日志初始化函数。
type ZapLogConfig = logger.ZapLogConfig

// ZapLogConfigProvider 定义获取 Zap 日志配置的能力。
//
// 配置对象实现该接口后，resolveZapLogConfig 会优先通过接口获取日志配置，
// 无需再使用反射查找 ZapLog 或 ZapLogConfig 字段。
type ZapLogConfigProvider interface {
	// ZapLogConfig 返回当前配置对象中的 Zap 日志配置。
	ZapLogConfig() ZapLogConfig
}

// LoadLog 从配置对象中解析日志配置并初始化 Zap 日志系统。
//
// 初始化成功后，函数会同步更新 goutils 包对外暴露的 Logger、ZapLogger 和
// SugaredLogger，使后续业务代码使用已经完成配置的日志实例。
//
// 参数：
//   - cfg：日志配置来源，可以实现 ZapLogConfigProvider 接口，也可以是包含
//     ZapLog 或 ZapLogConfig 字段的结构体或结构体指针。
//
// 返回值：
//   - error：日志配置解析失败或 Zap 初始化失败时返回错误；成功时返回 nil。
func LoadLog(cfg any) error {
	conf, err := resolveZapLogConfig(cfg)
	if err != nil {
		return err
	}
	if err := logger.InitZapWithConfig(nil, conf); err != nil {
		return err
	}
	Logger = logger.Logger
	ZapLogger = logger.Logger
	SugaredLogger = logger.SugaredLogger
	return nil
}

// SyncZap 刷新 Zap 日志缓冲区，确保尚未写出的日志被提交到输出目标。
//
// 该函数通常通过 defer 在程序退出前调用。底层同步过程中产生的错误由日志
// 模块统一忽略，因此本函数没有参数和返回值。
func SyncZap() {
	logger.Sync()
}

// resolveZapLogConfig 从任意配置对象中解析 Zap 日志配置。
//
// 函数优先判断 cfg 是否实现 ZapLogConfigProvider；如果没有实现，则通过
// 反射逐层解引用指针，并依次查找 ZapLog、ZapLogConfig 字段。字段类型既可以
// 是 ZapLogConfig，也可以是指向 ZapLogConfig 的非 nil 指针。
//
// 参数：
//   - cfg：日志配置来源，不能是 nil 或 nil 指针。
//
// 返回值：
//   - ZapLogConfig：成功解析到的日志配置。
//   - error：配置来源无效、字段无法访问或未找到支持的日志配置时返回错误。
func resolveZapLogConfig(cfg any) (ZapLogConfig, error) {
	if cfg == nil {
		return ZapLogConfig{}, fmt.Errorf("log config source is nil")
	}

	// 优先使用显式接口，避免依赖字段名称和反射。
	if provider, ok := cfg.(ZapLogConfigProvider); ok {
		return provider.ZapLogConfig(), nil
	}

	// 逐层解引用指针，直到获得实际配置值。
	value := reflect.ValueOf(cfg)
	for value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return ZapLogConfig{}, fmt.Errorf("log config source is nil pointer")
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return ZapLogConfig{}, fmt.Errorf("log config source must be a struct or implement ZapLogConfigProvider")
	}

	// 兼容 ZapLog 和 ZapLogConfig 两种常见字段命名。
	for _, name := range []string{"ZapLog", "ZapLogConfig"} {
		field := value.FieldByName(name)
		if !field.IsValid() {
			continue
		}
		if !field.CanInterface() {
			return ZapLogConfig{}, fmt.Errorf("log config field %s cannot be accessed", name)
		}
		if conf, ok := field.Interface().(ZapLogConfig); ok {
			return conf, nil
		}
		if conf, ok := field.Interface().(*ZapLogConfig); ok && conf != nil {
			return *conf, nil
		}
	}
	return ZapLogConfig{}, fmt.Errorf("log config not found: implement ZapLogConfigProvider or define ZapLog/ZapLogConfig field")
}
