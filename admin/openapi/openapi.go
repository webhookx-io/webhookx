package admin

import (
	_ "embed"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

// OpenAPISpec contains the OpenAPI specification for the admin API.
//
//go:embed openapi.yaml
var OpenAPISpec []byte

func NewOpenAPIRouter() (routers.Router, error) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(OpenAPISpec)
	if err != nil {
		return nil, err
	}

	if err = doc.Validate(loader.Context); err != nil {
		return nil, err
	}
	router, err := gorillamux.NewRouter(doc)
	if err != nil {
		return nil, err
	}
	return router, nil
}
