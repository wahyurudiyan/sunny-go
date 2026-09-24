package proto

import (
	"fmt"
	"strings"
	"unicode"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// Kind is a simplified Go-facing view of a protobuf field type, used to
// drive domain struct and mapper generation without every consumer
// needing to understand protoreflect.Kind directly.
type Kind int

const (
	KindUnknown Kind = iota
	KindString
	KindBool
	KindInt32
	KindInt64
	KindUint32
	KindUint64
	KindFloat
	KindDouble
	KindBytes
	KindMessage
	KindEnum
)

// GoType returns the Go type for a scalar Kind. For KindMessage, use
// Field.GoType() instead, since it needs the referenced message's name.
func (k Kind) GoType() string {
	switch k {
	case KindString:
		return "string"
	case KindBool:
		return "bool"
	case KindInt32:
		return "int32"
	case KindInt64:
		return "int64"
	case KindUint32:
		return "uint32"
	case KindUint64:
		return "uint64"
	case KindFloat:
		return "float32"
	case KindDouble:
		return "float64"
	case KindBytes:
		return "[]byte"
	case KindEnum:
		return "int32"
	default:
		return "any"
	}
}

// Field is one field of a Message.
type Field struct {
	Name        string // proto field name, e.g. "first_name"
	GoName      string // Go field name, e.g. "FirstName"
	Kind        Kind
	MessageType string // Go type name of the referenced message, set only when Kind == KindMessage
	Repeated    bool

	// JSONName is this field's explicit `[json_name = "..."]` override,
	// or "" if none was set — in which case EffectiveJSONName falls
	// back to Name, sgo's long-standing default (ARCHITECTURE.md §22).
	// Populated from the real protobuf descriptor field, not a custom
	// sgo extension, by comparing it against protobuf's own computed
	// default (protoreflect's HasJSONName() doesn't distinguish
	// explicit from computed with this compiler, verified directly).
	// Known, narrow limitation from that: an explicit override that
	// happens to equal the computed default is indistinguishable from
	// unset, so it falls back to Name rather than being honored.
	JSONName string

	// PII reports `(sgo.pii) = true` — classification/documentation
	// only, no masking behavior by itself (ARCHITECTURE.md §22).
	PII bool

	// ObfuscateVisible and HasObfuscateVisible carry
	// `(sgo.obfuscate_visible) = N` — the number of real leading
	// characters left visible when this field is masked, with the rest
	// replaced by a fixed-length mask. HasObfuscateVisible distinguishes
	// an explicit 0 (fully masked) from "not obfuscated at all", which
	// proto3's zero-value int32 can't do on its own (ARCHITECTURE.md
	// §22).
	ObfuscateVisible    int32
	HasObfuscateVisible bool
}

// EffectiveJSONName returns this field's JSON key: the explicit
// `[json_name = "..."]` override when one was set, otherwise the raw
// proto field name — sgo's default since before this option existed,
// kept unchanged so a field with no override renders identical output
// to every prior release (ARCHITECTURE.md §22).
func (f Field) EffectiveJSONName() string {
	if f.JSONName != "" {
		return f.JSONName
	}
	return f.Name
}

// IsSensitive reports whether this field is flagged in any way this
// package tracks — PII or ObfuscateVisible — the union a caller checks
// before deciding whether a message needs masking-aware codegen at all
// (a generated MarshalJSON/LogValue, an OpenAPI vendor extension).
func (f Field) IsSensitive() bool {
	return f.PII || f.HasObfuscateVisible
}

// IsMessage reports whether this field references another message
// (as opposed to a scalar or enum), which callers use to decide whether
// a value needs recursive mapping rather than a plain assignment.
func (f Field) IsMessage() bool {
	return f.Kind == KindMessage
}

// GoType returns this field's Go type, including the "*" for message
// fields and the "[]" wrapper for repeated fields.
func (f Field) GoType() string {
	var t string
	if f.Kind == KindMessage {
		t = "*" + f.MessageType
	} else {
		t = f.Kind.GoType()
	}
	if f.Repeated {
		return "[]" + t
	}
	return t
}

// Message is a proto message, destined to become a Go struct.
type Message struct {
	Name   string // Go/proto message name, e.g. "User"
	Fields []Field
}

// Method is one RPC on a Service.
type Method struct {
	Name   string // RPC name, e.g. "GetUser"
	Input  string // input message name
	Output string // output message name
}

// Service is a proto service.
type Service struct {
	Name    string
	Methods []Method
}

// File is the intermediate representation of a compiled .proto file: the
// subset of information the rest of internal/codegen needs, independent
// of how the descriptor was obtained.
type File struct {
	Path      string // path the file was compiled from, e.g. "user.proto"
	Package   string // proto package, e.g. "user.v1"
	GoPackage string // the go_package option value
	Messages  []Message
	Services  []Service
}

// HasObfuscatedFields reports whether any field on m carries
// `(sgo.obfuscate_visible)` — the trigger for generating a
// masking-aware MarshalJSON/LogValue for this specific message
// (ARCHITECTURE.md §22). A message with only PII-marked fields (no
// obfuscation) doesn't trigger either — that marker is
// documentation-only by design.
func (m Message) HasObfuscatedFields() bool {
	for _, f := range m.Fields {
		if f.HasObfuscateVisible {
			return true
		}
	}
	return false
}

// FindMessage returns the message named name, or nil if there isn't one.
func (f *File) FindMessage(name string) *Message {
	for i := range f.Messages {
		if f.Messages[i].Name == name {
			return &f.Messages[i]
		}
	}
	return nil
}

// Build walks a linked file descriptor into a File.
func Build(fd protoreflect.FileDescriptor) (*File, error) {
	file := &File{
		Path:      fd.Path(),
		Package:   string(fd.Package()),
		GoPackage: goPackageOption(fd),
	}

	msgs := fd.Messages()
	for i := 0; i < msgs.Len(); i++ {
		m, err := buildMessage(msgs.Get(i))
		if err != nil {
			return nil, err
		}
		file.Messages = append(file.Messages, m)
	}

	svcs := fd.Services()
	for i := 0; i < svcs.Len(); i++ {
		file.Services = append(file.Services, buildService(svcs.Get(i)))
	}

	return file, nil
}

func buildMessage(md protoreflect.MessageDescriptor) (Message, error) {
	msg := Message{Name: string(md.Name())}

	fields := md.Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)

		field := Field{
			Name:     string(fd.Name()),
			GoName:   goName(string(fd.Name())),
			Repeated: fd.Cardinality() == protoreflect.Repeated,
			PII:      IsPII(fd),
		}

		// protoreflect.FieldDescriptor.HasJSONName() is documented to
		// report an explicit override, but protocompile's linked
		// descriptors always report true (verified directly against a
		// real compiled proto, not assumed) — protoc-family compilers
		// fill in the computed default at parse time, indistinguishable
		// from an explicit one at this layer. Compare against the
		// standard default computation instead: an override that
		// happens to equal what the default would have been anyway is
		// genuinely indistinguishable from not setting it at all, and
		// produces identical output either way, so treating it as
		// "unset" here is not a real behavior difference.
		if computed := defaultJSONName(string(fd.Name())); fd.JSONName() != computed {
			field.JSONName = fd.JSONName()
		}

		if visible, ok := ObfuscateVisible(fd); ok {
			field.ObfuscateVisible = visible
			field.HasObfuscateVisible = true
		}

		kind, err := mapKind(fd)
		if err != nil {
			return Message{}, fmt.Errorf("message %s: %w", md.Name(), err)
		}
		field.Kind = kind

		if kind == KindMessage {
			field.MessageType = string(fd.Message().Name())
		}

		if field.HasObfuscateVisible {
			if field.Kind != KindString || field.Repeated {
				return Message{}, fmt.Errorf("message %s: field %s: (sgo.obfuscate_visible) is only valid on a string field, not %s",
					md.Name(), fd.Name(), fieldKindDescription(field))
			}
			if field.ObfuscateVisible < 0 {
				return Message{}, fmt.Errorf("message %s: field %s: (sgo.obfuscate_visible) must be >= 0, got %d",
					md.Name(), fd.Name(), field.ObfuscateVisible)
			}
		}

		msg.Fields = append(msg.Fields, field)
	}

	return msg, nil
}

