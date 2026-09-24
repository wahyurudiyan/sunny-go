package openapigen_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/openapigen"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/project"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
	"github.com/wahyurudiyan/sunny-go/internal/config"
)

func floatPtr(f float64) *float64 { return &f }

// scaffoldProject builds a real project (proto files included) and
// returns its directory plus a config whose Services lists exactly the
// stub protos just created — Build only needs contract/pb/<name>.proto
// on disk and cfg.Services naming them, the same two things a real
// `sgo generate openapi` run has.
func scaffoldProject(root string, services ...string) (string, *config.Config) {
	GinkgoHelper()

	opts := project.Options{
		Name:          "demo",
		Module:        "demo",
		HTTPFramework: config.HTTPFrameworkGin,
		Persistence:   config.Persistence{Mode: config.PersistenceModeORM},
		OpenAPI:       config.DefaultOpenAPI(),
	}
	dir := filepath.Join(root, "demo")
	Expect(project.Scaffold(dir, opts)).To(Succeed())

	protoDir := filepath.Join(dir, "contract", "pb")
	for _, name := range services {
		Expect(proto.GenerateStub(protoDir, name, "demo")).To(Succeed())
	}

	cfg, err := config.Load(dir)
	Expect(err).NotTo(HaveOccurred())
	cfg.Services = services

	return dir, cfg
}

