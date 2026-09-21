package openapigen

import (
	"embed"

	"github.com/wahyurudiyan/sunny-go/internal/config"
)

//go:embed schemas/openapi-3.0.json schemas/openapi-3.1.json
var metaSchemaFS embed.FS

// metaSchemaPath is the embedded-FS path and the resource URL the
// checker registers it under (an internal label for $ref resolution,
// not a network address — see schemas/NOTICE.md) for each supported
// version.
var metaSchemaPath = map[config.OpenAPIVersion]string{
	config.OpenAPIVersion30: "schemas/openapi-3.0.json",
	config.OpenAPIVersion31: "schemas/openapi-3.1.json",
}
