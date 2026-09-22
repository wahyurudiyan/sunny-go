package httpgen

import (
	"strings"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

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
	HasID          bool
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

func buildRoutesData(f *sgoproto.File, p core.Paths, idPlaceholder string) routesData {
	routes := BuildRoutes(f, p.Entity)
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
			Path:      r.FullPath(idPlaceholder),
			HasBody:   r.HasBody,
			HasID:     r.HasID,
			Name:      r.Method.Name,
			Input:     r.Method.Input,
			Output:    r.Method.Output,
		}
		tr.RespKind, tr.RespField, tr.RespTotalField = classifyResponse(f, aggregateName, r.Method.Name, r.Method.Output)
		data.Routes = append(data.Routes, tr)
	}

	return data
}

// classifyResponse derives how a route's handler should JSON-wrap its
// application service call's result. kind follows the exact same
// RPC-name-prefix convention core.returnType (application.go) uses to
// decide what the service method itself returns, so the two always
// agree: "list" for List*, "delete" (no aggregate at all) for Delete*,
// "single" otherwise. field/totalField are the outputMsg's own field
// names (falling back to a lowercase-entity-name convention if outputMsg
// doesn't declare a matching field — reachable only via a hand-edited
// proto that doesn't follow `sgo generate proto`'s starter shape).
func classifyResponse(f *sgoproto.File, aggregateName, rpcName, outputMsg string) (kind, field, totalField string) {
	switch {
	case strings.HasPrefix(rpcName, "List"):
		kind = "list"
	case strings.HasPrefix(rpcName, "Delete"):
		return "delete", "", ""
	default:
		kind = "single"
	}

	wantRepeated := kind == "list"
	field = findAggregateField(f, outputMsg, aggregateName, wantRepeated)
	if field == "" {
		field = strings.ToLower(aggregateName)
		if wantRepeated {
			field += "s"
		}
	}
	if kind == "list" {
		totalField = findTotalField(f, outputMsg)
	}
	return kind, field, totalField
}

func findAggregateField(f *sgoproto.File, outputMsg, aggregateName string, repeated bool) string {
	msg := f.FindMessage(outputMsg)
	if msg == nil {
		return ""
	}
	for _, fld := range msg.Fields {
		if fld.IsMessage() && fld.MessageType == aggregateName && fld.Repeated == repeated {
			return fld.Name
		}
	}
	return ""
}

func findTotalField(f *sgoproto.File, outputMsg string) string {
	msg := f.FindMessage(outputMsg)
	if msg == nil {
		return ""
	}
	for _, fld := range msg.Fields {
		if !fld.IsMessage() && fld.GoName == "Total" {
			return fld.Name
		}
	}
	return ""
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
