package webhookx

import (
	_ "embed"
)

var (
	VERSION = "dev"
	COMMIT  = "unknown"
)

//go:embed openapi.yml
var OpenAPI []byte

//go:embed logo.txt
var Logo string
