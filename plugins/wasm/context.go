package wasm

import (
	"context"

	"github.com/webhookx-io/webhookx/pkg/plugin"
)

type key struct{}

func withContext(ctx context.Context, val *plugin.Context) context.Context {
	return context.WithValue(ctx, key{}, val)
}

func fromContext(ctx context.Context) (*plugin.Context, bool) {
	value, ok := ctx.Value(key{}).(*plugin.Context)
	return value, ok
}
