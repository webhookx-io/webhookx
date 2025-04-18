package wasm

import (
	"context"
	"github.com/webhookx-io/webhookx/pkg/plugin"
)

type key struct{}

func withContext(ctx context.Context, val *plugin.Request) context.Context {
	return context.WithValue(ctx, key{}, val)
}

func fromContext(ctx context.Context) (*plugin.Request, bool) {
	value, ok := ctx.Value(key{}).(*plugin.Request)
	return value, ok
}
