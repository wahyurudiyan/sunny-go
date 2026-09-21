package openapigen

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/httpgen"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
	"github.com/wahyurudiyan/sunny-go/internal/config"
)

// apiVersion is the placeholder Info.Version stamped on every generated
// document. sgo.yaml has no concept of an API version distinct from the
// OpenAPI spec version (config.OpenAPIVersion), and the document is
// fully regenerated on every run (like contract/gen), so a hand-edit
// here wouldn't survive anyway — see ARCHITECTURE.md §13.
const apiVersion = "0.1.0"

// Build walks every service registered in cfg.Services, recompiling
// each one's contract/pb/<name>.proto and deriving its HTTP routes via
// httpgen.BuildRoutes — the exact framework-agnostic function that
// already drives the generated Gin/Echo/Chi adapters — so the resulting
// Document structurally cannot describe an endpoint the generated HTTP
// adapter doesn't actually serve.
func Build(cfg *config.Config, projectDir string) (*Document, error) {
	doc := &Document{
		Info: Info{
			Title:   cfg.Module,
			Version: apiVersion,
		},
		Paths:      map[string]*PathItem{},
		Components: &Components{Schemas: map[string]*Schema{}},
	}

	protoDir := filepath.Join(projectDir, "contract", "pb")

	names := append([]string(nil), cfg.Services...)
	sort.Strings(names)

	for _, name := range names {
		fd, err := sgoproto.Compile(protoDir, name+".proto")
		if err != nil {
			return nil, fmt.Errorf("openapigen: compiling %s.proto: %w", name, err)
		}

		file, err := sgoproto.Build(fd)
		if err != nil {
			return nil, fmt.Errorf("openapigen: building IR for %s: %w", name, err)
		}

		if err := addMessages(doc, file, name); err != nil {
			return nil, err
		}

		for _, r := range httpgen.BuildRoutes(file, name) {
			if err := addRoute(doc, r, name); err != nil {
				return nil, err
			}
		}
	}

	return doc, nil
}

// addMessages registers every message in file as a component schema.
// Two services independently declaring a same-named-but-differently-
// shaped message is a real (if rare) hazard once docs aggregate across
// every registered service into one flat components.schemas namespace —
// this fails loudly instead of silently letting the later one win.
func addMessages(doc *Document, file *sgoproto.File, entity string) error {
	for _, m := range file.Messages {
		schema := messageSchema(m)

		if existing, ok := doc.Components.Schemas[m.Name]; ok && !reflect.DeepEqual(existing, schema) {
			return fmt.Errorf("openapigen: message %q is defined differently by another service's proto (conflict found while processing %q) — rename one of them", m.Name, entity)
		}

		doc.Components.Schemas[m.Name] = schema
	}

	return nil
}

func messageSchema(m sgoproto.Message) *Schema {
	props := make(map[string]*Schema, len(m.Fields))
	for _, f := range m.Fields {
		props[f.Name] = fieldToSchema(f)
	}

	return &Schema{Type: "object", Properties: props}
}

func fieldToSchema(f sgoproto.Field) *Schema {
	var s *Schema
	if f.Kind == sgoproto.KindMessage {
		s = &Schema{Ref: "#/components/schemas/" + f.MessageType}
	} else {
		s = scalarSchema(f.Kind)
	}

	if f.Repeated {
		return &Schema{Type: "array", Items: s}
	}
	return s
}

func scalarSchema(k sgoproto.Kind) *Schema {
	switch k {
	case sgoproto.KindString:
		return &Schema{Type: "string"}
	case sgoproto.KindBool:
		return &Schema{Type: "boolean"}
	case sgoproto.KindInt32:
		return &Schema{Type: "integer", Format: "int32"}
	case sgoproto.KindInt64:
		return &Schema{Type: "integer", Format: "int64"}
	case sgoproto.KindUint32:
		return &Schema{Type: "integer", Format: "int32", Minimum: floatPtr(0)}
	case sgoproto.KindUint64:
		return &Schema{Type: "integer", Format: "int64", Minimum: floatPtr(0)}
	case sgoproto.KindFloat:
		return &Schema{Type: "number", Format: "float"}
	case sgoproto.KindDouble:
		return &Schema{Type: "number", Format: "double"}
	case sgoproto.KindBytes:
		return &Schema{Type: "string", Format: "byte"}
	case sgoproto.KindEnum:
		// Matches Kind.GoType()'s own int32 mapping (ARCHITECTURE.md §12
		// "Enum fields") — sgo doesn't generate typed enum constants yet.
		return &Schema{Type: "integer", Format: "int32"}
	default:
		return &Schema{Type: "string"}
	}
}

func floatPtr(f float64) *float64 { return &f }

// addRoute adds one derived HTTP endpoint to doc, matching exactly what
// the generated Gin/Echo/Chi handler for it does (see
// internal/codegen/httpgen/templates/*_routes.go.tmpl): a body decode
// failure is 400 (only when the route has a body), any service error is
// 500, and success is always 200 with the RPC's output message.
func addRoute(doc *Document, r httpgen.Route, entity string) error {
	path := r.BasePath
	if r.HasID {
		path += "/{id}"
	}

	item, ok := doc.Paths[path]
	if !ok {
		item = &PathItem{}
		doc.Paths[path] = item
	}

	op := &Operation{
		OperationID: r.Method.Name,
		Tags:        []string{entity},
		Responses: map[string]Response{
			"200": {
				Description: "OK",
				Content: map[string]MediaType{
					"application/json": {Schema: &Schema{Ref: "#/components/schemas/" + r.Method.Output}},
				},
			},
			"500": errorResponse("Internal Server Error"),
		},
	}

	if r.HasID {
		op.Parameters = append(op.Parameters, Parameter{
			Name: "id", In: "path", Required: true, Schema: &Schema{Type: "string"},
		})
	}

	if r.HasBody {
		op.RequestBody = &RequestBody{
			Required: true,
			Content: map[string]MediaType{
				"application/json": {Schema: &Schema{Ref: "#/components/schemas/" + r.Method.Input}},
			},
		}
		op.Responses["400"] = errorResponse("Bad Request")
	}

	var slot **Operation
	switch r.Verb {
	case "GET":
		slot = &item.Get
	case "POST":
		slot = &item.Post
	case "PUT":
		slot = &item.Put
	case "DELETE":
		slot = &item.Delete
	default:
		return fmt.Errorf("openapigen: unsupported HTTP verb %q for %s", r.Verb, r.Method.Name)
	}

	if *slot != nil {
		return fmt.Errorf("openapigen: %s %s is served by both %s and %s — two RPCs derive the same route, rename one to fit the Create/Get/List/Update/Delete naming convention", r.Verb, path, (*slot).OperationID, r.Method.Name)
	}
	*slot = op

	return nil
}

func errorResponse(description string) Response {
	return Response{
		Description: description,
		Content: map[string]MediaType{
			"application/json": {Schema: &Schema{
				Type:       "object",
				Properties: map[string]*Schema{"error": {Type: "string"}},
			}},
		},
	}
}
