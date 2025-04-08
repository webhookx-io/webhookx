// makes golangci-lint happy
//go:build wasip1

package main

import (
	"encoding/json"
	"os"
	"unsafe"
)

const OK = 0

type LogLevel = int32

const (
	Debug LogLevel = iota
	Info
	Warn
	Error
)

func main() {}

//go:wasmimport env get_request_json
func getRequestJson(returnValueData unsafe.Pointer, returnValueSize unsafe.Pointer) int32

//go:wasmimport env set_request_json
func setRequestJson(value string) int32

//go:wasmimport env log
func log(logLevel int32, str string) int32

func ptrToString(p uintptr, size int32) string {
	return unsafe.String((*byte)(unsafe.Pointer(p)), size)
}

//go:wasmexport allocate
func allocate(size int32) *byte {
	buf := make([]byte, size)
	return &buf[0]
}

//go:wasmexport transform
func transform() int32 {
	var requestJsonPtr int32  // var to store json pointer
	var requestJsonSize int32 // var to store json size

	status := getRequestJson(unsafe.Pointer(&requestJsonPtr), unsafe.Pointer(&requestJsonSize))
	if status != OK {
		return 0
	}

	// cast to string
	requestJson := ptrToString(uintptr(requestJsonPtr), requestJsonSize)
	request := make(map[string]interface{})
	if err := json.Unmarshal([]byte(requestJson), &request); err != nil {
		return 0
	}

	if headers, ok := request["headers"].(map[string]interface{}); ok {
		// add a custom header
		log(Debug, "setting headers[x-wasm-transform] = true")
		headers["x-wasm-transform"] = "true"
		if os.Getenv("secret") != "" {
			log(Debug, "setting headers[x-wasm-secret] = "+os.Getenv("secret"))
			headers["x-wasm-secret"] = os.Getenv("secret")
		}
	}

	bytes, _ := json.Marshal(request)

	status = setRequestJson(string(bytes))
	if status != OK {
		return 0
	}

	return 1
}
