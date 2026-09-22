package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"

	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

// GenerateEventPublisher writes internal/application/ports/event_publisher.go
// (ARCHITECTURE.md §17): the EventPublisher interface every entity's
// application service depends on, and NoopEventPublisher, its one real
// default implementation. Shared across entities (like
// internal/domain/event), always overwritten — fixed content, no
// hand-written part.
func GenerateEventPublisher(p Paths, destDir string) error {
	data := struct {
		EventImportPath  string
		AggregateExample string
	}{
		EventImportPath:  p.EventImportPath(),
		AggregateExample: entityTitle(p.Entity),
	}
	path := filepath.Join(destDir, "event_publisher.go")
	return writeGoFile("templates/event_publisher_gen.go.tmpl", data, path)
}

// appMethod is one application service method, with its parameter and
// return types already resolved to concrete Go type strings so the
// templates don't need conditional package-qualification logic.
type appMethod struct {
	Name       string
	ParamType  string // same-package Command/Query DTO, e.g. "CreateOrderRequest"
	ReturnType string // domain-qualified, e.g. "*domain.Order", "[]*domain.Order", or "" for error-only (Delete)
}

// GenerateApplicationService writes
// internal/application/<entity>/service.go (ARCHITECTURE.md §17):
// replaces internal/core/service/<entity>_service.go's role. Same
// owned-file, stub-appended-per-new-RPC contract as the pre-Phase-12
// service generator (Decision #12) — reuses ensureGoMethods
// (astmethods.go), the exact same AST mechanism, not a reimplementation
// of it — but now depends on the aggregate repository and
// EventPublisher, and each method's request parameter is a CQRS
// Command/Query DTO rather than an opaque wire-derived Request type.
//
// Method return shape follows the same naming-convention families HTTP
// route derivation already uses (§8.1): a `List*` RPC returns a slice,
// a `Delete*` RPC returns only an error, everything else returns a
// single aggregate pointer.
func GenerateApplicationService(f *sgoproto.File, fd protoreflect.FileDescriptor, p Paths, destDir string) error {
	svc, ok := primaryService(f)
	if !ok {
		return fmt.Errorf("%s declares no service; sgo needs exactly one service per proto file", f.Path)
	}

	aggregateMD, err := AggregateRoot(fd, p.Entity)
	if err != nil {
		return err
	}
	aggregateName := string(aggregateMD.Name())

	methods := make([]appMethod, len(svc.Methods))
	for i, m := range svc.Methods {
		methods[i] = appMethod{
			Name:       m.Name,
			ParamType:  m.Input,
			ReturnType: returnType(m.Name, aggregateName),
		}
	}

	receiver := entityTitle(p.Entity) + "Service"
	path := filepath.Join(destDir, "service.go")

	_, err = os.Stat(path)
	switch {
	case os.IsNotExist(err):
		return writeApplicationServiceSkeleton(path, p, receiver, methods)
	case err != nil:
		return fmt.Errorf("failed to check %s: %w", path, err)
	default:
		return ensureApplicationServiceMethods(path, p, receiver, methods)
	}
}

func returnType(rpcName, aggregateName string) string {
	switch {
	case strings.HasPrefix(rpcName, "List"):
		return "[]*domain." + aggregateName
	case strings.HasPrefix(rpcName, "Delete"):
		return ""
	default:
		return "*domain." + aggregateName
	}
}

func writeApplicationServiceSkeleton(path string, p Paths, receiver string, methods []appMethod) error {
	data := struct {
		Package          string
		Entity           string
		DomainImportPath string
		PortsImportPath  string
		Methods          []appMethod
	}{
		Package:          p.Entity,
		Entity:           entityTitle(p.Entity),
		DomainImportPath: p.AggregateDomainImportPath(),
		PortsImportPath:  p.ApplicationPortsImportPath(),
		Methods:          methods,
	}
	return writeGoFile("templates/application_service_owned.go.tmpl", data, path)
}

func ensureApplicationServiceMethods(path string, p Paths, receiver string, methods []appMethod) error {
	byName := make(map[string]appMethod, len(methods))
	names := make([]string, len(methods))
	for i, m := range methods {
		byName[m.Name] = m
		names[i] = m.Name
	}

	return ensureGoMethods(path, receiver, names, func(name string) ([]byte, error) {
		m := byName[name]
		return renderTemplate("templates/application_service_method_stub.go.tmpl", struct {
			Receiver   string
			Name       string
			ParamType  string
			ReturnType string
		}{
			Receiver:   receiver,
			Name:       m.Name,
			ParamType:  m.ParamType,
			ReturnType: m.ReturnType,
		})
	})
}
