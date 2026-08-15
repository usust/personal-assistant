package config

import (
	"fmt"
	"strconv"
	"strings"

	go_utils "personal_assistant_server/internal/goutils"
)

var cfg *Config

// Initialize 加载、校验并保存应用的全局配置。
//
// 配置文件路径和加载规则由 goUtils.Bootstrap 统一处理；
// 只有配置加载、校验和日志初始化全部成功后，才会更新全局配置。
//
// 参数：
//   - 无。
//
// 返回值：
//   - error：配置加载、校验或日志初始化失败时返回错误；初始化成功时返回 nil。
func Initialize() error {
	loaded := &Config{}
	if err := go_utils.Bootstrap(loaded); err != nil {
		return err
	}
	cfg = loaded
	return nil
}

// Get 获取已经初始化的全局应用配置。
//
// 调用方应当在 Initialize 成功后调用该函数，并将返回的配置视为只读数据。
//
// 参数：
//   - 无。
//
// 返回值：
//   - *Config：当前全局应用配置的指针；尚未成功初始化时返回 nil。
func Get() *Config {
	return cfg
}

// Config 定义应用启动和运行所需的全部配置。
type Config struct {
	Host                string                              `mapstructure:"host"`                 // HTTP 服务监听地址
	Port                string                              `mapstructure:"port"`                 // HTTP 服务监听端口
	Mode                string                              `mapstructure:"mode"`                 // 运行模式：debug、release 或 test
	Database            DatabaseConfig                      `mapstructure:"database"`             // 当前数据库选择配置
	DatabaseConnections map[string]DatabaseConnectionConfig `mapstructure:"database_connections"` // 一级数据库连接列表
	JWTSecret           string                              `mapstructure:"jwt_secret"`           // JWT 签名密钥
	JWTExpireHours      int                                 `mapstructure:"jwt_expire_hours"`     // JWT 有效期，单位为小时
	AllowedOrigins      []string                            `mapstructure:"allowed_origins"`      // 允许跨域访问的来源列表
	ZapLog              go_utils.ZapLogConfig               `mapstructure:"zap_log"`              // Zap 日志配置
}

// DatabaseConfig 定义数据库选择器。
type DatabaseConfig struct {
	Active string `mapstructure:"active"` // 当前启用的连接名称
}

// DatabaseConnectionConfig 定义单个 SQLite 或 MySQL 数据库的连接参数。
// Driver 决定实际使用哪一组字段，未被选择的连接不会参与参数校验或建立连接。
type DatabaseConnectionConfig struct {
	Driver   string `mapstructure:"driver"`   // 数据库驱动，可选 sqlite 或 mysql
	Path     string `mapstructure:"path"`     // SQLite 数据库文件路径
	Host     string `mapstructure:"host"`     // MySQL 服务器地址
	Port     int    `mapstructure:"port"`     // MySQL 服务端口
	Username string `mapstructure:"username"` // MySQL 登录用户名
	Password string `mapstructure:"password"` // MySQL 登录密码
	Name     string `mapstructure:"name"`     // MySQL 数据库名称
	Charset  string `mapstructure:"charset"`  // MySQL 连接字符集
}

// ActiveDatabaseConnection 获取当前启用的命名数据库连接。
//
// 参数：
//   - c：应用配置，包含当前启用的数据库连接名称和命名数据库连接列表。
//
// 返回值：
//   - string：去除首尾空白后的当前数据库连接名称。
//   - DatabaseConnectionConfig：当前数据库连接对应的配置。
//   - error：Active 为空或指定名称不存在时返回错误；选择成功时返回 nil。
func (c *Config) ActiveDatabaseConnection() (string, DatabaseConnectionConfig, error) {
	active := strings.TrimSpace(c.Database.Active)
	if active == "" {
		return "", DatabaseConnectionConfig{}, fmt.Errorf("database.active must not be empty")
	}
	connection, ok := c.DatabaseConnections[active]
	if !ok {
		return "", DatabaseConnectionConfig{}, fmt.Errorf("database.active %q does not exist in database_connections", active)
	}
	return active, connection, nil
}

