// Package httpgen derives HTTP routes from a proto service — a
// `google.api.http` annotation on an RPC if it has one, falling back to
// matching the RPC name against the Create/Get/List/Update/Delete
// naming convention `sgo generate proto`'s starter template commits to
// otherwise (ARCHITECTURE.md §8.1/§20) — and generates the HTTP adapter
// for whichever framework a project selected. This is shared across
// every framework generator; only the Go code emitted for a given route
// differs per framework.
package httpgen

import (
	"fmt"
	"regexp"
	"strings"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/reflect/protoreflect"

	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

// PathParam is one path-bound field: a "{name}" segment in a Route's
// PathTemplate, bound to the named field on the RPC's request message.
type PathParam struct {
	Name   string // proto field name, e.g. "order_id" — also the PathTemplate's "{name}" placeholder and every framework's own URL-param accessor name (gin/echo's c.Param(name), chi's chi.URLParam(req, name)) — frameworks differ in the route-registration placeholder syntax (":name" vs "{name}"), never in the accessor name
	GoName string // Go field name to bind it to on the request struct, e.g. "OrderId"
}

// Route is one derived HTTP endpoint for a single RPC method.
type Route struct {
	Method       sgoproto.Method
	Verb         string      // "GET", "POST", "PUT", "DELETE", "PATCH"
	PathTemplate string      // e.g. "/api/v1/users/{id}" — always "{name}" syntax; FullPath translates it per framework
	PathParams   []PathParam // path-bound fields, in path order
	HasBody      bool        // true if the whole request should be decoded from the request body
}

// FullPath returns r's path template translated to a framework's own
// route-registration syntax: every "{name}" placeholder becomes
// ":name" if colonStyle (gin/echo), or is left as "{name}" otherwise
// (chi already uses this syntax natively — ColonStyle(fw) reports
// which one a given framework needs).
func (r Route) FullPath(colonStyle bool) string {
	if !colonStyle || len(r.PathParams) == 0 {
		return r.PathTemplate
	}
	out := r.PathTemplate
	for _, p := range r.PathParams {
		out = strings.ReplaceAll(out, "{"+p.Name+"}", ":"+p.Name)
	}
	return out
}

var pathVarRe = regexp.MustCompile(`\{([^{}=*]*)([=*][^{}]*)?\}`)

// BuildRoutes derives one Route per RPC on f's (single) service, except
// one marked `option (sgo.hide_route) = true;` (ARCHITECTURE.md §21),
// which gets none at all — its gRPC method and any repository-port
// counterpart are unaffected, only its HTTP route is skipped. Of the
// rest, an RPC carrying a `(google.api.http)` annotation
// (ARCHITECTURE.md §20) uses it; one without falls back to the
// Create/Get/List/Update/Delete naming convention (§8.1), unchanged
// from before this annotation support existed — every proto and every
// generated project that predates it keeps working identically. fd is
// f's underlying descriptor, needed to read the annotation and the
// service's `(sgo.base_path)` override; f's IR has already dropped that
// information by the time it's built.
func BuildRoutes(fd protoreflect.FileDescriptor, f *sgoproto.File, entity string) ([]Route, error) {
	svc, ok := primaryService(f)
	if !ok {
		return nil, nil
	}

	fdSvc := fd.Services().Get(0)
	methodByName := make(map[string]protoreflect.MethodDescriptor, fdSvc.Methods().Len())
	for i := 0; i < fdSvc.Methods().Len(); i++ {
		md := fdSvc.Methods().Get(i)
		methodByName[string(md.Name())] = md
	}

	basePath := sgoproto.BasePath(fdSvc)
	conventionBase := basePath
	if conventionBase == "" {
		conventionBase = "/api/v1"
	}
	conventionBase += "/" + strings.ToLower(entity) + "s"

	routes := make([]Route, 0, len(svc.Methods))
	for _, m := range svc.Methods {
		mtd, ok := methodByName[m.Name]
		if !ok {
			return nil, fmt.Errorf("httpgen: RPC %q has no matching descriptor (compiler/IR mismatch)", m.Name)
		}

		if sgoproto.IsHideRoute(mtd) {
			continue
		}

		if rule := sgoproto.HTTPRule(mtd); rule != nil {
			r, err := routeFromRule(m, rule, f, basePath)
			if err != nil {
				return nil, fmt.Errorf("httpgen: %s: %w", m.Name, err)
			}
			routes = append(routes, r)
			continue
		}

		routes = append(routes, conventionRoute(f, m, conventionBase))
	}

	return routes, nil
}

// conventionRoute derives r the way BuildRoutes always has, before
// google.api.http support existed — unchanged so every proto that
// doesn't use an annotation keeps generating byte-identical routes.
func conventionRoute(f *sgoproto.File, m sgoproto.Method, base string) Route {
	r := Route{Method: m, PathTemplate: base}

	idParam := func() []PathParam {
		if !hasIDField(f, m.Input) {
			return nil
		}
		return []PathParam{{Name: "id", GoName: "Id"}}
	}

	switch {
	case strings.HasPrefix(m.Name, "Create"):
		r.Verb = "POST"
		r.HasBody = true
	case strings.HasPrefix(m.Name, "List"):
		r.Verb = "GET"
	case strings.HasPrefix(m.Name, "Get"):
		r.Verb = "GET"
		r.PathParams = idParam()
	case strings.HasPrefix(m.Name, "Update"):
		r.Verb = "PUT"
		r.PathParams = idParam()
		r.HasBody = true
	case strings.HasPrefix(m.Name, "Delete"):
		r.Verb = "DELETE"
		r.PathParams = idParam()
	default:
		// No naming convention matched (e.g. a custom RPC like
		// ArchiveUser): fall back to POST, keyed by id if the input has
		// one, decoding whatever body is sent.
		r.Verb = "POST"
		r.PathParams = idParam()
		r.HasBody = true
	}

	if len(r.PathParams) > 0 {
		r.PathTemplate += "/{" + r.PathParams[0].Name + "}"
	}

	return r
}

// routeFromRule derives a Route from rule, an RPC's real
// `(google.api.http)` annotation. v1 constraints, stated rather than
// silently mishandled (ARCHITECTURE.md §20): only the get/post/put/
// delete/patch verb fields are supported, not `custom`; only a bare
// "{name}" path variable is supported, not a "*"/"**" wildcard segment
// or a "{name=sub/pattern}" nested pattern, or a dotted field path into
// a nested message; only body:"*" or an empty body are supported, not
// body:"<field>" (binding just one nested field). Each rejects with a
// clear error naming the RPC and what's unsupported, rather than
// guessing at a mapping that could silently misroute requests.
// additional_bindings (multiple HTTP mappings for one RPC) are ignored
// — only the primary rule is used, since Route (and everything
// generated from it) is one route per RPC.
func routeFromRule(m sgoproto.Method, rule *annotations.HttpRule, f *sgoproto.File, basePath string) (Route, error) {
	verb, path, ok := verbAndPath(rule)
	if !ok {
		return Route{}, fmt.Errorf("google.api.http annotation uses an unsupported pattern (only get/post/put/delete/patch are supported, not a custom verb)")
	}

	template, params, err := parsePathTemplate(path)
	if err != nil {
		return Route{}, err
	}

	input := f.FindMessage(m.Input)
	if input == nil {
		return Route{}, fmt.Errorf("request message %q not found (compiler/IR mismatch)", m.Input)
	}
	for i, p := range params {
		field := findScalarField(input, p.Name)
		if field == nil {
			return Route{}, fmt.Errorf("path parameter %q has no matching scalar field on %s (message-typed and repeated fields can't be path parameters)", p.Name, m.Input)
		}
		params[i].GoName = field.GoName
	}

	body := rule.GetBody()
	if body != "" && body != "*" {
		return Route{}, fmt.Errorf("google.api.http body:%q is not supported in v1 (only body:\"*\" or an omitted body are) — binding a single nested field isn't implemented yet", body)
	}

	r := Route{
		Method:       m,
		Verb:         verb,
		PathTemplate: basePath + template,
		PathParams:   params,
		HasBody:      body == "*",
	}
	return r, nil
}

// verbAndPath returns rule's HTTP verb and path template from whichever
// of its get/post/put/delete/patch fields is set. ok is false if none
// are (i.e. only the unsupported `custom` pattern is set).
func verbAndPath(rule *annotations.HttpRule) (verb, path string, ok bool) {
	switch {
	case rule.GetGet() != "":
		return "GET", rule.GetGet(), true
	case rule.GetPost() != "":
		return "POST", rule.GetPost(), true
	case rule.GetPut() != "":
		return "PUT", rule.GetPut(), true
	case rule.GetDelete() != "":
		return "DELETE", rule.GetDelete(), true
	case rule.GetPatch() != "":
		return "PATCH", rule.GetPatch(), true
	default:
		return "", "", false
	}
}

// parsePathTemplate validates path (a google.api.http path template)
// and extracts its bare "{name}" variables in order. Rejects (with a
// clear error) anything beyond the v1-supported shape: a wildcard
// segment ("*"/"**"), or a variable with a "=sub/pattern" or a dotted
// field path — see routeFromRule's doc comment for why.
func parsePathTemplate(path string) (string, []PathParam, error) {
	if strings.Contains(path, "*") {
		return "", nil, fmt.Errorf("google.api.http path %q uses a wildcard segment (\"*\"/\"**\"), not supported in v1", path)
	}

	var params []PathParam
	seen := map[string]bool{}

	matches := pathVarRe.FindAllStringSubmatch(path, -1)
	for _, m := range matches {
		name, subPattern := m[1], m[2]
		if subPattern != "" {
			return "", nil, fmt.Errorf("google.api.http path %q uses a variable sub-pattern (%q), not supported in v1 — only a bare \"{name}\" is", path, "{"+name+subPattern+"}")
		}
		if name == "" || strings.Contains(name, ".") {
			return "", nil, fmt.Errorf("google.api.http path %q uses a variable name (%q) that isn't a bare top-level field, not supported in v1", path, name)
		}
		if seen[name] {
			return "", nil, fmt.Errorf("google.api.http path %q binds %q more than once", path, name)
		}
		seen[name] = true
		params = append(params, PathParam{Name: name})
	}

	return path, params, nil
}

func findScalarField(msg *sgoproto.Message, name string) *sgoproto.Field {
	for i, f := range msg.Fields {
		if f.Name == name {
			if f.IsMessage() || f.Repeated {
				return nil
			}
			return &msg.Fields[i]
		}
	}
	return nil
}

func hasIDField(f *sgoproto.File, messageName string) bool {
	msg := f.FindMessage(messageName)
	if msg == nil {
		return false
	}
	for _, field := range msg.Fields {
		if field.GoName == "Id" {
			return true
		}
	}
	return false
}

func primaryService(f *sgoproto.File) (*sgoproto.Service, bool) {
	if len(f.Services) == 0 {
		return nil, false
	}
	return &f.Services[0], true
}
