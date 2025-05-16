package accesslog

import (
	"github.com/rs/zerolog"
	"io"
)

type JsonLogger struct {
	logger *zerolog.Logger
}

func NewJsonLogger(name string, writer io.Writer) *JsonLogger {
	zerolog.TimeFieldFormat = "2006/01/02 15:04:05.000"
	zerolog.TimestampFieldName = "ts"
	logger := zerolog.New(writer).With().Str("name", name).Logger()

	return &JsonLogger{
		logger: &logger,
	}
}

func (l *JsonLogger) Log(entry *Entry) {
	l.logger.Log().Timestamp().EmbedObject(entry).Send()
}