// validate 校验并规范化单个数据库连接配置。
//
// 该方法仅由当前启用的连接调用，并根据所选驱动规范化相关字符串字段；
// MySQL 连接未设置字符集时，将默认使用 utf8mb4。
//
// 参数：
//   - c：待校验和规范化的数据库连接配置，必须为非 nil 指针。
//
// 返回值：
//   - error：驱动不受支持或所选驱动的必要参数无效时返回错误；校验通过时返回 nil。
func (c *DatabaseConnectionConfig) validate() error {
	c.Driver = strings.ToLower(strings.TrimSpace(c.Driver))
	switch c.Driver {
	case "sqlite":
		c.Path = strings.TrimSpace(c.Path)
		if c.Path == "" {
			return fmt.Errorf("path must not be empty when driver is sqlite")
		}
	case "mysql":
		c.Host = strings.TrimSpace(c.Host)
		c.Username = strings.TrimSpace(c.Username)
		c.Name = strings.TrimSpace(c.Name)
		c.Charset = strings.TrimSpace(c.Charset)
		if c.Host == "" {
			return fmt.Errorf("host must not be empty when driver is mysql")
		}
		if c.Port <= 0 || c.Port > 65535 {
			return fmt.Errorf("port must be between 1 and 65535 when driver is mysql")
		}
		if c.Username == "" {
			return fmt.Errorf("username must not be empty when driver is mysql")
		}
		if c.Name == "" {
			return fmt.Errorf("name must not be empty when driver is mysql")
		}
		if c.Charset == "" {
			c.Charset = "utf8mb4"
		}
	default:
		return fmt.Errorf("driver must be sqlite or mysql")
	}
	return nil
}

// Validate 实现 goUtils.ConfigValidator 接口。
//
// 该方法在数据库和 HTTP 服务启动前校验并规范化配置，避免应用使用无效配置运行。
//
// 参数：
//   - c：待校验和规范化的应用配置，必须为非 nil 指针。
//
// 返回值：
//   - error：端口、运行模式、当前数据库连接、JWT 或跨域配置无效时返回错误；
//     全部配置校验通过时返回 nil。
func (c *Config) Validate() error {
	// 监听地址不能为空；可使用主机名、IP 地址或 0.0.0.0 监听所有 IPv4 网卡。
	c.Host = strings.TrimSpace(c.Host)
	if c.Host == "" {
		return fmt.Errorf("host must not be empty")
	}
	// 端口必须是 1～65535 范围内的有效 TCP 端口。
	port, err := strconv.ParseUint(c.Port, 10, 16)
	if err != nil || port == 0 {
		if err == nil {
			err = fmt.Errorf("port is zero")
		}
		return fmt.Errorf("port must be a valid TCP port: %w", err)
	}
	// 运行模式仅允许使用 Gin 支持的标准模式。
	switch c.Mode {
	case "debug", "release", "test":
	default:
		return fmt.Errorf("mode must be debug, release, or test")
	}
	// 仅校验当前启用的命名数据库连接，其他连接配置可以继续保存在配置文件中。
	active, connection, err := c.ActiveDatabaseConnection()
	if err != nil {
		return err
	}
	c.Database.Active = active
	if err := connection.validate(); err != nil {
		return fmt.Errorf("database_connections.%s: %w", active, err)
	}
	c.DatabaseConnections[active] = connection
	// JWT 签名密钥和有效期是签发访问令牌的必要参数。
	if strings.TrimSpace(c.JWTSecret) == "" {
		return fmt.Errorf("jwt_secret must not be empty")
	}
	if c.JWTExpireHours <= 0 {
		return fmt.Errorf("jwt_expire_hours must be greater than zero")
	}
	// 跨域来源列表至少包含一项，且不允许出现空字符串。
	if len(c.AllowedOrigins) == 0 {
		return fmt.Errorf("allowed_origins must contain at least one origin")
	}
	for i, origin := range c.AllowedOrigins {
		c.AllowedOrigins[i] = strings.TrimSpace(origin)
		if c.AllowedOrigins[i] == "" {
			return fmt.Errorf("allowed_origins must not contain an empty origin")
		}
	}
	return nil
}
