package openapi

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/webhookx-io/webhookx/utils"
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
		// if err == routers.ErrPathNotFound {
		// 	w.WriteHeader(http.StatusNotFound)
		// 	w.Write([]byte(err.Error()))
		// 	return
		// }
		// if err == routers.ErrMethodNotAllowed {
		// 	w.WriteHeader(http.StatusMethodNotAllowed)
		// 	w.Write([]byte(err.Error()))
		// 	return
		// }
		// w.WriteHeader(http.StatusInternalServerError)
		// w.Write([]byte(fmt.Sprintf("internal error: %s", err.Error())))
		next.ServeHTTP(w, r)
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
		issues := convertError(err, "@body")
		jsonIssues := utils.ConvertJSONPaths(issues)
		bytes, _ := json.Marshal(jsonIssues)
		w.WriteHeader(http.StatusBadRequest)
		w.Write(bytes)
		return
	default:
		w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	next.ServeHTTP(w, r)
}

func convertError(me openapi3.MultiError, pathPrefix string) map[string][]interface{} {
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
					errs := convertError(se, name)
					for k, v := range errs {
						issues[k] = append(issues[k], v...)
					}
				}
				continue
			}

			if err, ok := err.Err.(openapi3.MultiError); ok {
				for k, v := range convertError(err, pathPrefix) {
					issues[k] = append(issues[k], v...)
				}
				continue
			}

			if err.RequestBody != nil {
				issues[pathPrefix] = append(issues[pathPrefix], err.Error())
				continue
			}
		default:
			const unknown = "@unknown"
			issues[unknown] = append(issues[unknown], err.Error())
		}
	}
	return issues
}
