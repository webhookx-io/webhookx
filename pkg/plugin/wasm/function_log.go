package wasm

import (
	"context"
	"fmt"
	"github.com/tetratelabs/wazero/api"
	"go.uber.org/zap"
)

func Log(ctx context.Context, m api.Module, logLevel, strValue, strSize uint32) Status {
	str, ok := readString(m.Memory(), strValue, strSize)
	if !ok {
		return StatusInvalidMemoryAccess
	}

	log := zap.S()
	message := fmt.Sprintf("[wasm]: %s", str)

	switch LogLevel(logLevel) {
	case LogLveDebug:
		log.Debug(message)
	case LogLveInfo:
		log.Info(message)
	case LogLveWarn:
		log.Warn(message)
	case LogLveError:
		log.Error(message)
	default:
		return StatusBadArgument
	}

	return StatusOk
}
