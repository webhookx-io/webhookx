package wasm

import (
	"context"
	"github.com/webhookx-io/webhookx/pkg/plugin/types"
)

type key struct{}

func withContext(ctx context.Context, val *types.Request) context.Context {
	return context.WithValue(ctx, key{}, val)
}

func fromContext(ctx context.Context) (*types.Request, bool) {
	value, ok := ctx.Value(key{}).(*types.Request)
	return value, ok
}
