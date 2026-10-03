package core

import (
	"fmt"
	"os"
	"path/filepath"

	"google.golang.org/protobuf/reflect/protoreflect"

	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

// GenerateAggregate writes internal/domain/<entity>/<entity>_gen.go —
// the Aggregate Root, its Value Objects, Domain Events, and any plain
// child entities (ARCHITECTURE.md §17) — always overwritten, and, if it
// doesn't already exist, aggregate.go (owned, for hand-written business
// methods and event recording).
//
// fd is f's underlying descriptor (role inference — AggregateRoot,
// ValueObjects, DomainEvents — reads the sgo/options.proto markings off
// it; f's IR has already dropped that information by the time it's
// built).
func GenerateAggregate(f *sgoproto.File, fd protoreflect.FileDescriptor, p Paths, destDir string) error {
	aggregateMD, err := AggregateRoot(fd, p.Entity)
	if err != nil {
		return err
	}
	aggregateName := string(aggregateMD.Name())

	aggregate := f.FindMessage(aggregateName)
	if aggregate == nil {
		return fmt.Errorf("%s: aggregate root %s has no corresponding IR message (compiler/IR mismatch)", f.Path, aggregateName)
	}

	voNames := messageNames(ValueObjects(fd))
	eventNames := messageNames(DomainEvents(fd))

	var valueObjects, domainEvents, childEntities []sgoproto.Message
	for _, m := range f.Messages {
		switch {
		case m.Name == aggregateName:
			continue
		case IsDTO(m.Name):
			continue // application-layer Command/Query DTO, not domain
		case contains(voNames, m.Name):
			valueObjects = append(valueObjects, m)
		case contains(eventNames, m.Name):
			domainEvents = append(domainEvents, m)
		default:
			childEntities = append(childEntities, m)
		}
	}

	needsMasking := aggregate.HasObfuscatedFields()
	for _, m := range valueObjects {
		needsMasking = needsMasking || m.HasObfuscatedFields()
	}
	for _, m := range childEntities {
		needsMasking = needsMasking || m.HasObfuscatedFields()
	}

	genData := struct {
		Package         string
		AggregateName   string
		Aggregate       sgoproto.Message
		ValueObjects    []sgoproto.Message
		DomainEvents    []sgoproto.Message
		ChildEntities   []sgoproto.Message
		EventImportPath string
		MaskImportPath  string
		NeedsMasking    bool
	}{
		Package:         p.Entity,
		AggregateName:   aggregateName,
		Aggregate:       *aggregate,
		ValueObjects:    valueObjects,
		DomainEvents:    domainEvents,
		ChildEntities:   childEntities,
		EventImportPath: p.EventImportPath(),
		MaskImportPath:  p.MaskImportPath(),
		NeedsMasking:    needsMasking,
	}

	genPath := filepath.Join(destDir, p.Entity+"_gen.go")
	if err := writeGoFile("templates/aggregate_gen.go.tmpl", genData, genPath); err != nil {
		return err
	}

	ownedPath := filepath.Join(destDir, "aggregate.go")
	if _, err := os.Stat(ownedPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check %s: %w", ownedPath, err)
	}

	ownedData := struct {
		Package       string
		AggregateName string
	}{
		Package:       p.Entity,
		AggregateName: aggregateName,
	}
	return writeGoFile("templates/aggregate_owned.go.tmpl", ownedData, ownedPath)
}

// GenerateAggregateRepositoryPort writes
// internal/domain/<entity>/repository.go (ARCHITECTURE.md §21): the
// fixed CRUD shape, in the domain package itself and typed directly
// against the aggregate root — no separate port/out package or domain
// import needed now that it's co-located with what it's a port for.
// Named distinctly from the pre-Phase-12 GenerateRepositoryPort
// (port.go) while both exist side by side; GenerateCode switches over
// to this one and the old generators are removed together (§17/Phase
// 12's project-wide rewiring).
//
// Also adds one interface method per RPC marked
// `option (sgo.repository_query) = true;`, beyond the fixed CRUD shape
// (§21/Phase 16).
func GenerateAggregateRepositoryPort(f *sgoproto.File, fd protoreflect.FileDescriptor, p Paths, destDir string) error {
	aggregateMD, err := AggregateRoot(fd, p.Entity)
	if err != nil {
		return err
	}

	queryMethods, err := RepositoryQueryMethods(fd, f, p.Entity)
	if err != nil {
		return err
	}

	data := struct {
		Package       string
		AggregateName string
		QueryMethods  []RepositoryQueryMethod
	}{
		Package:       p.Entity,
		AggregateName: string(aggregateMD.Name()),
		QueryMethods:  queryMethods,
	}

	path := filepath.Join(destDir, "repository.go")
	return writeGoFile("templates/domain_repository_gen.go.tmpl", data, path)
}

// GenerateDomainErrors writes internal/domain/<entity>/errors.go
// (ARCHITECTURE.md §17): sentinel errors a repository implementation
// returns.
func GenerateDomainErrors(fd protoreflect.FileDescriptor, p Paths, destDir string) error {
	aggregateMD, err := AggregateRoot(fd, p.Entity)
	if err != nil {
		return err
	}

	data := struct {
		Package       string
		AggregateName string
	}{
		Package:       p.Entity,
		AggregateName: string(aggregateMD.Name()),
	}

	path := filepath.Join(destDir, "errors.go")
	return writeGoFile("templates/domain_errors_gen.go.tmpl", data, path)
}

// GenerateEventKernel writes internal/domain/event/event.go
// (ARCHITECTURE.md §17): the DomainEvent interface every entity's
// generated event structs implement, shared across entities (not
// per-entity) so a single EventPublisher can accept events from any of
// them. Fixed content, no hand-written part — always overwritten, once
// per project rather than once per entity (safe to call once per
// `sgo generate code` run; idempotent, since it's the same content
// every time).
func GenerateEventKernel(destDir string) error {
	path := filepath.Join(destDir, "event.go")
	return writeGoFile("templates/domain_event_kernel_gen.go.tmpl", struct{}{}, path)
}

// GenerateMaskKernel writes internal/domain/mask/mask.go
// (ARCHITECTURE.md §22): the Obfuscate helper every entity's
// masking-aware MarshalJSON/LogValue, and the infra mapper's gRPC
// masking, call into. Fixed content, no hand-written part — always
// overwritten, once per project rather than once per entity, the same
// as GenerateEventKernel.
func GenerateMaskKernel(destDir string) error {
	path := filepath.Join(destDir, "mask.go")
	return writeGoFile("templates/mask_gen.go.tmpl", struct{}{}, path)
}

func messageNames(mds []protoreflect.MessageDescriptor) []string {
	names := make([]string, len(mds))
	for i, md := range mds {
		names[i] = string(md.Name())
	}
	return names
}

func contains(names []string, name string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}
