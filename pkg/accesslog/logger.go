package accesslog

import (
	"errors"
	"io"
	"os"
)

type AccessLogger interface {
	Log(entry *Entry)
}

type Options struct {
	File   string
	Format string
}

func NewAccessLogger(name string, opts Options) (AccessLogger, error) {
	if opts.File == "" {
		return nil, errors.New("accesslog file is required")
	}

	var writer io.Writer = os.Stdout
	if opts.File != "/dev/stdout" {
		file, err := os.OpenFile(opts.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
		if err != nil {
			return nil, err
		}
		writer = file
	}

	switch opts.Format {
	case "text":
		return NewTextLogger(name, writer), nil
	case "json":
		return NewJsonLogger(name, writer), nil
	default:
		return nil, errors.New("invalid format: " + opts.Format)
	}
}
