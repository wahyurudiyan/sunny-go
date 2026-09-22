package core

import (
	"strings"

	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

// ResponseShape derives how a transport adapter (HTTP, gRPC) should wrap
// an application service call's result into the proto-declared Response
// envelope, since the application service itself returns a bare
// aggregate (or slice, or nothing) rather than that envelope
// (GenerateApplicationService's returnType — application.go — decides
// what the service method returns; Kind here follows the exact same
// RPC-name-prefix convention so the two always agree).
//
// Kind is one of "single" (wrap the one returned aggregate), "list"
// (wrap the returned slice, plus a total count if the Response message
// declares one), or "delete" (no aggregate returned at all — Delete*).
// The *Field values are outputMsg's own field names — Field/TotalField
// as the proto field name (a JSON key, for httpgen's map-based
// envelope), FieldGoName/TotalFieldGoName as the corresponding Go
// struct field name (for grpcgen's real wire.Output{...} struct
// literal) — so the shape a caller builds from them still matches what
// the proto (and the OpenAPI doc generated from it) declares. Field
// falls back to a lowercase-entity-name convention (FieldGoName to its
// title-cased form) if outputMsg doesn't declare a matching field,
// reachable only via a hand-edited proto that doesn't follow
// `sgo generate proto`'s starter shape. SuccessField/SuccessFieldGoName
// (Kind == "delete" only) name a bool field on outputMsg, if it
// declares one — "" if not, in which case a caller returns a bare zero
// value envelope.
type ResponseShape struct {
	Kind               string
	Field              string
	FieldGoName        string
	TotalField         string
	TotalFieldGoName   string
	SuccessField       string
	SuccessFieldGoName string
}

// ClassifyResponse computes rpcName's ResponseShape against f's IR,
// treating aggregateName as the domain type the application service
// returns (the naming-convention aggregate — the entity title-cased —
// since a caller with only the IR, not fd, can't read an explicit
// `(sgo.aggregate_root)` override; callers that already resolved the
// aggregate via AggregateRoot(fd, entity) should pass its real name).
func ClassifyResponse(f *sgoproto.File, aggregateName, rpcName, outputMsg string) ResponseShape {
	if strings.HasPrefix(rpcName, "Delete") {
		shape := ResponseShape{Kind: "delete"}
		if fld := findField(f, outputMsg, func(fld sgoproto.Field) bool {
			return !fld.IsMessage() && fld.Kind == sgoproto.KindBool
		}); fld != nil {
			shape.SuccessField, shape.SuccessFieldGoName = fld.Name, fld.GoName
		}
		return shape
	}

	kind := "single"
	if strings.HasPrefix(rpcName, "List") {
		kind = "list"
	}
	wantRepeated := kind == "list"

	shape := ResponseShape{Kind: kind}
	if fld := findField(f, outputMsg, func(fld sgoproto.Field) bool {
		return fld.IsMessage() && fld.MessageType == aggregateName && fld.Repeated == wantRepeated
	}); fld != nil {
		shape.Field, shape.FieldGoName = fld.Name, fld.GoName
	} else {
		shape.Field = strings.ToLower(aggregateName)
		shape.FieldGoName = aggregateName
		if wantRepeated {
			shape.Field += "s"
			shape.FieldGoName += "s"
		}
	}

	if kind == "list" {
		if fld := findField(f, outputMsg, func(fld sgoproto.Field) bool {
			return !fld.IsMessage() && fld.GoName == "Total"
		}); fld != nil {
			shape.TotalField, shape.TotalFieldGoName = fld.Name, fld.GoName
		}
	}

	return shape
}

func findField(f *sgoproto.File, msgName string, match func(sgoproto.Field) bool) *sgoproto.Field {
	msg := f.FindMessage(msgName)
	if msg == nil {
		return nil
	}
	for i, fld := range msg.Fields {
		if match(fld) {
			return &msg.Fields[i]
		}
	}
	return nil
}
