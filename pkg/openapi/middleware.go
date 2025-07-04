package openapi

import (
	_ "embed"
	"encoding/json"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
)

type middleware struct {
	router routers.Router
}

func NewOpenAPIMiddleware(router routers.Router) func(http.Handler) http.Handler {
	h := middleware{
		router: router,
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r, next)
		})
	}
}

func (m *middleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.Handler) {
	route, pathParams, err := m.router.FindRoute(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = openapi3filter.ValidateRequest(r.Context(), &openapi3filter.RequestValidationInput{
		Request:    r,
		PathParams: pathParams,
		Route:      route,
		Options: &openapi3filter.Options{
			MultiError: true,
		},
	})
	switch err := err.(type) {
	case nil:
	case openapi3.MultiError:
		issues := convertError(err)
		bytes, _ := json.MarshalIndent(issues, "", "  ")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(bytes)
		return
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	next.ServeHTTP(w, r)
}
