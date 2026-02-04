package services

import "context"

type Service interface {
	Name() string
	Start() error
	Stop(ctx context.Context) error
}
