// Package httpgen derives HTTP routes from a proto service's RPC naming
// convention and generates the HTTP adapter for whichever framework a
// project selected (ARCHITECTURE.md §8.1).
//
// There is no google.api.http support yet (see ARCHITECTURE.md §12), so
// routes come from matching RPC name prefixes against the
// Create/Get/List/Update/Delete convention `sgo generate proto`'s
// starter template already commits to. This is shared across every
// framework generator; only the Go code emitted for a given route
// differs per framework.
package httpgen

import (
	"strings"

	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

// Route is one derived HTTP endpoint for a single RPC method.
type Route struct {
	Method   sgoproto.Method
	Verb     string // "POST", "GET", "PUT", "DELETE"
	BasePath string // e.g. "/api/v1/users"
	HasID    bool   // true if the id path segment should be appended and bound to req.Id
	HasBody  bool   // true if the request should be decoded from the request body
}

// BuildRoutes derives one Route per RPC on f's (single) service.
func BuildRoutes(f *sgoproto.File, entity string) []Route {
	svc, ok := primaryService(f)
	if !ok {
		return nil
	}

	base := "/api/v1/" + strings.ToLower(entity) + "s"

	routes := make([]Route, 0, len(svc.Methods))
	for _, m := range svc.Methods {
		r := Route{Method: m, BasePath: base}

		switch {
		case strings.HasPrefix(m.Name, "Create"):
			r.Verb = "POST"
			r.HasBody = true
		case strings.HasPrefix(m.Name, "List"):
			r.Verb = "GET"
		case strings.HasPrefix(m.Name, "Get"):
			r.Verb = "GET"
			r.HasID = hasIDField(f, m.Input)
		case strings.HasPrefix(m.Name, "Update"):
			r.Verb = "PUT"
			r.HasID = hasIDField(f, m.Input)
			r.HasBody = true
		case strings.HasPrefix(m.Name, "Delete"):
			r.Verb = "DELETE"
			r.HasID = hasIDField(f, m.Input)
		default:
			// No naming convention matched (e.g. a custom RPC like
			// ArchiveUser): fall back to POST, keyed by id if the
			// input has one, decoding whatever body is sent.
			r.Verb = "POST"
			r.HasID = hasIDField(f, m.Input)
			r.HasBody = true
		}

		routes = append(routes, r)
	}

	return routes
}

// FullPath returns r's path with the id placeholder in the given
// framework's own syntax (":id" for gin/echo, "{id}" for chi), or just
// BasePath if the route has no id.
func (r Route) FullPath(idPlaceholder string) string {
	if !r.HasID {
		return r.BasePath
	}
	return r.BasePath + "/" + idPlaceholder
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
