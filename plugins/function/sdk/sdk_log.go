package sdk

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogSDK struct{}

func NewLogSDK() *LogSDK {
	return &LogSDK{}
}

func log(level zapcore.Level, msg string) {
	zap.S().Log(level, msg)
}

func (m *LogSDK) Debug(msg string) {
	log(zapcore.DebugLevel, msg)
}

func (m *LogSDK) Info(msg string) {
	log(zapcore.InfoLevel, msg)
}

func (m *LogSDK) Warn(msg string) {
	log(zapcore.WarnLevel, msg)
}

func (m *LogSDK) Error(msg string) {
	log(zapcore.ErrorLevel, msg)
}
