package wasm

import (
	"context"
	"github.com/webhookx-io/webhookx/pkg/plugin"
)

type key struct{}

func withContext(ctx context.Context, val *plugin.OutboundRequest) context.Context {
	return context.WithValue(ctx, key{}, val)
}

func fromContext(ctx context.Context) (*plugin.OutboundRequest, bool) {
	value, ok := ctx.Value(key{}).(*plugin.OutboundRequest)
	return value, ok
}
