package sdk

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogSDK struct{}

func NewLogSDK() *LogSDK {
	return &LogSDK{}
}

func log(level zapcore.Level, args []interface{}) {
	zap.S().Logln(level, args...)
}

func (m *LogSDK) Debug(args ...interface{}) {
	log(zapcore.DebugLevel, args)
}

func (m *LogSDK) Info(args ...interface{}) {
	log(zapcore.InfoLevel, args)
}

func (m *LogSDK) Warn(args ...interface{}) {
	log(zapcore.WarnLevel, args)
}

func (m *LogSDK) Error(args ...interface{}) {
	log(zapcore.ErrorLevel, args)
}
