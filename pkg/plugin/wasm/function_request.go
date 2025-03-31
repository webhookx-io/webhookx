package wasm

import (
	"context"
	"encoding/json"
	"github.com/tetratelabs/wazero/api"
	"github.com/webhookx-io/webhookx/pkg/plugin/types"
	"go.uber.org/zap"
)

func GetRequestJSON(ctx context.Context, m api.Module, jsonPtr, jsonSizePtr uint32) Status {
	value, ok := fromContext(ctx)
	if !ok {
		zap.S().Error("[wasm] invalid context")
		return StatusInternalFailure
	}

	bytes, err := json.Marshal(value)
	if err != nil {
		zap.S().Errorf("[wasm] failed to marshal value: %v", err)
		return StatusInternalFailure
	}

	allocate := m.ExportedFunction("allocate")
	if allocate == nil {
		zap.S().Error("[wasm] exported function 'allocate' is not defined")
		return StatusInternalFailure
	}

	str := string(bytes)
	ptr, err := writeString(ctx, m.Memory(), allocate, str)
	if err != nil {
		return StatusInvalidMemoryAccess
	}
	if ptr == 0 {
		zap.S().Error("[wasm] exported function 'allocate' returned 0")
		return StatusInvalidMemoryAccess
	}

	if !m.Memory().WriteUint32Le(jsonPtr, ptr) {
		return StatusInvalidMemoryAccess
	}
	if !m.Memory().WriteUint32Le(jsonSizePtr, uint32(len(str))) {
		return StatusInvalidMemoryAccess
	}

	return StatusOk
}

func SetRequestJSON(ctx context.Context, m api.Module, jsonPtr, jsonSize uint32) Status {
	str, ok := readString(m.Memory(), jsonPtr, jsonSize)
	if !ok {
		return StatusInvalidMemoryAccess
	}

	var value types.Request
	if err := json.Unmarshal([]byte(str), &value); err != nil {
		return StatusInvalidJSON
	}

	req, ok := fromContext(ctx)
	if !ok {
		return StatusInternalFailure
	}

	req.URL = value.URL
	req.Headers = value.Headers
	req.Method = value.Method
	req.Payload = value.Payload

	return StatusOk
}
