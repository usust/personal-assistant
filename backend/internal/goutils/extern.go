// Derived from github.com/usust/goUtils and modified for local integration.
package goutils

import "go.uber.org/zap"

var (
	// ConfigFileEnv can override the default config.yaml path.
	ConfigFileEnv = "EXPORT_CONFIG_FILE"
	Logger        = zap.NewNop()
	ZapLogger     = Logger
	SugaredLogger = Logger.Sugar()
)
