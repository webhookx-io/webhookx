package ucontext

import (
	"context"
)

type key struct{}

type UContext struct {
	Name        string
	WorkspaceID string
}

func WithContext(ctx context.Context, uctx *UContext) context.Context {
	return context.WithValue(ctx, key{}, uctx)
}

func FromContext(ctx context.Context) (*UContext, bool) {
	value, ok := ctx.Value(key{}).(*UContext)
	return value, ok
}

func GetWorkspaceID(ctx context.Context) string {
	if w, ok := FromContext(ctx); ok {
		return w.WorkspaceID
	}
	return ""
}
