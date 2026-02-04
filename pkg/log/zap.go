package log

import (
	"fmt"
	"time"

	"github.com/webhookx-io/webhookx/config/modules"
	"github.com/webhookx-io/webhookx/utils"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewZapLogger(cfg *modules.LogConfig) (*zap.SugaredLogger, error) {
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
		DisableCaller:     true,
		DisableStacktrace: true,
		Encoding:          encodingMap[string(cfg.Format)],
		EncoderConfig:     encoderMap[string(cfg.Format)],
	}
	zapConfig.EncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006/01/02 15:04:05.000"))
	}
	if cfg.Format == modules.LogFormatText {
		zapConfig.EncoderConfig.EncodeName = func(loggerName string, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(fmt.Sprintf("%-8s", "["+loggerName+"]"))
		}
		if cfg.Colored {
			zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}
		zapConfig.EncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(utils.Colorize(t.Format("2006/01/02 15:04:05.000"), utils.ColorDarkGray, cfg.Colored))
		}
	}

	if cfg.File == "" {
		zapConfig.OutputPaths = []string{"/dev/stdout"}
	} else {
		zapConfig.OutputPaths = []string{cfg.File}
	}

	logger, err := zapConfig.Build()
	if err != nil {
		return nil, err
	}

	zap.ReplaceGlobals(logger)

	return logger.Sugar(), nil
}
