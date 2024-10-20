package middlewares

import (
	"context"
	"io"

	"github.com/justinas/alice"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/pkg/observability"
	"github.com/webhookx-io/webhookx/pkg/tracing"
)

type ObservabilityManager struct {
	config       *config.TracingConfig
	tracer       *tracing.Tracer
	tracerCloser io.Closer
}

func NewObservabilityManager(cfg *config.TracingConfig) (*ObservabilityManager, error) {
	tracer, closer, err := tracing.Setup(cfg)
	if err != nil {
		return nil, err
	}

	return &ObservabilityManager{
		config:       cfg,
		tracer:       tracer,
		tracerCloser: closer,
	}, nil
}

func (o *ObservabilityManager) IsTracingEnable() bool {
	return o.config != nil && o.tracer != nil
}

func (o *ObservabilityManager) BuildChain(ctx context.Context, entryPointName string) alice.Chain {
	chain := alice.New()

	if o.tracer != nil {
		chain = chain.Append(observability.WrapEntryPointHandler(ctx, o.tracer, entryPointName))
	}
	// TODO: Add more observability handlers here
	return chain
}

func (o *ObservabilityManager) Close() error {
	if o.tracerCloser != nil {
		return o.tracerCloser.Close()
	}
	return nil
}

func (o *ObservabilityManager) Tracer() *tracing.Tracer {
	return o.tracer
}
