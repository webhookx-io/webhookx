package function

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogAPI struct{}

func NewLogger() *LogAPI {
	return &LogAPI{}
}

func log(level zapcore.Level, msg string) {
	zap.S().Log(level, msg)
}

func (m *LogAPI) Debug(msg string) {
	log(zapcore.DebugLevel, msg)
}

func (m *LogAPI) Info(msg string) {
	log(zapcore.InfoLevel, msg)
}

func (m *LogAPI) Warn(msg string) {
	log(zapcore.WarnLevel, msg)
}

func (m *LogAPI) Error(msg string) {
	log(zapcore.ErrorLevel, msg)
}
