
all:
	cd assemblyscript && asc index.ts --target release
	cd rust && cargo build --release --target wasm32-unknown-unknown && mv target/wasm32-unknown-unknown/release/index.wasm . && cargo clean
	cd tinygo && tinygo build -scheduler=none -target=wasip1 -buildmode=c-shared -o index.wasm index.go
