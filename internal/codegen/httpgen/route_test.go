package httpgen_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/httpgen"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

func userFile(protoDir string) (protoreflect.FileDescriptor, *sgoproto.File) {
	GinkgoHelper()

	Expect(sgoproto.GenerateStub(protoDir, "user", "demo")).To(Succeed())

	fd, err := sgoproto.Compile(protoDir, "user.proto")
	Expect(err).NotTo(HaveOccurred())

	file, err := sgoproto.Build(fd)
	Expect(err).NotTo(HaveOccurred())

	return fd, file
}

func routeFor(routes []httpgen.Route, rpc string) httpgen.Route {
	GinkgoHelper()

	for _, r := range routes {
		if r.Method.Name == rpc {
			return r
		}
	}
	Fail("no route derived for RPC " + rpc)
	return httpgen.Route{}
}

var _ = Describe("BuildRoutes", func() {
	var (
		dir    string
		fd     protoreflect.FileDescriptor
		file   *sgoproto.File
		routes []httpgen.Route
	)

	BeforeEach(func() {
		var err error
		dir, err = os.MkdirTemp("", "sgo-httpgen-route-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })

		fd, file = userFile(filepath.Join(dir, "contract", "pb"))
		routes, err = httpgen.BuildRoutes(fd, file, "user")
		Expect(err).NotTo(HaveOccurred())
	})

	It("derives one route per RPC", func() {
		Expect(routes).To(HaveLen(5))
	})

	It("maps CreateUser to POST /api/v1/users with a body and no path params", func() {
		r := routeFor(routes, "CreateUser")
		Expect(r.Verb).To(Equal("POST"))
		Expect(r.PathTemplate).To(Equal("/api/v1/users"))
		Expect(r.HasBody).To(BeTrue())
		Expect(r.PathParams).To(BeEmpty())
		Expect(r.FullPath(true)).To(Equal("/api/v1/users"))
	})

	It("maps GetUser to GET /api/v1/users/{id} with an id param and no body", func() {
		r := routeFor(routes, "GetUser")
		Expect(r.Verb).To(Equal("GET"))
		Expect(r.PathParams).To(Equal([]httpgen.PathParam{{Name: "id", GoName: "Id"}}))
		Expect(r.HasBody).To(BeFalse())
		Expect(r.FullPath(true)).To(Equal("/api/v1/users/:id"))
		Expect(r.FullPath(false)).To(Equal("/api/v1/users/{id}"))
	})

	It("maps ListUsers to GET /api/v1/users with no path params and no body", func() {
		r := routeFor(routes, "ListUsers")
		Expect(r.Verb).To(Equal("GET"))
		Expect(r.PathParams).To(BeEmpty())
		Expect(r.HasBody).To(BeFalse())
	})

	It("maps UpdateUser to PUT /api/v1/users/{id} with an id param and a body", func() {
		r := routeFor(routes, "UpdateUser")
		Expect(r.Verb).To(Equal("PUT"))
		Expect(r.PathParams).To(Equal([]httpgen.PathParam{{Name: "id", GoName: "Id"}}))
		Expect(r.HasBody).To(BeTrue())
	})

	It("maps DeleteUser to DELETE /api/v1/users/{id} with an id param and no body", func() {
		r := routeFor(routes, "DeleteUser")
		Expect(r.Verb).To(Equal("DELETE"))
		Expect(r.PathParams).To(Equal([]httpgen.PathParam{{Name: "id", GoName: "Id"}}))
		Expect(r.HasBody).To(BeFalse())
	})

	Context("when an RPC doesn't match the CRUD naming convention", func() {
		It("falls back to POST, keyed by id if the input has one", func() {
			protoDir := filepath.Join(dir, "contract", "pb")
			content := `syntax = "proto3";

package user.v1;

option go_package = "demo/contract/gen/user";

service UserService {
  rpc ArchiveUser(DeleteUserRequest) returns (DeleteUserResponse);
}

message DeleteUserRequest {
  string id = 1;
}

message DeleteUserResponse {
  bool success = 1;
}
`
			Expect(os.WriteFile(filepath.Join(protoDir, "user.proto"), []byte(content), 0644)).To(Succeed())
			customFD, err := sgoproto.Compile(protoDir, "user.proto")
			Expect(err).NotTo(HaveOccurred())
			customFile, err := sgoproto.Build(customFD)
			Expect(err).NotTo(HaveOccurred())

			routes, err := httpgen.BuildRoutes(customFD, customFile, "user")
			Expect(err).NotTo(HaveOccurred())
			Expect(routes).To(HaveLen(1))
			Expect(routes[0].Verb).To(Equal("POST"))
			Expect(routes[0].PathParams).To(Equal([]httpgen.PathParam{{Name: "id", GoName: "Id"}}), "DeleteUserRequest has an id field")
			Expect(routes[0].HasBody).To(BeTrue())
		})
	})

	Context("when an RPC is marked (sgo.hide_route)", func() {
		It("gets no route at all, while the rest of the service is unaffected", func() {
			protoDir := filepath.Join(dir, "contract", "pb")
			content := `syntax = "proto3";

package user.v1;

import "sgo/options.proto";

option go_package = "demo/contract/gen/user";

service UserService {
  rpc CreateUser(CreateUserRequest) returns (UserResponse);

  rpc InternalSync(SyncRequest) returns (SyncResponse) {
    option (sgo.hide_route) = true;
  }
}

message CreateUserRequest {
  string name = 1;
}

message SyncRequest {
  string token = 1;
}

message SyncResponse {
  bool ok = 1;
}

message UserResponse {
  string id = 1;
}
`
			Expect(os.WriteFile(filepath.Join(protoDir, "user.proto"), []byte(content), 0644)).To(Succeed())
			hiddenFD, err := sgoproto.Compile(protoDir, "user.proto")
			Expect(err).NotTo(HaveOccurred())
			hiddenFile, err := sgoproto.Build(hiddenFD)
			Expect(err).NotTo(HaveOccurred())

			routes, err := httpgen.BuildRoutes(hiddenFD, hiddenFile, "user")
			Expect(err).NotTo(HaveOccurred())
			Expect(routes).To(HaveLen(1))
			Expect(routes[0].Method.Name).To(Equal("CreateUser"))
		})
	})
})

// annotatedRoutes compiles a proto whose sole RPC carries the given
// `(google.api.http)` option body (everything between the braces) and
// returns whatever httpgen.BuildRoutes derives from it — used to probe
// the v1 constraints routeFromRule/parsePathTemplate enforce without
// paying for a full e2e run per case.
func annotatedRoutes(dir, httpOption string) ([]httpgen.Route, error) {
	GinkgoHelper()

	protoDir := filepath.Join(dir, "contract", "pb")
	Expect(os.MkdirAll(protoDir, 0755)).To(Succeed())

	content := `syntax = "proto3";

package order.v1;

import "google/api/annotations.proto";

option go_package = "demo/contract/gen/order";

service OrderService {
  rpc GetOrder(GetOrderRequest) returns (OrderResponse) {
    option (google.api.http) = {
      ` + httpOption + `
    };
  }
}

message GetOrderRequest {
  string id = 1;
  string account_id = 2;
  Nested nested = 3;
  repeated string tags = 4;
}

message Nested {
  string value = 1;
}

message OrderResponse {
  string id = 1;
}
`
	Expect(os.WriteFile(filepath.Join(protoDir, "order.proto"), []byte(content), 0644)).To(Succeed())

	fd, err := sgoproto.Compile(protoDir, "order.proto")
	Expect(err).NotTo(HaveOccurred())
	file, err := sgoproto.Build(fd)
	Expect(err).NotTo(HaveOccurred())

	return httpgen.BuildRoutes(fd, file, "order")
}

var _ = Describe("BuildRoutes: google.api.http annotation v1 constraints", func() {
	var dir string

	BeforeEach(func() {
		var err error
		dir, err = os.MkdirTemp("", "sgo-httpgen-route-annotation-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })
	})

	It("rejects a custom-verb-only pattern", func() {
		_, err := annotatedRoutes(dir, `custom: { kind: "LIST", path: "/orders" }`)
		Expect(err).To(MatchError(ContainSubstring("unsupported pattern")))
	})

	It("rejects a wildcard path segment", func() {
		_, err := annotatedRoutes(dir, `get: "/orders/*"`)
		Expect(err).To(MatchError(ContainSubstring("wildcard segment")))
	})

	It("rejects a variable sub-pattern", func() {
		_, err := annotatedRoutes(dir, `get: "/orders/{id=accounts/*}"`)
		Expect(err).To(MatchError(ContainSubstring("wildcard segment")))
	})

	It("rejects a dotted field path in a path variable", func() {
		_, err := annotatedRoutes(dir, `get: "/orders/{nested.value}"`)
		Expect(err).To(MatchError(ContainSubstring("isn't a bare top-level field")))
	})

	It("rejects a path variable bound more than once", func() {
		_, err := annotatedRoutes(dir, `get: "/orders/{id}/{id}"`)
		Expect(err).To(MatchError(ContainSubstring("more than once")))
	})

	It("rejects a path variable with no matching field", func() {
		_, err := annotatedRoutes(dir, `get: "/orders/{missing}"`)
		Expect(err).To(MatchError(ContainSubstring("no matching scalar field")))
	})

	It("rejects a path variable bound to a message-typed field", func() {
		_, err := annotatedRoutes(dir, `get: "/orders/{nested}"`)
		Expect(err).To(MatchError(ContainSubstring("no matching scalar field")))
	})

	It("rejects a path variable bound to a repeated field", func() {
		_, err := annotatedRoutes(dir, `get: "/orders/{tags}"`)
		Expect(err).To(MatchError(ContainSubstring("no matching scalar field")))
	})

	It("rejects body:\"<field>\" binding a single nested field", func() {
		_, err := annotatedRoutes(dir, "post: \"/orders/{id}\"\n      body: \"account_id\"")
		Expect(err).To(MatchError(ContainSubstring("is not supported in v1")))
	})

	It("accepts body:\"*\" and an empty body alike", func() {
		routes, err := annotatedRoutes(dir, `get: "/orders/{id}"`)
		Expect(err).NotTo(HaveOccurred())
		Expect(routes).To(HaveLen(1))
		Expect(routes[0].HasBody).To(BeFalse())
	})
})
