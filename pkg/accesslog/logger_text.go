package accesslog

import (
	"context"
	"io"

	"github.com/rs/zerolog"
	"github.com/webhookx-io/webhookx/utils"
)

type TextLogger struct {
	logger *zerolog.Logger
}

func NewTextLogger(name string, writer io.Writer, colored bool) *TextLogger {
	zerolog.TimeFieldFormat = "2006/01/02 15:04:05.000"
	zerolog.TimestampFieldName = "ts"

	output := zerolog.ConsoleWriter{
		Out:        writer,
		NoColor:    !colored,
		TimeFormat: "2006/01/02 15:04:05.000",
	}
	output.PartsOrder = []string{
		zerolog.TimestampFieldName,
		"name",
		zerolog.LevelFieldName,
		zerolog.CallerFieldName,
		zerolog.MessageFieldName,
	}
	output.FieldsExclude = []string{"name"}
	output.FormatLevel = func(i interface{}) string { return "" }
	output.FormatFieldName = func(i interface{}) string { return "" }
	name = utils.Colorize("["+name+"]", utils.ColorDarkGray, colored)
	logger := zerolog.New(output).With().Str("name", name).Logger()
	return &TextLogger{
		logger: &logger,
	}
}

func (l *TextLogger) Log(ctx context.Context, entry *Entry) {
	e := l.logger.Log().Ctx(ctx).Timestamp()
	msg := entry.String(e)
	e.Msg(msg)
}
