package proto

import (
	"fmt"
	"strings"

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
		}

		kind, err := mapKind(fd)
		if err != nil {
			return Message{}, fmt.Errorf("message %s: %w", md.Name(), err)
		}
		field.Kind = kind

		if kind == KindMessage {
			field.MessageType = string(fd.Message().Name())
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
