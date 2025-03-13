package log

import (
	"github.com/webhookx-io/webhookx/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewZapLogger(cfg *config.LogConfig) (*zap.Logger, error) {
	level, err := zapcore.ParseLevel(string(cfg.Level))
	if err != nil {
		return nil, err
	}

	encodingMap := map[string]string{
		"text": "console",
		"json": "json",
	}
	encoderMap := map[string]zapcore.EncoderConfig{
		"text": zap.NewDevelopmentEncoderConfig(),
		"json": zap.NewProductionEncoderConfig(),
	}
	zapConfig := zap.Config{
		Level:             zap.NewAtomicLevelAt(level),
		Development:       false,
		Encoding:          encodingMap[string(cfg.Format)],
		EncoderConfig:     encoderMap[string(cfg.Format)],
		DisableStacktrace: true,
	}
	if len(cfg.File) > 0 {
		zapConfig.OutputPaths = []string{cfg.File}
	}
	return zapConfig.Build()
}
