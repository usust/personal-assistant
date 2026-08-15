// Derived from github.com/usust/goUtils and modified for local integration.
package logger

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	Logger        = zap.NewNop()
	SugaredLogger = Logger.Sugar()
	loggerMu      sync.Mutex
)

// InitZapCore initializes logging with defaults overridden by options.
func InitZapCore(encoder *zapcore.EncoderConfig, options ...ZapOption) error {
	conf := ZapLogConfig{
		LogDir:     "./log",
		LogLevel:   LOG_LEVEL_DEBUG,
		MaxSize:    10,
		MaxBackups: 30,
		MaxAge:     180,
		IsCompress: true,
	}
	for _, option := range options {
		if option != nil {
			option(&conf)
		}
	}
	return InitZapWithConfig(encoder, conf)
}

// InitZapWithConfig writes debug, info, and error levels to separate rotating
// files and writes every enabled level to stdout.
func InitZapWithConfig(encoder *zapcore.EncoderConfig, conf ZapLogConfig) error {
	loggerMu.Lock()
	defer loggerMu.Unlock()

	encoderConfig := defaultEncoderConfig()
	if encoder != nil {
		encoderConfig = *encoder
	}
	applyDefaults(&conf)
	if err := os.MkdirAll(conf.LogDir, 0o755); err != nil {
		return err
	}
	minLevel, err := zapcore.ParseLevel(strings.ToLower(conf.LogLevel))
	if err != nil {
		return err
	}

	jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)
	writer := func(filename string) zapcore.WriteSyncer {
		return zapcore.AddSync(&lumberjack.Logger{
			Filename:   filepath.Join(conf.LogDir, filename),
			MaxSize:    conf.MaxSize,
			MaxBackups: conf.MaxBackups,
			MaxAge:     conf.MaxAge,
			Compress:   conf.IsCompress,
		})
	}
	infoLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level >= minLevel && level >= zapcore.InfoLevel && level < zapcore.ErrorLevel
	})
	errorLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level >= minLevel && level >= zapcore.ErrorLevel
	})
	debugLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level >= minLevel && level == zapcore.DebugLevel
	})
	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		minLevel,
	)
	core := zapcore.NewTee(
		zapcore.NewCore(jsonEncoder, writer("info.log"), infoLevel),
		zapcore.NewCore(jsonEncoder, writer("error.log"), errorLevel),
		zapcore.NewCore(jsonEncoder, writer("debug.log"), debugLevel),
		consoleCore,
	)

	oldLogger := Logger
	oldSugaredLogger := SugaredLogger
	Logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zap.ErrorLevel))
	SugaredLogger = Logger.Sugar()
	zap.ReplaceGlobals(Logger)
	if oldLogger != nil {
		_ = oldLogger.Sync()
	}
	if oldSugaredLogger != nil {
		_ = oldSugaredLogger.Sync()
	}
	return nil
}

func applyDefaults(conf *ZapLogConfig) {
	if conf.LogDir == "" {
		conf.LogDir = "./log"
	}
	if conf.LogLevel == "" {
		conf.LogLevel = LOG_LEVEL_DEBUG
	}
	if conf.MaxSize <= 0 {
		conf.MaxSize = 10
	}
	if conf.MaxBackups <= 0 {
		conf.MaxBackups = 30
	}
	if conf.MaxAge <= 0 {
		conf.MaxAge = 180
	}
}

func defaultEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

func Sync() {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	if Logger != nil {
		_ = Logger.Sync()
	}
	if SugaredLogger != nil {
		_ = SugaredLogger.Sync()
	}
}
