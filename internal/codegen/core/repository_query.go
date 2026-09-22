package core

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"

	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

// RepositoryQueryParam is one flattened parameter of a custom
// repository-port method (ARCHITECTURE.md §21) — one of the request
// message's own scalar fields.
type RepositoryQueryParam struct {
	GoParam string         // Go parameter name, e.g. "email"
	GoType  string         // Go parameter type, e.g. "string"
	Field   sgoproto.Field // the source field, for auto-impl matching (memgen)
}

// RepositoryQueryMethod is one custom repository-port method derived
// from an RPC marked `option (sgo.repository_query) = true;`
// (ARCHITECTURE.md §21).
type RepositoryQueryMethod struct {
	Name   string // repository-port method name, e.g. "FindByEmail"
	RPC    string // the originating RPC name, e.g. "FindUserByEmail"
	Params []RepositoryQueryParam
}

var fixedRepositoryMethods = map[string]bool{
	"Create": true, "Get": true, "List": true, "Update": true, "Delete": true,
}

// RepositoryQueryMethods returns every RPC on f's service explicitly
// marked `option (sgo.repository_query) = true;`, flattened to a
// repository-port method signature: parameters are the request
// message's own fields, in declaration order, each mapped to a scalar
// Go parameter — the same flat style this port's fixed
// Create/Get/List/Update/Delete methods already use, not the
// application layer's opaque *Request DTO style. **v1 constraint,
// stated rather than silently unsupported:** every request field must
// be scalar (no nested message, no repeated field) — one that isn't
// fails generation with a clear error naming the RPC and the offending
// field, instead of guessing how to flatten it. Also rejects a derived
// method name that collides with the port's fixed CRUD methods, which
// would otherwise fail only much later as a confusing "duplicate
// method" Go compiler error.
//
// fd is f's underlying descriptor, needed to read the
// `(sgo.repository_query)` marking off it; f's IR has already dropped
// that information by the time it's built.
func RepositoryQueryMethods(fd protoreflect.FileDescriptor, f *sgoproto.File, entity string) ([]RepositoryQueryMethod, error) {
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

	entityName := entityTitle(entity)

	var methods []RepositoryQueryMethod
	for _, m := range svc.Methods {
		mtd, ok := methodByName[m.Name]
		if !ok {
			return nil, fmt.Errorf("core: RPC %q has no matching descriptor (compiler/IR mismatch)", m.Name)
		}
		if !sgoproto.IsRepositoryQuery(mtd) {
			continue
		}

		input := f.FindMessage(m.Input)
		if input == nil {
			return nil, fmt.Errorf("core: request message %q not found for RPC %q (compiler/IR mismatch)", m.Input, m.Name)
		}

		params := make([]RepositoryQueryParam, 0, len(input.Fields))
		for _, field := range input.Fields {
			if field.IsMessage() || field.Repeated {
				return nil, fmt.Errorf(
					"repository_query on RPC %q requires every request field to be scalar; %q on %s is not (message-typed and repeated fields can't be flattened into a repository-port method)",
					m.Name, field.Name, m.Input)
			}
			params = append(params, RepositoryQueryParam{
				GoParam: lowerFirst(field.GoName),
				GoType:  field.GoType(),
				Field:   field,
			})
		}

		name := repositoryMethodName(m.Name, entityName)
		if fixedRepositoryMethods[name] {
			return nil, fmt.Errorf(
				"repository_query on RPC %q derives repository-port method name %q, which collides with the port's fixed %s method; rename the RPC or drop repository_query from it",
				m.Name, name, name)
		}

		methods = append(methods, RepositoryQueryMethod{Name: name, RPC: m.Name, Params: params})
	}

	return methods, nil
}

// repositoryMethodName derives a repository-port method name from an
// RPC name by removing every occurrence of the entity's own name from
// it (ARCHITECTURE.md §21) — e.g. "FindUserByEmail" on entity "User"
// becomes "FindByEmail". An RPC name that doesn't mention the entity at
// all keeps its own name as-is (e.g. "ArchiveAccount" on entity "User"
// with no "User" in it).
func repositoryMethodName(rpcName, entityName string) string {
	stripped := strings.ReplaceAll(rpcName, entityName, "")
	if stripped == "" {
		return rpcName
	}
	return stripped
}

func lowerFirst(name string) string {
	if name == "" {
		return name
	}
	return strings.ToLower(name[:1]) + name[1:]
}
