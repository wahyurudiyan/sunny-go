package openapigen

// Document is the root of an OpenAPI document. One shared model serves
// both OpenAPI 3.0.x and 3.1.x (ARCHITECTURE.md §13): the two versions
// are close enough structurally for what sgo's proto-derived schemas
// actually need — plain scalars, arrays, and $ref'd object schemas,
// never a nullable field (proto3's Kind has no "optional" concept in
// this IR yet, see internal/codegen/proto's Field) — that a single
// Schema type serialized the same way is valid under either version's
// meta-schema. Encode (encode.go) only changes the top-level `openapi:`
// version string.
type Document struct {
	OpenAPI    string               `yaml:"openapi" json:"openapi"`
	Info       Info                 `yaml:"info" json:"info"`
	Paths      map[string]*PathItem `yaml:"paths" json:"paths"`
	Components *Components          `yaml:"components,omitempty" json:"components,omitempty"`
}

// Info is the document's required Info Object.
type Info struct {
	Title   string `yaml:"title" json:"title"`
	Version string `yaml:"version" json:"version"`
}

// PathItem holds the operations for one path, keyed by HTTP method.
type PathItem struct {
	Get    *Operation `yaml:"get,omitempty" json:"get,omitempty"`
	Post   *Operation `yaml:"post,omitempty" json:"post,omitempty"`
	Put    *Operation `yaml:"put,omitempty" json:"put,omitempty"`
	Delete *Operation `yaml:"delete,omitempty" json:"delete,omitempty"`
}

// Operation is one RPC's HTTP endpoint.
type Operation struct {
	OperationID string              `yaml:"operationId" json:"operationId"`
	Summary     string              `yaml:"summary,omitempty" json:"summary,omitempty"`
	Tags        []string            `yaml:"tags,omitempty" json:"tags,omitempty"`
	Parameters  []Parameter         `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	RequestBody *RequestBody        `yaml:"requestBody,omitempty" json:"requestBody,omitempty"`
	Responses   map[string]Response `yaml:"responses" json:"responses"`
}

// Parameter is a path or query parameter.
type Parameter struct {
	Name     string  `yaml:"name" json:"name"`
	In       string  `yaml:"in" json:"in"` // "path" or "query"
	Required bool    `yaml:"required" json:"required"`
	Schema   *Schema `yaml:"schema,omitempty" json:"schema,omitempty"`
}

// RequestBody is an operation's JSON request body.
type RequestBody struct {
	Required bool                 `yaml:"required,omitempty" json:"required,omitempty"`
	Content  map[string]MediaType `yaml:"content" json:"content"`
}

// Response is one status code's response.
type Response struct {
	Description string               `yaml:"description" json:"description"`
	Content     map[string]MediaType `yaml:"content,omitempty" json:"content,omitempty"`
}

// MediaType is one content-type's schema within a request/response body.
type MediaType struct {
	Schema *Schema `yaml:"schema,omitempty" json:"schema,omitempty"`
}

// Components holds reusable schema definitions, referenced by $ref.
type Components struct {
	Schemas map[string]*Schema `yaml:"schemas,omitempty" json:"schemas,omitempty"`
}

// Schema is the subset of the JSON Schema / OAS Schema Object that
// sgo's proto-derived fields need: scalars (with format), arrays
// (Items), objects (Properties), and references to a named component
// schema (Ref). See the package doc comment on Document for why this
// doesn't need a version-specific nullable/type-array encoding yet.
type Schema struct {
	Ref        string             `yaml:"$ref,omitempty" json:"$ref,omitempty"`
	Type       string             `yaml:"type,omitempty" json:"type,omitempty"`
	Format     string             `yaml:"format,omitempty" json:"format,omitempty"`
	Minimum    *float64           `yaml:"minimum,omitempty" json:"minimum,omitempty"`
	Items      *Schema            `yaml:"items,omitempty" json:"items,omitempty"`
	Properties map[string]*Schema `yaml:"properties,omitempty" json:"properties,omitempty"`

	// Sensitive and ObfuscateVisible are documentation-only vendor
	// extensions (ARCHITECTURE.md §22) for a field marked (sgo.pii) or
	// (sgo.obfuscate_visible) — never set on a bare $ref schema (see
	// fieldToSchema), so they're always valid alongside this type's
	// other fields under both OpenAPI 3.0 and 3.1.
	Sensitive        bool   `yaml:"x-sensitive,omitempty" json:"x-sensitive,omitempty"`
	ObfuscateVisible *int32 `yaml:"x-obfuscate-visible,omitempty" json:"x-obfuscate-visible,omitempty"`
}
