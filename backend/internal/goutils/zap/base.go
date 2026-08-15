// Derived from github.com/usust/goUtils and modified for local integration.
package logger

const (
	LOG_LEVEL_DEBUG  = "debug"
	LOG_LEVEL_INFO   = "info"
	LOG_LEVEL_WARN   = "warn"
	LOG_LEVEL_ERROR  = "error"
	LOG_LEVEL_DPANIC = "dpanic"
	LOG_LEVEL_PANIC  = "panic"
	LOG_LEVEL_FATAL  = "fatal"
)

// ZapLogConfig contains the file output, level, and rotation settings.
type ZapLogConfig struct {
	LogDir     string `mapstructure:"log_dir"`
	LogLevel   string `mapstructure:"log_level"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	IsCompress bool   `mapstructure:"iscompress"`
}

type ZapOption func(*ZapLogConfig)

func ZapWithLogDir(dir string) ZapOption  { return func(c *ZapLogConfig) { c.LogDir = dir } }
func ZapWithLevel(level string) ZapOption { return func(c *ZapLogConfig) { c.LogLevel = level } }
func ZapWithMaxSize(max int) ZapOption    { return func(c *ZapLogConfig) { c.MaxSize = max } }
func ZapWithMaxBackups(max int) ZapOption { return func(c *ZapLogConfig) { c.MaxBackups = max } }
func ZapWithMaxAge(days int) ZapOption    { return func(c *ZapLogConfig) { c.MaxAge = days } }
func ZapWithIsCompress(enabled bool) ZapOption {
	return func(c *ZapLogConfig) { c.IsCompress = enabled }
}
