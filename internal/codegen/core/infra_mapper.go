package core

import (
	"path/filepath"

	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

// GenerateInfraMapper writes
// internal/infrastructure/transport/mapper_gen.go (ARCHITECTURE.md
// §17): ToDomain/FromDomain conversions between contract/gen's wire
// types and internal/domain's aggregate/value-object/entity structs,
// for every domain message in f (the Aggregate Root, its Value
// Objects, Domain Events, and any child entities) — everything
// GenerateAggregate also generates a domain struct for. Excludes
// Request/Response DTOs, which map to the application layer's
// Command/Query types instead (a separate, not-yet-built mapper —
// PLAN.md).
//
// Replaces internal/adapter/mapper/<entity>_mapper_gen.go's role, with
// one behavioral addition: ToDomain now validates the incoming wire
// message (protovalidate) and returns an error, rather than
// unconditionally converting it.
func GenerateInfraMapper(f *sgoproto.File, p Paths, destDir string) error {
	var messages []sgoproto.Message
	for _, m := range f.Messages {
		if IsDTO(m.Name) {
			continue
		}
		messages = append(messages, m)
	}

	data := struct {
		WireImportPath   string
		DomainImportPath string
		Messages         []sgoproto.Message
	}{
		WireImportPath:   p.WireImportPath(),
		DomainImportPath: p.AggregateDomainImportPath(),
		Messages:         messages,
	}

	path := filepath.Join(destDir, "mapper_gen.go")
	return writeGoFile("templates/infra_mapper_gen.go.tmpl", data, path)
}
