package openapi

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/webhookx-io/webhookx/pkg/errs"
	"github.com/webhookx-io/webhookx/pkg/http/response"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/utils"
	"net/http"
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
		next.ServeHTTP(w, r)
		return
	}

	err = openapi3filter.ValidateRequest(r.Context(), &openapi3filter.RequestValidationInput{
		Request:    r,
		PathParams: pathParams,
		Route:      route,
		Options: &openapi3filter.Options{
			MultiError:         true,
			ExcludeRequestBody: true, // skip request body validate
		},
	})
	switch err := err.(type) {
	case nil:
	case openapi3.MultiError:
		issues := ConvertError(err, "@body")
		jsonIssues := utils.ConvertJSONPaths(issues)
		validateErr := errs.NewValidateFieldsError(errs.ErrRequestValidate, jsonIssues)

		response.JSON(w, 400, types.ErrorResponse{
			Message: "Request Validation",
			Error:   validateErr,
		})
		return
	default:
		response.JSON(w, 400, types.ErrorResponse{
			Message: "Request Validation",
			Error:   err,
		})
		return
	}
	next.ServeHTTP(w, r)
}
