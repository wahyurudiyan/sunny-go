package proto

import (
	validatepb "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	sgopb "github.com/wahyurudiyan/sunny-go/pkg/sgoproto"
)

// messageOptions returns md's compiled MessageOptions as the real
// static descriptorpb type, or nil if it has none. protocompile's
// linked descriptors carry options as a *dynamicpb.Message (built from
// the descriptor it just parsed, not from any generated Go package's
// static type), so reading a custom extension registered against the
// static type — sgopb.E_AggregateRoot, validatepb.E_Field, etc. — needs
// a marshal/unmarshal round-trip through the real descriptorpb type
// first. proto.GetExtension on the dynamic message directly panics
// (confirmed directly, not assumed) because the two types' extension
// registrations don't line up.
func messageOptions(md protoreflect.MessageDescriptor) *descriptorpb.MessageOptions {
	return roundTrip(md.Options(), &descriptorpb.MessageOptions{})
}

func methodOptions(mtd protoreflect.MethodDescriptor) *descriptorpb.MethodOptions {
	return roundTrip(mtd.Options(), &descriptorpb.MethodOptions{})
}

func fieldOptions(fd protoreflect.FieldDescriptor) *descriptorpb.FieldOptions {
	return roundTrip(fd.Options(), &descriptorpb.FieldOptions{})
}

func roundTrip[T proto.Message](dyn protoreflect.ProtoMessage, static T) T {
	msg, ok := dyn.(proto.Message)
	if !ok {
		return static
	}
	raw, err := proto.Marshal(msg)
	if err != nil {
		return static
	}
	_ = proto.Unmarshal(raw, static)
	return static
}

// IsAggregateRoot reports whether md is explicitly marked
// `option (sgo.aggregate_root) = true;`.
func IsAggregateRoot(md protoreflect.MessageDescriptor) bool {
	return proto.GetExtension(messageOptions(md), sgopb.E_AggregateRoot).(bool)
}

// IsValueObject reports whether md is explicitly marked
// `option (sgo.value_object) = true;`.
func IsValueObject(md protoreflect.MessageDescriptor) bool {
	return proto.GetExtension(messageOptions(md), sgopb.E_ValueObject).(bool)
}

// IsDomainEvent reports whether md is explicitly marked
// `option (sgo.domain_event) = true;`.
func IsDomainEvent(md protoreflect.MessageDescriptor) bool {
	return proto.GetExtension(messageOptions(md), sgopb.E_DomainEvent).(bool)
}

// IsCommand reports whether mtd is explicitly marked
// `option (sgo.command) = true;`.
func IsCommand(mtd protoreflect.MethodDescriptor) bool {
	return proto.GetExtension(methodOptions(mtd), sgopb.E_Command).(bool)
}

// IsQuery reports whether mtd is explicitly marked
// `option (sgo.query) = true;`.
func IsQuery(mtd protoreflect.MethodDescriptor) bool {
	return proto.GetExtension(methodOptions(mtd), sgopb.E_Query).(bool)
}

// FieldConstraints returns fd's protovalidate constraints
// (`(buf.validate.field) = {...}`), or nil if it has none.
func FieldConstraints(fd protoreflect.FieldDescriptor) *validatepb.FieldRules {
	opts := fieldOptions(fd)
	if !proto.HasExtension(opts, validatepb.E_Field) {
		return nil
	}
	rules, _ := proto.GetExtension(opts, validatepb.E_Field).(*validatepb.FieldRules)
	return rules
}
