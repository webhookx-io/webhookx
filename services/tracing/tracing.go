package tracing

import (
	"context"

	"github.com/webhookx-io/webhookx/pkg/tracing"
)

type TracingService struct {
	Tracer *tracing.Tracer
}

func (s *TracingService) Name() string {
	return "tracing"
}

func (s *TracingService) Start() error { return nil }

func (s *TracingService) Stop(ctx context.Context) error {
	return s.Tracer.Stop(ctx)
}
