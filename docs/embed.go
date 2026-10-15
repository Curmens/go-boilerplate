package docs

import _ "embed"

// OpenAPISpec is the raw OpenAPI 3 YAML spec, embedded at compile time.
//
//go:embed openapi.yaml
var OpenAPISpec []byte
