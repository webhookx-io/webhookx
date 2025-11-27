package contextx

import (
	"context"
)

type key struct{}

type Context struct {
	WorkspaceID   string
	WorkspaceName string
}

func WithContext(ctx context.Context, v *Context) context.Context {
	return context.WithValue(ctx, key{}, v)
}

func FromContext(ctx context.Context) (*Context, bool) {
	value, ok := ctx.Value(key{}).(*Context)
	return value, ok
}

func GetWorkspaceID(ctx context.Context) string {
	if w, ok := FromContext(ctx); ok {
		return w.WorkspaceID
	}
	return ""
}
