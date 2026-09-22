package core

import (
	"path/filepath"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"

	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

// IsCommandRPC classifies mtd as a CQRS command (state-changing) by the
// same naming convention HTTP route derivation already uses
// (Create/Update/Delete, §8.1), overridable with an explicit
// `option (sgo.command) = true;`/`option (sgo.query) = true;`
// (ARCHITECTURE.md §17).
func IsCommandRPC(mtd protoreflect.MethodDescriptor) bool {
	if sgoproto.IsCommand(mtd) {
		return true
	}
	if sgoproto.IsQuery(mtd) {
		return false
	}
	name := string(mtd.Name())
	return strings.HasPrefix(name, "Create") || strings.HasPrefix(name, "Update") || strings.HasPrefix(name, "Delete")
}

// GenerateCommandsAndQueries writes
// internal/application/<entity>/{command,query}.go (ARCHITECTURE.md
// §17): every RPC's request message, classified as a command or query
// by IsCommandRPC. Always overwritten — mirrors the request shape the
// proto declares, same generated/owned split principle as everything
// else derived directly from the proto (Decision #3).
//
// v1 limitation, stated rather than silently mishandled: a request
// message with a message-typed field (not just scalars) generates an
// unqualified reference to that type, which only resolves if it's also
// declared as a domain type in the *same* entity's package — sgo's own
// `generate proto` starter template never produces this shape, so it's
// only reachable by hand-editing a proto to add one. Revisit if a real
// use case needs it.
func GenerateCommandsAndQueries(f *sgoproto.File, fd protoreflect.FileDescriptor, p Paths, destDir string) error {
	svc, ok := primaryService(f)
	if !ok {
		return nil
	}

	methods := fd.Services().Get(0).Methods()
	methodByName := make(map[string]protoreflect.MethodDescriptor, methods.Len())
	for i := 0; i < methods.Len(); i++ {
		methodByName[string(methods.Get(i).Name())] = methods.Get(i)
	}

	seen := map[string]bool{}
	var commands, queries []sgoproto.Message
	for _, m := range svc.Methods {
		if seen[m.Input] {
			continue // two RPCs sharing one request message (e.g. both List and a filtered variant) only need it generated once
		}
		seen[m.Input] = true

		msg := f.FindMessage(m.Input)
		if msg == nil {
			continue
		}

		mtd, ok := methodByName[m.Name]
		if !ok {
			continue
		}

		if IsCommandRPC(mtd) {
			commands = append(commands, *msg)
		} else {
			queries = append(queries, *msg)
		}
	}

	if len(commands) > 0 {
		if err := writeGoFile("templates/cqrs_gen.go.tmpl", struct {
			Package  string
			Messages []sgoproto.Message
		}{Package: p.Entity, Messages: commands}, filepath.Join(destDir, "command_gen.go")); err != nil {
			return err
		}
	}

	if len(queries) > 0 {
		if err := writeGoFile("templates/cqrs_gen.go.tmpl", struct {
			Package  string
			Messages []sgoproto.Message
		}{Package: p.Entity, Messages: queries}, filepath.Join(destDir, "query_gen.go")); err != nil {
			return err
		}
	}

	return nil
}
