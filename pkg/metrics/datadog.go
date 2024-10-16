package metrics

import (
	"context"
	"fmt"
	"github.com/go-kit/kit/metrics/dogstatsd"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/pkg/safe"
	"go.uber.org/zap"
	"strings"
	"time"
)

type logAdapter struct {
	log *zap.SugaredLogger
}

func (log logAdapter) Log(keyvals ...interface{}) error {
	log.log.Warnw("", keyvals...)
	return nil
}

func SetupDataDog(ctx context.Context, cfg *config.Datadog, metrics *Metrics) error {
	prefix := cfg.Prefix
	if prefix != "" {
		prefix = fmt.Sprintf("%s.", prefix)
	}
	datadogClient := dogstatsd.New(prefix, &logAdapter{log: zap.S()})

	// proxy
	metrics.RequestCount = datadogClient.NewCounter("request.total", 1.0)
	metrics.RequestDuration = datadogClient.NewHistogram("request.duration", 1.0)

	// runtime
	metrics.Runtime.Goroutine = datadogClient.NewGauge("runtime.goroutine")

	// worker
	metrics.AttemptsTotal = datadogClient.NewCounter("attempts.total", 1.0)
	metrics.AttemptsFailed = datadogClient.NewCounter("attempts_failed.total", 1.0)
	metrics.AttemptsResponseDuration = datadogClient.NewHistogram("attempts.response.duration", 1.0)

	safe.Go(func() {
		protocol, addr := parseAddress(cfg.Address)
		ticker := time.NewTicker(time.Second * time.Duration(cfg.Interval))
		defer ticker.Stop()
		datadogClient.SendLoop(ctx, ticker.C, protocol, addr)
	})

	return nil
}

func parseAddress(address string) (protocol string, addr string) {
	if address == "" {
		address = "udp://127.0.0.1:8125"
	}

	parts := strings.Split(address, "://")
	if len(parts) == 2 {
		protocol = parts[0]
		addr = parts[1]
	}

	return
}
