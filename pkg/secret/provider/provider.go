package provider

import (
	"context"
)

type Provider interface {
	GetValue(ctx context.Context, key string, properties map[string]string) (string, error)
}
