package proto

import (
	validatepb "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"

	"google.golang.org/genproto/googleapis/api/annotations"
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

func serviceOptions(sd protoreflect.ServiceDescriptor) *descriptorpb.ServiceOptions {
	return roundTrip(sd.Options(), &descriptorpb.ServiceOptions{})
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

// IsRepositoryQuery reports whether mtd is explicitly marked
// `option (sgo.repository_query) = true;` — it also needs a
// counterpart on the aggregate repository port (ARCHITECTURE.md §21).
func IsRepositoryQuery(mtd protoreflect.MethodDescriptor) bool {
	return proto.GetExtension(methodOptions(mtd), sgopb.E_RepositoryQuery).(bool)
}

// IsHideRoute reports whether mtd is explicitly marked
// `option (sgo.hide_route) = true;` — it should get no HTTP route
// (ARCHITECTURE.md §21), independent of repository_query.
func IsHideRoute(mtd protoreflect.MethodDescriptor) bool {
	return proto.GetExtension(methodOptions(mtd), sgopb.E_HideRoute).(bool)
}

// IsPII reports whether fd is explicitly marked
// `option (sgo.pii) = true;` — classification/documentation only,
// independent of ObfuscateVisible (ARCHITECTURE.md §22).
func IsPII(fd protoreflect.FieldDescriptor) bool {
	return proto.GetExtension(fieldOptions(fd), sgopb.E_Pii).(bool)
}

// ObfuscateVisible returns fd's `option (sgo.obfuscate_visible) = N;`
// value and whether it was set at all — proto3 gives int32 fields no
// way to distinguish an explicit 0 from "unset" on the wire, so
// presence is reported separately rather than folded into the int
// return value (ARCHITECTURE.md §22).
func ObfuscateVisible(fd protoreflect.FieldDescriptor) (visible int32, ok bool) {
	opts := fieldOptions(fd)
	if !proto.HasExtension(opts, sgopb.E_ObfuscateVisible) {
		return 0, false
	}
	return proto.GetExtension(opts, sgopb.E_ObfuscateVisible).(int32), true
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

// HTTPRule returns mtd's `(google.api.http) = {...}` annotation, or nil
// if it has none — in which case httpgen.BuildRoutes falls back to its
// existing naming-convention route derivation (ARCHITECTURE.md §20).
// Read through the real, official generated Go bindings
// (google.golang.org/genproto/googleapis/api/annotations), the same
// "read a real extension through its real generated type" approach
// FieldConstraints already uses for buf.validate.field.
func HTTPRule(mtd protoreflect.MethodDescriptor) *annotations.HttpRule {
	opts := methodOptions(mtd)
	if !proto.HasExtension(opts, annotations.E_Http) {
		return nil
	}
	rule, _ := proto.GetExtension(opts, annotations.E_Http).(*annotations.HttpRule)
	return rule
}

// BasePath returns sd's `option (sgo.base_path) = "...";` override, or
// "" if it has none — in which case the caller uses the default
// "/api/v1" prefix (ARCHITECTURE.md §8.1/§20).
func BasePath(sd protoreflect.ServiceDescriptor) string {
	return proto.GetExtension(serviceOptions(sd), sgopb.E_BasePath).(string)
}
