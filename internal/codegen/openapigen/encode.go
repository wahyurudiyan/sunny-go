package openapigen

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/wahyurudiyan/sunny-go/internal/config"
)

// specVersion is the exact `openapi:` field value stamped for each
// supported config.OpenAPIVersion — the field must be a full version
// (matching each meta-schema's own pattern, e.g. "3.0.3" not just
// "3.0"), not the family selector config.OpenAPIVersion itself.
var specVersion = map[config.OpenAPIVersion]string{
	config.OpenAPIVersion30: "3.0.3",
	config.OpenAPIVersion31: "3.1.0",
}

// Ext is the file extension (no leading dot) generated docs get for
// format.
func Ext(format config.OpenAPIFormat) string {
	if format == config.OpenAPIFormatJSON {
		return "json"
	}
	return "yaml"
}

// Encode marshals doc for version/format, stamping doc.OpenAPI as a
// side effect so callers don't need to set it themselves.
func Encode(doc *Document, version config.OpenAPIVersion, format config.OpenAPIFormat) ([]byte, error) {
	v, ok := specVersion[version]
	if !ok {
		return nil, fmt.Errorf("openapigen: unsupported OpenAPI version %q", version)
	}
	doc.OpenAPI = v

	switch format {
	case config.OpenAPIFormatJSON:
		data, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("openapigen: encoding JSON: %w", err)
		}
		return append(data, '\n'), nil
	case config.OpenAPIFormatYAML:
		data, err := yaml.Marshal(doc)
		if err != nil {
			return nil, fmt.Errorf("openapigen: encoding YAML: %w", err)
		}
		return data, nil
	default:
		return nil, fmt.Errorf("openapigen: unsupported format %q", format)
	}
}