var _ = Describe("Build", func() {
	var root string

	BeforeEach(func() {
		var err error
		root, err = os.MkdirTemp("", "sgo-openapigen-build-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })
	})

	It("derives CRUD paths and a component schema for a single service", func() {
		dir, cfg := scaffoldProject(root, "user")

		doc, err := openapigen.Build(cfg, dir)
		Expect(err).NotTo(HaveOccurred())

		Expect(doc.Info.Title).To(Equal("demo"))
		Expect(doc.Paths).To(HaveKey("/api/v1/users"))
		Expect(doc.Paths).To(HaveKey("/api/v1/users/{id}"))

		collection := doc.Paths["/api/v1/users"]
		Expect(collection.Post).NotTo(BeNil())
		Expect(collection.Post.OperationID).To(Equal("CreateUser"))
		Expect(collection.Get).NotTo(BeNil())
		Expect(collection.Get.OperationID).To(Equal("ListUsers"))

		byID := doc.Paths["/api/v1/users/{id}"]
		Expect(byID.Get.OperationID).To(Equal("GetUser"))
		Expect(byID.Put.OperationID).To(Equal("UpdateUser"))
		Expect(byID.Delete.OperationID).To(Equal("DeleteUser"))
		Expect(byID.Get.Parameters).To(ConsistOf(openapigen.Parameter{
			Name: "id", In: "path", Required: true, Schema: &openapigen.Schema{Type: "string"},
		}))

		Expect(doc.Components.Schemas).To(HaveKey("User"))
		userSchema := doc.Components.Schemas["User"]
		Expect(userSchema.Type).To(Equal("object"))
		Expect(userSchema.Properties).To(HaveKey("name"))
		Expect(userSchema.Properties["name"]).To(Equal(&openapigen.Schema{Type: "string"}))
	})

	It("aggregates every registered service into one document", func() {
		dir, cfg := scaffoldProject(root, "user", "order")

		doc, err := openapigen.Build(cfg, dir)
		Expect(err).NotTo(HaveOccurred())

		Expect(doc.Paths).To(HaveKey("/api/v1/users"))
		Expect(doc.Paths).To(HaveKey("/api/v1/orders"))
		Expect(doc.Components.Schemas).To(HaveKey("User"))
		Expect(doc.Components.Schemas).To(HaveKey("Order"))
	})

	It("marks a repeated message field as an array of $ref", func() {
		dir, cfg := scaffoldProject(root, "user")

		doc, err := openapigen.Build(cfg, dir)
		Expect(err).NotTo(HaveOccurred())

		listResp := doc.Components.Schemas["ListUsersResponse"]
		Expect(listResp).NotTo(BeNil())
		usersField := listResp.Properties["users"]
		Expect(usersField.Type).To(Equal("array"))
		Expect(usersField.Items).To(Equal(&openapigen.Schema{Ref: "#/components/schemas/User"}))
	})

	It("maps every scalar kind to its OpenAPI schema type/format", func() {
		dir := filepath.Join(root, "demo")
		protoDir := filepath.Join(dir, "contract", "pb")
		Expect(os.MkdirAll(protoDir, 0755)).To(Succeed())

		// Every field kind proto/ir_test.go's "field kind mapping" spec
		// already exercises at the proto-IR level — this checks
		// fieldToSchema/scalarSchema map each one to a valid OpenAPI
		// schema, which the CRUD starter template's string/int32/bool
		// fields alone never touch.
		Expect(os.WriteFile(filepath.Join(protoDir, "kinds.proto"), []byte(`syntax = "proto3";
package kinds.v1;
option go_package = "demo/contract/gen/kinds";

enum Status {
  STATUS_UNKNOWN = 0;
  STATUS_ACTIVE = 1;
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
}

service KindsService {
  rpc GetKinds(AllKinds) returns (AllKinds);
}
`), 0644)).To(Succeed())

		cfg := &config.Config{Module: "demo", Services: []string{"kinds"}}

		doc, err := openapigen.Build(cfg, dir)
		Expect(err).NotTo(HaveOccurred())

		props := doc.Components.Schemas["AllKinds"].Properties
		Expect(props["s"]).To(Equal(&openapigen.Schema{Type: "string"}))
		Expect(props["b"]).To(Equal(&openapigen.Schema{Type: "boolean"}))
		Expect(props["i32"]).To(Equal(&openapigen.Schema{Type: "integer", Format: "int32"}))
		Expect(props["i64"]).To(Equal(&openapigen.Schema{Type: "integer", Format: "int64"}))
		Expect(props["u32"]).To(Equal(&openapigen.Schema{Type: "integer", Format: "int32", Minimum: floatPtr(0)}))
		Expect(props["u64"]).To(Equal(&openapigen.Schema{Type: "integer", Format: "int64", Minimum: floatPtr(0)}))
		Expect(props["f"]).To(Equal(&openapigen.Schema{Type: "number", Format: "float"}))
		Expect(props["d"]).To(Equal(&openapigen.Schema{Type: "number", Format: "double"}))
		Expect(props["by"]).To(Equal(&openapigen.Schema{Type: "string", Format: "byte"}))
		Expect(props["status"]).To(Equal(&openapigen.Schema{Type: "integer", Format: "int32"}))
	})

	It("fails clearly when a registered service's proto file is missing", func() {
		dir := filepath.Join(root, "demo")
		Expect(os.MkdirAll(filepath.Join(dir, "contract", "pb"), 0755)).To(Succeed())

		cfg := &config.Config{Module: "demo", Services: []string{"ghost"}}

		_, err := openapigen.Build(cfg, dir)
		Expect(err).To(MatchError(ContainSubstring("compiling ghost.proto")))
	})

	It("fails clearly when two services define the same message name with different shapes", func() {
		dir := filepath.Join(root, "demo")
		protoDir := filepath.Join(dir, "contract", "pb")
		Expect(os.MkdirAll(protoDir, 0755)).To(Succeed())

		Expect(os.WriteFile(filepath.Join(protoDir, "a.proto"), []byte(`syntax = "proto3";
package a.v1;
option go_package = "demo/contract/gen/a";
message Shared { string name = 1; }
service AService { rpc GetA(Shared) returns (Shared); }
`), 0644)).To(Succeed())

		Expect(os.WriteFile(filepath.Join(protoDir, "b.proto"), []byte(`syntax = "proto3";
package b.v1;
option go_package = "demo/contract/gen/b";
message Shared { int32 count = 1; }
service BService { rpc GetB(Shared) returns (Shared); }
`), 0644)).To(Succeed())

		cfg := &config.Config{Module: "demo", Services: []string{"a", "b"}}

		_, err := openapigen.Build(cfg, dir)
		Expect(err).To(MatchError(ContainSubstring(`message "Shared" is defined differently`)))
	})

	It("fails clearly when two RPCs derive the same route", func() {
		dir := filepath.Join(root, "demo")
		protoDir := filepath.Join(dir, "contract", "pb")
		Expect(os.MkdirAll(protoDir, 0755)).To(Succeed())

		// Neither RPC matches the Create/Get/List/Update/Delete
		// convention, so BuildRoutes falls both back to POST on the
		// entity's base path — a genuine collision, the same one gin
		// would panic on at route-registration time.
		Expect(os.WriteFile(filepath.Join(protoDir, "widget.proto"), []byte(`syntax = "proto3";
package widget.v1;
option go_package = "demo/contract/gen/widget";
message ArchiveWidgetRequest { string note = 1; }
message RestoreWidgetRequest { string note = 1; }
message WidgetResponse { string status = 1; }
service WidgetService {
  rpc ArchiveWidget(ArchiveWidgetRequest) returns (WidgetResponse);
  rpc RestoreWidget(RestoreWidgetRequest) returns (WidgetResponse);
}
`), 0644)).To(Succeed())

		cfg := &config.Config{Module: "demo", Services: []string{"widget"}}

		_, err := openapigen.Build(cfg, dir)
		Expect(err).To(MatchError(ContainSubstring("is served by both")))
	})

	// Sensitive-field vendor extensions (ARCHITECTURE.md §22): a field
	// marked (sgo.pii) and/or (sgo.obfuscate_visible) gets x-sensitive/
	// x-obfuscate-visible on its schema, and the doc still validates —
	// vendor extensions are always meta-schema-valid, but this proves it
	// against the real validator (validate.go), not just assumed.
	It("adds x-sensitive/x-obfuscate-visible to a marked field's schema, and the field-key honors an explicit json_name", func() {
		dir := filepath.Join(root, "demo")
		protoDir := filepath.Join(dir, "contract", "pb")
		Expect(os.MkdirAll(protoDir, 0755)).To(Succeed())

		Expect(os.WriteFile(filepath.Join(protoDir, "account.proto"), []byte(`syntax = "proto3";
package account.v1;

import "sgo/options.proto";

option go_package = "demo/contract/gen/account";

message Account {
  string id = 1;
  string id_number = 2 [(sgo.obfuscate_visible) = 3, json_name = "governmentId"];
  string email = 3 [(sgo.pii) = true];
  string display_name = 4;
}

service AccountService {
  rpc GetAccount(Account) returns (Account);
}
`), 0644)).To(Succeed())

		cfg := &config.Config{Module: "demo", Services: []string{"account"}}

		doc, err := openapigen.Build(cfg, dir)
		Expect(err).NotTo(HaveOccurred())

		props := doc.Components.Schemas["Account"].Properties

		Expect(props).To(HaveKey("governmentId"))
		Expect(props).NotTo(HaveKey("id_number"))
		idNumber := props["governmentId"]
		Expect(idNumber.Type).To(Equal("string"))
		Expect(idNumber.Sensitive).To(BeTrue())
		Expect(idNumber.ObfuscateVisible).NotTo(BeNil())
		Expect(*idNumber.ObfuscateVisible).To(Equal(int32(3)))

		email := props["email"]
		Expect(email.Sensitive).To(BeTrue())
		Expect(email.ObfuscateVisible).To(BeNil())

		displayName := props["display_name"]
		Expect(displayName.Sensitive).To(BeFalse())
		Expect(displayName.ObfuscateVisible).To(BeNil())

		id := props["id"]
		Expect(id.Sensitive).To(BeFalse())

		data, err := openapigen.Encode(doc, config.OpenAPIVersion30, config.OpenAPIFormatJSON)
		Expect(err).NotTo(HaveOccurred())
		Expect(openapigen.Validate(data)).To(Succeed())
	})
})
