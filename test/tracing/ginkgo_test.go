package tracing

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/utils"
	"go.opentelemetry.io/otel/trace"
	"testing"
)

var _ = Describe("tracing disabled", Ordered, func() {
	for _, protocol := range []string{"grpc", "http/protobuf"} {
		Context(protocol, func() {
			var app *app.Application

			entitiesConfig := helper.EntitiesConfig{
				Endpoints: []*entities.Endpoint{factory.EndpointP()},
				Sources:   []*entities.Source{factory.SourceP()},
			}

			BeforeAll(func() {
				helper.InitOtelOutput()
				helper.InitDB(true, &entitiesConfig)

				envs := map[string]string{
					"WEBHOOKX_TRACING_ENABLED":       "false",
					"WEBHOOKX_TRACING_SAMPLING_RATE": "1.0",
				}
				app = utils.Must(helper.Start(envs))
			})

			AfterAll(func() {
				app.Stop()
			})

			It("disabled tracing "+protocol, func() {
				ctx := context.Background()
				tCtx, span := tracing.Start(ctx, "test")
				spanCtxValid := span.SpanContext().IsValid()
				assert.False(GinkgoT(), spanCtxValid, "span context should be invalid")
				valid := trace.SpanContextFromContext(tCtx).IsValid()
				assert.False(GinkgoT(), valid, "span context should be invalid")
			})
		})
	}
})

func TestTracing(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tracing Suite")
}
