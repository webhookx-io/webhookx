// tinygo build -scheduler=none -target=wasip1 -buildmode=c-shared -o index.wasm index.go
package main

import "C"

import (
	"unsafe"
)

const OK = 0

type LogLevel int

const (
	Debug LogLevel = iota
	Info
	Warn
	Error
)

func main() {}

//go:wasmimport env get_request_json
func getRequestJson(returnValueData uintptr, returnValueSize uintptr) int32

//go:wasmimport env set_request_json
func setRequestJson(valueData uintptr, valueSize uint32) int32

//go:wasmimport env log
func log(logLevel int32, strValue uint32, strSize uint32) int32

func stringToPtr(s string) (uint32, uint32) {
	ptr := unsafe.Pointer(unsafe.StringData(s))
	return uint32(uintptr(ptr)), uint32(len(s))
}

func ptrToString(ptr uint32, size uint32) string {
	return unsafe.String((*byte)(unsafe.Pointer(uintptr(ptr))), size)
}

// Log a string
func logString(level LogLevel, str string) {
	ptr, size := stringToPtr(str)
	log(int32(level), ptr, size)
}

//go:wasmexport allocate
func allocate(size int32) *byte {
	buf := make([]byte, size)
	return &buf[0]
}

//go:wasmexport transform
func transform() int32 {
	var requestJsonPtr uintptr  // var to store json pointer
	var requestJsonSize uintptr // var to store json size

	status := getRequestJson(uintptr(unsafe.Pointer(&requestJsonPtr)), uintptr(unsafe.Pointer(&requestJsonSize)))
	if status != OK {
		return 0
	}

	requestJson := ptrToString(*(*uint32)(unsafe.Pointer(&requestJsonPtr)), *(*uint32)(unsafe.Pointer(&requestJsonSize)))

	logString(Debug, requestJson)
	logString(Info, "a info message")
	logString(Warn, "a warn message")
	logString(Error, "a error message")

	json := `{"url":"https://httpbin.org/anything","method":"POST","headers":{"foo":"bar"},"payload":"{}"}`
	jsonPtr, jsonSize := stringToPtr(json)
	status = setRequestJson(uintptr(jsonPtr), jsonSize)
	if status != OK {
		return 0
	}

	return 1
}
