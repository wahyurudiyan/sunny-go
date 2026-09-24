package proto_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

var _ = Describe("GenerateStub", func() {
	var dir string

	BeforeEach(func() {
		var err error
		dir, err = os.MkdirTemp("", "sgo-proto-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })
	})

	Context("when the proto file doesn't exist yet", func() {
		It("writes a starter proto with the service and CRUD messages", func() {
			Expect(proto.GenerateStub(dir, "user", "demo")).To(Succeed())

			content, err := os.ReadFile(filepath.Join(dir, "user.proto"))
			Expect(err).NotTo(HaveOccurred())

			Expect(string(content)).To(ContainSubstring("service UserService"))
			Expect(string(content)).To(ContainSubstring("message User {"))
			Expect(string(content)).To(ContainSubstring(`go_package = "demo/contract/gen/user"`))
		})
	})

	Context("when the proto file already exists", func() {
		It("refuses to overwrite it", func() {
			Expect(proto.GenerateStub(dir, "user", "demo")).To(Succeed())

			err := proto.GenerateStub(dir, "user", "demo")

			Expect(err).To(MatchError(ContainSubstring("already exists")))
		})
	})
})

var _ = Describe("Compile and Build", func() {
	var dir string

	BeforeEach(func() {
		var err error
		dir, err = os.MkdirTemp("", "sgo-proto-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })

		Expect(proto.GenerateStub(dir, "user", "demo")).To(Succeed())
	})

	It("compiles the generated stub without error", func() {
		_, err := proto.Compile(dir, "user.proto")
		Expect(err).NotTo(HaveOccurred())
	})

	It("builds an IR whose User message matches the proto fields", func() {
		fd, err := proto.Compile(dir, "user.proto")
		Expect(err).NotTo(HaveOccurred())

		file, err := proto.Build(fd)
		Expect(err).NotTo(HaveOccurred())

		Expect(file.Package).To(Equal("user.v1"))

		msg := file.FindMessage("User")
		Expect(msg).NotTo(BeNil())
		Expect(msg.Fields).To(ConsistOf(
			proto.Field{Name: "id", GoName: "Id", Kind: proto.KindString},
			proto.Field{Name: "name", GoName: "Name", Kind: proto.KindString},
			proto.Field{Name: "description", GoName: "Description", Kind: proto.KindString},
		))
	})

	It("builds an IR whose service methods match the proto RPCs", func() {
		fd, err := proto.Compile(dir, "user.proto")
		Expect(err).NotTo(HaveOccurred())

		file, err := proto.Build(fd)
		Expect(err).NotTo(HaveOccurred())

		Expect(file.Services).To(HaveLen(1))
		svc := file.Services[0]
		Expect(svc.Name).To(Equal("UserService"))
		Expect(svc.Methods).To(ContainElement(proto.Method{
			Name: "GetUser", Input: "GetUserRequest", Output: "UserResponse",
		}))
		Expect(svc.Methods).To(HaveLen(5))
	})

	It("builds a repeated message field correctly", func() {
		fd, err := proto.Compile(dir, "user.proto")
		Expect(err).NotTo(HaveOccurred())

		file, err := proto.Build(fd)
		Expect(err).NotTo(HaveOccurred())

		msg := file.FindMessage("ListUsersResponse")
		Expect(msg).NotTo(BeNil())

		var usersField *proto.Field
		for i := range msg.Fields {
			if msg.Fields[i].Name == "users" {
				usersField = &msg.Fields[i]
			}
		}
		Expect(usersField).NotTo(BeNil())
		Expect(usersField.Kind).To(Equal(proto.KindMessage))
		Expect(usersField.Repeated).To(BeTrue())
		Expect(usersField.MessageType).To(Equal("User"))
		Expect(usersField.GoType()).To(Equal("[]*User"))
	})

	It("returns nil for a message that doesn't exist", func() {
		fd, err := proto.Compile(dir, "user.proto")
		Expect(err).NotTo(HaveOccurred())

		file, err := proto.Build(fd)
		Expect(err).NotTo(HaveOccurred())

		Expect(file.FindMessage("NoSuchMessage")).To(BeNil())
	})

	It("returns a clear error for invalid proto syntax", func() {
		Expect(os.WriteFile(filepath.Join(dir, "broken.proto"), []byte("not valid proto {{{"), 0644)).To(Succeed())

		_, err := proto.Compile(dir, "broken.proto")

		Expect(err).To(MatchError(ContainSubstring("failed to compile")))
	})
})

// Exercises every scalar Kind (plus enum and a nested message), since the
// CRUD starter template used by the specs above only ever declares
// string/bool/int32-ish fields — mapKind and Kind.GoType() otherwise go
// almost entirely untested for int64/uint32/uint64/float/double/bytes/
// enum. Also the concrete answer to ARCHITECTURE.md's "Enum fields" open
// question: an enum field compiles and maps to int32 like any other
// integral kind, it's just never been exercised by a generated template.
var _ = Describe("field kind mapping", func() {
	var dir string

	BeforeEach(func() {
		var err error
		dir, err = os.MkdirTemp("", "sgo-proto-kinds-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })

		src := `syntax = "proto3";

package kinds.v1;

option go_package = "demo/contract/gen/kinds";

enum Status {
  STATUS_UNKNOWN = 0;
  STATUS_ACTIVE = 1;
}

message Nested {
  string value = 1;
}

message AllKinds {
  string s = 1;
  bool b = 2;
  int32 i32 = 3;
  int64 i64 = 4;
  uint32 u32 = 5;
  uint64 u64 = 6;
  float f = 7;
  double d = 8;
  bytes by = 9;
  Status status = 10;
  Nested nested = 11;
}
`
		Expect(os.WriteFile(filepath.Join(dir, "kinds.proto"), []byte(src), 0644)).To(Succeed())
	})

	It("maps every scalar kind, enum, and message field to its Go type", func() {
		fd, err := proto.Compile(dir, "kinds.proto")
		Expect(err).NotTo(HaveOccurred())

		file, err := proto.Build(fd)
		Expect(err).NotTo(HaveOccurred())

		msg := file.FindMessage("AllKinds")
		Expect(msg).NotTo(BeNil())

		byName := map[string]proto.Field{}
		for _, f := range msg.Fields {
			byName[f.Name] = f
		}

		Expect(byName["s"].GoType()).To(Equal("string"))
		Expect(byName["b"].GoType()).To(Equal("bool"))
		Expect(byName["i32"].GoType()).To(Equal("int32"))
		Expect(byName["i64"].GoType()).To(Equal("int64"))
		Expect(byName["u32"].GoType()).To(Equal("uint32"))
		Expect(byName["u64"].GoType()).To(Equal("uint64"))
		Expect(byName["f"].GoType()).To(Equal("float32"))
		Expect(byName["d"].GoType()).To(Equal("float64"))
		Expect(byName["by"].GoType()).To(Equal("[]byte"))
		Expect(byName["status"].Kind).To(Equal(proto.KindEnum))
		Expect(byName["status"].GoType()).To(Equal("int32"))
		Expect(byName["status"].IsMessage()).To(BeFalse())

		nested := byName["nested"]
		Expect(nested.IsMessage()).To(BeTrue())
		Expect(nested.MessageType).To(Equal("Nested"))
		Expect(nested.GoType()).To(Equal("*Nested"))
	})
})

// Exercises the sensitive-field options (ARCHITECTURE.md §22) against a
// real compiled proto — in particular that HasJSONName/JSONName really
// does distinguish an explicit `[json_name = "..."]` override from
// protobuf's own computed default, which is what EffectiveJSONName's
// backward-compatible fallback depends on.
var _ = Describe("sensitive-field options", func() {
	var dir string

	BeforeEach(func() {
		var err error
		dir, err = os.MkdirTemp("", "sgo-proto-sensitive-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })

		src := `syntax = "proto3";

package sensitive.v1;

import "sgo/options.proto";

option go_package = "demo/contract/gen/sensitive";

message Account {
  string id = 1;
  string id_number = 2 [(sgo.obfuscate_visible) = 3, (sgo.pii) = true];
  string email = 3 [(sgo.pii) = true];
  string display_name = 4 [json_name = "fullName"];
  int32 balance = 5;
}
`
		Expect(os.WriteFile(filepath.Join(dir, "sensitive.proto"), []byte(src), 0644)).To(Succeed())
	})

	It("leaves an unmarked field's JSONName empty, falling back to Name", func() {
		fd, err := proto.Compile(dir, "sensitive.proto")
		Expect(err).NotTo(HaveOccurred())
		file, err := proto.Build(fd)
		Expect(err).NotTo(HaveOccurred())

		msg := file.FindMessage("Account")
		Expect(msg).NotTo(BeNil())

		var id proto.Field
		for _, f := range msg.Fields {
			if f.Name == "id" {
				id = f
			}
		}
		Expect(id.JSONName).To(Equal(""))
		Expect(id.EffectiveJSONName()).To(Equal("id"))
		Expect(id.IsSensitive()).To(BeFalse())
	})

	It("picks up an explicit [json_name=...] override, distinct from the computed default", func() {
		fd, err := proto.Compile(dir, "sensitive.proto")
		Expect(err).NotTo(HaveOccurred())
		file, err := proto.Build(fd)
		Expect(err).NotTo(HaveOccurred())

		msg := file.FindMessage("Account")
		var displayName proto.Field
		for _, f := range msg.Fields {
			if f.Name == "display_name" {
				displayName = f
			}
		}
		Expect(displayName.JSONName).To(Equal("fullName"))
		Expect(displayName.EffectiveJSONName()).To(Equal("fullName"))
	})

	It("treats an explicit override that happens to match the computed default the same as unset (documented edge case, not a bug)", func() {
		src := `syntax = "proto3";

package sensitive.v1;

option go_package = "demo/contract/gen/sensitive";

message Coincidence {
  string first_name = 1 [json_name = "firstName"];
}
`
		Expect(os.WriteFile(filepath.Join(dir, "coincidence.proto"), []byte(src), 0644)).To(Succeed())

		fd, err := proto.Compile(dir, "coincidence.proto")
		Expect(err).NotTo(HaveOccurred())
		file, err := proto.Build(fd)
		Expect(err).NotTo(HaveOccurred())

		msg := file.FindMessage("Coincidence")
		// A real, narrow limitation, not silently glossed over: since
		// this can't be distinguished from "unset" at this layer,
		// EffectiveJSONName falls back to Name ("first_name", sgo's own
		// long-standing default) rather than honoring "firstName" here —
		// the one case where an explicit override is not honored is
		// exactly when it equals protobuf's own standard camelCase
		// default. Documented in ARCHITECTURE.md §22.
		Expect(msg.Fields[0].JSONName).To(Equal(""))
		Expect(msg.Fields[0].EffectiveJSONName()).To(Equal("first_name"))
	})

	It("reads obfuscate_visible and pii together on the same field", func() {
		fd, err := proto.Compile(dir, "sensitive.proto")
		Expect(err).NotTo(HaveOccurred())
		file, err := proto.Build(fd)
		Expect(err).NotTo(HaveOccurred())

		msg := file.FindMessage("Account")
		var idNumber proto.Field
		for _, f := range msg.Fields {
			if f.Name == "id_number" {
				idNumber = f
			}
		}
		Expect(idNumber.HasObfuscateVisible).To(BeTrue())
		Expect(idNumber.ObfuscateVisible).To(Equal(int32(3)))
		Expect(idNumber.PII).To(BeTrue())
		Expect(idNumber.IsSensitive()).To(BeTrue())
	})

	It("reads a plain pii marker with no obfuscation", func() {
		fd, err := proto.Compile(dir, "sensitive.proto")
		Expect(err).NotTo(HaveOccurred())
		file, err := proto.Build(fd)
		Expect(err).NotTo(HaveOccurred())

		msg := file.FindMessage("Account")
		var email proto.Field
		for _, f := range msg.Fields {
			if f.Name == "email" {
				email = f
			}
		}
		Expect(email.PII).To(BeTrue())
		Expect(email.HasObfuscateVisible).To(BeFalse())
		Expect(email.IsSensitive()).To(BeTrue())
	})

	It("leaves an entirely unmarked field with no sensitivity at all", func() {
		fd, err := proto.Compile(dir, "sensitive.proto")
		Expect(err).NotTo(HaveOccurred())
		file, err := proto.Build(fd)
		Expect(err).NotTo(HaveOccurred())

		msg := file.FindMessage("Account")
		var balance proto.Field
		for _, f := range msg.Fields {
			if f.Name == "balance" {
				balance = f
			}
		}
		Expect(balance.IsSensitive()).To(BeFalse())
	})

	It("rejects (sgo.obfuscate_visible) on a non-string field with a clear error", func() {
		src := `syntax = "proto3";

package sensitive.v1;

import "sgo/options.proto";

option go_package = "demo/contract/gen/sensitive";

message Bad {
  int32 amount = 1 [(sgo.obfuscate_visible) = 2];
}
`
		Expect(os.WriteFile(filepath.Join(dir, "bad.proto"), []byte(src), 0644)).To(Succeed())

		fd, err := proto.Compile(dir, "bad.proto")
		Expect(err).NotTo(HaveOccurred())

		_, err = proto.Build(fd)
		Expect(err).To(MatchError(ContainSubstring("only valid on a string field")))
	})

	It("rejects (sgo.obfuscate_visible) on a repeated string field", func() {
		src := `syntax = "proto3";

package sensitive.v1;

import "sgo/options.proto";

option go_package = "demo/contract/gen/sensitive";

message Bad {
  repeated string tags = 1 [(sgo.obfuscate_visible) = 2];
}
`
		Expect(os.WriteFile(filepath.Join(dir, "bad2.proto"), []byte(src), 0644)).To(Succeed())

		fd, err := proto.Compile(dir, "bad2.proto")
		Expect(err).NotTo(HaveOccurred())

		_, err = proto.Build(fd)
		Expect(err).To(MatchError(ContainSubstring("only valid on a string field")))
	})

	It("rejects a negative (sgo.obfuscate_visible) with a clear error", func() {
		src := `syntax = "proto3";

package sensitive.v1;

import "sgo/options.proto";

option go_package = "demo/contract/gen/sensitive";

message Bad {
  string secret = 1 [(sgo.obfuscate_visible) = -1];
}
`
		Expect(os.WriteFile(filepath.Join(dir, "bad3.proto"), []byte(src), 0644)).To(Succeed())

		fd, err := proto.Compile(dir, "bad3.proto")
		Expect(err).NotTo(HaveOccurred())

		_, err = proto.Build(fd)
		Expect(err).To(MatchError(ContainSubstring("must be >= 0")))
	})
})
