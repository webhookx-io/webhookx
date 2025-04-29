package sdk

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogSDK struct{}

func NewLogSDK() *LogSDK {
	return &LogSDK{}
}

func log(level zapcore.Level, message string) {
	zap.S().Log(level, message)
}

func (m *LogSDK) Debug(message string) {
	log(zapcore.DebugLevel, message)
}

func (m *LogSDK) Info(message string) {
	log(zapcore.InfoLevel, message)
}

func (m *LogSDK) Warn(message string) {
	log(zapcore.WarnLevel, message)
}

func (m *LogSDK) Error(message string) {
	log(zapcore.ErrorLevel, message)
}
