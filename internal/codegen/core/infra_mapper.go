package core

import (
	"path/filepath"
	"strings"

	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

// GenerateInfraMapper writes
// internal/infrastructure/transport/<entity>_mapper_gen.go
// (ARCHITECTURE.md §17) — entity-prefixed like every other transport
// file, since (as with the pre-Phase-12 internal/adapter/mapper it
// replaces) every entity's conversions share one package — two families
// of wire conversions, both protovalidate-checked:
//
//   - ToDomain/FromDomain, between contract/gen's wire types and
//     internal/domain's aggregate/value-object/entity structs, for
//     every domain message in f (the Aggregate Root, its Value
//     Objects, Domain Events, and any child entities) — everything
//     GenerateAggregate also generates a domain struct for.
//   - ToApp, from a Request-suffixed DTO's wire type to its
//     application-layer Command/Query counterpart
//     (GenerateCommandsAndQueries's output) — the gRPC adapter's own
//     request-side conversion, since unlike the HTTP adapter (which
//     binds JSON directly into the Command/Query DTO, no wire type
//     involved) a gRPC handler's parameter IS the wire type. Only
//     scalar-shaped fields are copied, the same v1 limitation
//     GenerateCommandsAndQueries itself already documents. Response
//     DTOs get no function here — building the Response envelope is a
//     per-RPC concern the caller (grpcgen) handles itself, the same
//     way httpgen's classifyResponse-derived code does for HTTP.
//
// Replaces internal/adapter/mapper/<entity>_mapper_gen.go's role, with
// one behavioral addition: every conversion now validates the incoming
// wire message (protovalidate) and returns an error, rather than
// unconditionally converting it.
func GenerateInfraMapper(f *sgoproto.File, p Paths, destDir string) error {
	var messages []sgoproto.Message
	var requestDTOs []sgoproto.Message
	for _, m := range f.Messages {
		if !IsDTO(m.Name) {
			messages = append(messages, m)
			continue
		}
		if strings.HasSuffix(m.Name, "Request") {
			requestDTOs = append(requestDTOs, m)
		}
	}

	data := struct {
		WireImportPath        string
		DomainImportPath      string
		ApplicationImportPath string
		Messages              []sgoproto.Message
		RequestDTOs           []sgoproto.Message
	}{
		WireImportPath:        p.WireImportPath(),
		DomainImportPath:      p.AggregateDomainImportPath(),
		ApplicationImportPath: p.ApplicationImportPath(),
		Messages:              messages,
		RequestDTOs:           requestDTOs,
	}

	path := filepath.Join(destDir, p.Entity+"_mapper_gen.go")
	return writeGoFile("templates/infra_mapper_gen.go.tmpl", data, path)
}
