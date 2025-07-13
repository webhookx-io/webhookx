package openapi

import (
	_ "embed"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/webhookx-io/webhookx/pkg/errs"
	"github.com/webhookx-io/webhookx/pkg/http/response"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/utils"
	"net/http"
	"strings"
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

func ConvertError(me openapi3.MultiError, pathPrefix string) map[string][]interface{} {
	issues := make(map[string][]interface{})
	for _, err := range me {
		switch err := err.(type) {
		case *openapi3.SchemaError:
			field := pathPrefix
			if path := err.JSONPointer(); len(path) > 0 {
				field = fmt.Sprintf("%s.%s", field, strings.Join(path, "."))
			}
			issues[field] = append(issues[field], err.Reason)
		case *openapi3filter.RequestError:
			if err.Parameter != nil {
				prefix := err.Parameter.In
				name := fmt.Sprintf("@%s.%s", prefix, err.Parameter.Name)
				if se, ok := err.Err.(openapi3.MultiError); ok {
					errs := ConvertError(se, name)
					for k, v := range errs {
						issues[k] = append(issues[k], v...)
					}
				}
				continue
			}

			if err, ok := err.Err.(openapi3.MultiError); ok {
				for k, v := range ConvertError(err, pathPrefix) {
					issues[k] = append(issues[k], v...)
				}
				continue
			}

			if err.RequestBody != nil {
				if se, ok := err.Err.(openapi3.MultiError); ok {
					errs := ConvertError(se, pathPrefix)
					for k, v := range errs {
						issues[k] = append(issues[k], v...)
					}
				} else {
					errs := ConvertError(openapi3.MultiError{err.Err}, pathPrefix)
					for k, v := range errs {
						issues[k] = append(issues[k], v...)
					}
				}
				continue
			}
		default:
			const unknown = "@unknown"
			issues[unknown] = append(issues[unknown], err.Error())
		}
	}
	return issues
}
