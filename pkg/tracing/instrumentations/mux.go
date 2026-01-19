package instrumentations

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/webhookx-io/webhookx/pkg/tracing"
)

type InstrumentedMux struct{}

func NewInstrumentedMux() *InstrumentedMux {
	return &InstrumentedMux{}
}

func (m *InstrumentedMux) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := mux.CurrentRoute(r)
		if route == nil {
			next.ServeHTTP(w, r)
			return
		}

		name := route.GetName()
		if name == "" {
			tpl, _ := route.GetPathTemplate()
			name = fmt.Sprintf("%s %s", r.Method, tpl)
		}
		ctx, span := tracing.Start(r.Context(), name)
		defer span.End()

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
