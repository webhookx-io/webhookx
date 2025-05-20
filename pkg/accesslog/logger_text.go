package accesslog

import (
	"fmt"
	"github.com/rs/zerolog"
	"io"
	"os"
)

type TextLogger struct {
	logger *zerolog.Logger
}

func NewTextLogger(name string, writer io.Writer) *TextLogger {
	zerolog.TimeFieldFormat = "2006/01/02 15:04:05.000"
	zerolog.TimestampFieldName = "ts"

	output := zerolog.ConsoleWriter{
		Out:        writer,
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
	name = colorize("["+name+"]", 90)
	logger := zerolog.New(output).With().Str("name", name).Logger()
	return &TextLogger{
		logger: &logger,
	}
}

func colorize(s interface{}, c int) string {
	if os.Getenv("NO_COLOR") != "" || c == 0 {
		return fmt.Sprintf("%s", s)
	}

	return fmt.Sprintf("\x1b[%dm%v\x1b[0m", c, s)
}

func (l *TextLogger) Log(entry *Entry) {
	l.logger.Log().Timestamp().Msg(entry.String())
}
