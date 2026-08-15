// Derived from github.com/usust/goUtils and modified for local integration.
package goutils

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/spf13/viper"
)

// ConfigValidator 定义配置对象的校验能力。
//
// 配置结构体可以实现该接口，使 LoadConfigFileInto 在完成反序列化后
// 自动校验配置内容。校验失败时，配置加载流程将返回错误。
type ConfigValidator interface {
	// Validate 校验配置对象中的字段是否合法。
	// 返回 nil 表示校验通过，否则返回具体的配置错误。
	Validate() error
}

// newViper 创建用于读取应用配置的 Viper 实例。
//
// 参数：
//   - configFile：指定配置文件路径；为空时使用当前目录下的 config.yaml。
//
// 返回值：
//   - *viper.Viper：已经设置配置文件路径或默认查找规则的 Viper 实例。
func newViper(configFile string) *viper.Viper {
	v := viper.New()
	// 环境变量指定了配置文件时，直接使用该文件，不再执行默认查找。
	if configFile != "" {
		v.SetConfigFile(configFile)
		return v
	}

	// 未指定文件时，默认读取当前工作目录中的 config.yaml。
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	return v
}

// LoadConfigFileInto 读取配置文件并将内容反序列化到指定对象中。
//
// 函数优先读取 ConfigFileEnv 环境变量指定的配置文件；环境变量为空时，
// 默认读取当前工作目录下的 config.yaml。反序列化完成后，如果 cfg 实现了
// ConfigValidator 接口，还会自动调用 Validate 校验配置内容。
//
// 参数：
//   - cfg：接收配置数据的目标对象，必须是非 nil 指针。
//
// 返回值：
//   - error：目标对象无效、配置文件读取失败、反序列化失败或配置校验失败时
//     返回包含具体阶段信息的错误；全部成功时返回 nil。
func LoadConfigFileInto(cfg any) error {
	// 提前拒绝无效目标，避免反序列化过程中出现不明确的错误。
	if cfg == nil {
		return fmt.Errorf("config destination is nil")
	}
	value := reflect.ValueOf(cfg)
	if value.Kind() != reflect.Ptr || value.IsNil() {
		return fmt.Errorf("config destination must be a non-nil pointer")
	}

	// 去除环境变量两端的空白，空字符串将使用默认配置文件查找规则。
	configFile := strings.TrimSpace(os.Getenv(ConfigFileEnv))
	v := newViper(configFile)
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("read config file failed (env %s=%q): %w", ConfigFileEnv, configFile, err)
	}
	bootstrapLogger.Printf("配置文件读取成功：%s", v.ConfigFileUsed())
	if err := v.Unmarshal(cfg); err != nil {
		return fmt.Errorf("unmarshal config file failed: %w", err)
	}
	bootstrapLogger.Printf("配置文件反序列化成功")

	// 仅对实现了 ConfigValidator 的配置对象执行额外校验。
	if validator, ok := cfg.(ConfigValidator); ok {
		if err := validator.Validate(); err != nil {
			return fmt.Errorf("invalid config: %w", err)
		}
		bootstrapLogger.Printf("配置校验完成")
	} else {
		bootstrapLogger.Printf("已跳过配置校验：配置对象未实现校验接口")
	}
	return nil
}
