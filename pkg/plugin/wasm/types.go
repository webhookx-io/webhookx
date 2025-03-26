package wasm

type Status = int32

const (
	StatusOk                  Status = 0
	StatusInternalFailure     Status = 1
	StatusBadArgument         Status = 2
	StatusInvalidMemoryAccess Status = 3
	StatusInvalidJSON         Status = 11
)

type LogLevel = int

const (
	LogLveDebug LogLevel = 0
	LogLveInfo  LogLevel = 1
	LogLveWarn  LogLevel = 2
	LogLveError LogLevel = 3
)
