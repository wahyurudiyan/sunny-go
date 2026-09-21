package core

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"

	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

// IsDTO reports whether name is a wire-level Request/Response DTO by
// sgo's own naming convention (the same one `sgo generate proto`'s
// starter template already commits to: CreateUserRequest, UserResponse,
// ...) — these belong to the application layer's Command/Query DTOs
// (ARCHITECTURE.md §17), never the domain model itself.
func IsDTO(name string) bool {
	return strings.HasSuffix(name, "Request") || strings.HasSuffix(name, "Response")
}

// AggregateRoot returns f's Aggregate Root: the message explicitly
// marked `option (sgo.aggregate_root) = true;`, or — if none is — the
// message matching the entity name, the same fallback convention HTTP
// route derivation already uses (§8.1). Errors if neither is found, or
// if more than one message is explicitly marked (ambiguous — sgo
// generates one aggregate per proto file, matching one service per
// file, Decision context in §4).
func AggregateRoot(fd protoreflect.FileDescriptor, entity string) (protoreflect.MessageDescriptor, error) {
	msgs := fd.Messages()

	var marked []protoreflect.MessageDescriptor
	for i := 0; i < msgs.Len(); i++ {
		md := msgs.Get(i)
		if sgoproto.IsAggregateRoot(md) {
			marked = append(marked, md)
		}
	}

	switch len(marked) {
	case 1:
		return marked[0], nil
	case 0:
		fallback := msgs.ByName(protoreflect.Name(entityTitle(entity)))
		if fallback == nil {
			return nil, fmt.Errorf(
				"%s: no aggregate root found — mark one message with `option (sgo.aggregate_root) = true;`, or name it %s to match the entity",
				fd.Path(), entityTitle(entity))
		}
		return fallback, nil
	default:
		names := make([]string, len(marked))
		for i, md := range marked {
			names[i] = string(md.Name())
		}
		return nil, fmt.Errorf(
			"%s: more than one message marked `option (sgo.aggregate_root) = true;` (%s) — sgo generates exactly one aggregate per proto file",
			fd.Path(), strings.Join(names, ", "))
	}
}

// ValueObjects returns every message in fd marked
// `option (sgo.value_object) = true;`.
func ValueObjects(fd protoreflect.FileDescriptor) []protoreflect.MessageDescriptor {
	return markedMessages(fd, sgoproto.IsValueObject)
}

// DomainEvents returns every message in fd marked
// `option (sgo.domain_event) = true;`.
func DomainEvents(fd protoreflect.FileDescriptor) []protoreflect.MessageDescriptor {
	return markedMessages(fd, sgoproto.IsDomainEvent)
}

func markedMessages(fd protoreflect.FileDescriptor, marked func(protoreflect.MessageDescriptor) bool) []protoreflect.MessageDescriptor {
	msgs := fd.Messages()
	var out []protoreflect.MessageDescriptor
	for i := 0; i < msgs.Len(); i++ {
		if md := msgs.Get(i); marked(md) {
			out = append(out, md)
		}
	}
	return out
}
