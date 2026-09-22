package httpgen

import (
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

// templatePathParam is one path-bound field a route's handler must read
// off the URL and assign onto the request/body struct before calling
// the application service — every framework's own URL-param accessor
// (gin/echo's c.Param(urlName), chi's chi.URLParam(req, urlName)) takes
// the same bare field name regardless of the route-registration syntax
// (":name" vs "{name}"), which FullPath has already resolved by the
// time this is built.
type templatePathParam struct {
	URLName string
	GoName  string
}

// templateRoute is the per-route view every framework template renders
// from. TitleVerb ("Get"/"Post"/"Put"/"Delete") is only used by chi,
// whose convenience methods are named that way; gin and echo use Verb
// directly, since their route methods are named GET/POST/PUT/DELETE.
//
// RespKind/RespField/RespTotalField describe how to JSON-wrap the
// application service's return value, since it returns a bare aggregate
// (or slice, or nothing) rather than the proto-declared Response
// envelope message (ARCHITECTURE.md §17 — the application layer has no
// Response DTO concept, only Command/Query). The template builds that
// envelope itself as a plain map, keyed by the Response message's own
// field name, so the JSON on the wire still matches what the proto (and
// the OpenAPI doc generated from it, §13) declares — RespKind is one of
// "single" (wrap the one returned aggregate), "list" (wrap the returned
// slice, plus a total count if the Response message has one), or
// "delete" (no aggregate returned at all; a fixed `{"success": true}`).
type templateRoute struct {
	Verb           string
	TitleVerb      string
	Path           string
	HasBody        bool
	PathParams     []templatePathParam
	Name           string
	Input          string
	Output         string
	RespKind       string
	RespField      string
	RespTotalField string
}

// routesData is what every <framework>_routes.go.tmpl renders from.
type routesData struct {
	Entity                string
	ApplicationPkg        string
	ApplicationImportPath string
	Routes                []templateRoute
}

func buildRoutesData(fd protoreflect.FileDescriptor, f *sgoproto.File, p core.Paths, colonStyle bool) (routesData, error) {
	routes, err := BuildRoutes(fd, f, p.Entity)
	if err != nil {
		return routesData{}, err
	}
	aggregateName := entityTitle(p.Entity)

	data := routesData{
		Entity:                entityTitle(p.Entity),
		ApplicationPkg:        p.Entity,
		ApplicationImportPath: p.ApplicationImportPath(),
	}

	for _, r := range routes {
		tr := templateRoute{
			Verb:      r.Verb,
			TitleVerb: titleVerb(r.Verb),
			Path:      r.FullPath(colonStyle),
			HasBody:   r.HasBody,
			Name:      r.Method.Name,
			Input:     r.Method.Input,
			Output:    r.Method.Output,
		}
		for _, p := range r.PathParams {
			tr.PathParams = append(tr.PathParams, templatePathParam{URLName: p.Name, GoName: p.GoName})
		}
		shape := core.ClassifyResponse(f, aggregateName, r.Method.Name, r.Method.Output)
		tr.RespKind, tr.RespField, tr.RespTotalField = shape.Kind, shape.Field, shape.TotalField
		data.Routes = append(data.Routes, tr)
	}

	return data, nil
}

func titleVerb(verb string) string {
	if verb == "" {
		return verb
	}
	return strings.ToUpper(verb[:1]) + strings.ToLower(verb[1:])
}

func entityTitle(name string) string {
	if name == "" {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}
