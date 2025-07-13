package webhookx

import _ "embed"

// OpenAPI is the embedded OpenAPI specification file.
//
//go:embed openapi.yml
var OpenAPI []byte
