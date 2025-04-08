# Go example

> Note: Go >= 1.24 is required to compile the example, as it uses features like go:wasmexport and string parameter that were introduced in Go 1.24.

```
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o index.wasm
```
