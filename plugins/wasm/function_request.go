package wasm

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/tetratelabs/wazero/api"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"go.uber.org/zap"
)

type Request struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Payload string            `json:"payload"`
}

func toRequest(c *plugin.Context) *Request {
	r := c.Request
	req := &Request{
		URL:     r.URL.String(),
		Method:  r.Method,
		Headers: make(map[string]string),
	}
	for header := range r.Header {
		req.Headers[header] = r.Header.Get(header)
	}
	req.Payload = string(c.GetRequestBody())
	return req
}

func GetRequestJSON(ctx context.Context, m api.Module, jsonPtr, jsonSizePtr uint32) Status {
	value, ok := fromContext(ctx)
	if !ok {
		zap.S().Error("[wasm] invalid context")
		return StatusInternalFailure
	}

	bytes, err := json.Marshal(toRequest(value))
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

	var req Request
	if err := json.Unmarshal([]byte(str), &req); err != nil {
		return StatusInvalidJSON
	}

	c, ok := fromContext(ctx)
	if !ok {
		return StatusInternalFailure
	}

	u, err := url.Parse(req.URL)
	if err != nil {
		return StatusInternalFailure
	}
	c.Request.URL = u
	c.Request.Method = req.Method
	for k, v := range req.Headers {
		c.Request.Header.Set(k, v)
	}
	c.SetRequestBody([]byte(req.Payload))

	return StatusOk
}
