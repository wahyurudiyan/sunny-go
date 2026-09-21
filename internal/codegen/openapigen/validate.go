package openapigen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"

	"github.com/wahyurudiyan/sunny-go/internal/config"
)

// Validate checks data (an OpenAPI document, as either JSON or YAML —
// the "import from json or yaml" requirement, ARCHITECTURE.md §13)
// against the real OpenAPI JSON Schema meta-schema, auto-detected from
// the document's own top-level `openapi:`/`"openapi"` field. This is
// the checker: it validates a document's own well-formedness, not
// whether some project's generated routes match it (see PLAN.md's
// Non-goals).
func Validate(data []byte) error {
	value, err := decode(data)
	if err != nil {
		return fmt.Errorf("openapigen: %w", err)
	}

	version, specVersionString, err := detectVersion(value)
	if err != nil {
		return fmt.Errorf("openapigen: %w", err)
	}

	schema, err := metaSchema(version)
	if err != nil {
		return fmt.Errorf("openapigen: %w", err)
	}

	if err := schema.Validate(value); err != nil {
		return fmt.Errorf("openapigen: document declares openapi %q but is not valid OpenAPI %s:\n%v", specVersionString, version, err)
	}

	return nil
}

// decode parses data as JSON if it looks like JSON (starts with '{'
// once whitespace is trimmed), else as YAML — gopkg.in/yaml.v3 parses
// valid JSON too, but trying JSON first gives a clearer error message
// for documents that are almost-but-not-quite valid JSON.
func decode(data []byte) (any, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty document")
	}

	if trimmed[0] == '{' {
		var v any
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, fmt.Errorf("looks like JSON but failed to parse: %w", err)
		}
		return v, nil
	}

	var v any
	if err := yaml.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("not valid JSON or YAML: %w", err)
	}
	return v, nil
}

// detectVersion reads the decoded document's "openapi" field and maps
// it to the config.OpenAPIVersion family (3.0.x -> "3.0", 3.1.x ->
// "3.1") whose meta-schema should validate it.
func detectVersion(value any) (config.OpenAPIVersion, string, error) {
	obj, ok := value.(map[string]any)
	if !ok {
		return "", "", fmt.Errorf("document root is not an object")
	}

	raw, ok := obj["openapi"]
	if !ok {
		return "", "", fmt.Errorf(`document has no top-level "openapi" field`)
	}

	v, ok := raw.(string)
	if !ok {
		return "", "", fmt.Errorf(`"openapi" field is not a string`)
	}

	switch {
	case strings.HasPrefix(v, "3.0."):
		return config.OpenAPIVersion30, v, nil
	case strings.HasPrefix(v, "3.1."):
		return config.OpenAPIVersion31, v, nil
	default:
		return "", "", fmt.Errorf(`unsupported "openapi" version %q: sgo validates 3.0.x and 3.1.x`, v)
	}
}

var (
	metaSchemaCacheMu sync.Mutex
	metaSchemaCache   = map[config.OpenAPIVersion]*jsonschema.Schema{}
)

// metaSchema compiles (once, cached) the vendored meta-schema for
// version.
func metaSchema(version config.OpenAPIVersion) (*jsonschema.Schema, error) {
	metaSchemaCacheMu.Lock()
	defer metaSchemaCacheMu.Unlock()

	if s, ok := metaSchemaCache[version]; ok {
		return s, nil
	}

	path, ok := metaSchemaPath[version]
	if !ok {
		return nil, fmt.Errorf("unsupported OpenAPI version %q", version)
	}

	raw, err := metaSchemaFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading vendored meta-schema for %s: %w", version, err)
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(path, bytes.NewReader(raw)); err != nil {
		return nil, fmt.Errorf("loading vendored meta-schema for %s: %w", version, err)
	}

	schema, err := compiler.Compile(path)
	if err != nil {
		return nil, fmt.Errorf("compiling vendored meta-schema for %s: %w", version, err)
	}

	metaSchemaCache[version] = schema
	return schema, nil
}
