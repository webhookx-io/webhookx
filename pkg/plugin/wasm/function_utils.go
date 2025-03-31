package wasm

import (
	"context"
	"errors"
	"github.com/tetratelabs/wazero/api"
)

func writeString(ctx context.Context, memory api.Memory, allocate api.Function, str string) (uint32, error) {
	ptr, err := allocate.Call(ctx, uint64(len(str)))
	if err != nil || len(ptr) == 0 {
		return 0, err
	}

	p := uint32(ptr[0])
	if !memory.WriteString(p, str) {
		return 0, errors.New("failed to write string to memory")
	}

	return p, nil
}

func readString(memory api.Memory, ptr uint32, length uint32) (string, bool) {
	buf, ok := memory.Read(ptr, length)
	return string(buf), ok
}