func buildService(sd protoreflect.ServiceDescriptor) Service {
	svc := Service{Name: string(sd.Name())}

	methods := sd.Methods()
	for i := 0; i < methods.Len(); i++ {
		md := methods.Get(i)
		svc.Methods = append(svc.Methods, Method{
			Name:   string(md.Name()),
			Input:  string(md.Input().Name()),
			Output: string(md.Output().Name()),
		})
	}

	return svc
}

func mapKind(fd protoreflect.FieldDescriptor) (Kind, error) {
	switch fd.Kind() {
	case protoreflect.StringKind:
		return KindString, nil
	case protoreflect.BoolKind:
		return KindBool, nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return KindInt32, nil
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return KindInt64, nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return KindUint32, nil
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return KindUint64, nil
	case protoreflect.FloatKind:
		return KindFloat, nil
	case protoreflect.DoubleKind:
		return KindDouble, nil
	case protoreflect.BytesKind:
		return KindBytes, nil
	case protoreflect.EnumKind:
		return KindEnum, nil
	case protoreflect.MessageKind:
		return KindMessage, nil
	default:
		return KindUnknown, fmt.Errorf("field %s: unsupported kind %s", fd.Name(), fd.Kind())
	}
}

// fieldKindDescription names field's kind for an error message, calling
// out "repeated string" specifically since the underlying scalar kind
// alone (KindString) would otherwise read as a false positive.
func fieldKindDescription(field Field) string {
	if field.Repeated {
		return "a repeated field"
	}
	return "a " + field.Kind.GoType() + " field"
}

// defaultJSONName computes protobuf's own standard default JSON name
// for a field (used by every protoc-family compiler when no explicit
// `[json_name = "..."]` is set): each underscore is dropped and the
// character after it is upper-cased; every other character passes
// through unchanged. E.g. "first_name" -> "firstName",
// "id" -> "id".
func defaultJSONName(protoName string) string {
	var b strings.Builder
	capNext := false
	for _, r := range protoName {
		switch {
		case r == '_':
			capNext = true
		case capNext:
			b.WriteRune(unicode.ToUpper(r))
			capNext = false
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func goPackageOption(fd protoreflect.FileDescriptor) string {
	opts, ok := fd.Options().(*descriptorpb.FileOptions)
	if !ok {
		return ""
	}
	return opts.GetGoPackage()
}

// goName converts a proto field/identifier name (snake_case) to the Go
// field name protoc-gen-go would produce for it (PascalCase). Kept
// consistent with wiregen's real protoc-gen-go output so domain structs
// and the generated mapper agree with contract/gen's field names.
func goName(protoName string) string {
	parts := strings.Split(protoName, "_")
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]))
		if len(p) > 1 {
			b.WriteString(p[1:])
		}
	}
	return b.String()
}
