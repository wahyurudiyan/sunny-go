package core_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

const repositoryQueryUserProto = `syntax = "proto3";

package user.v1;

import "sgo/options.proto";

option go_package = "demo/contract/gen/user";

service UserService {
  rpc CreateUser(CreateUserRequest) returns (UserResponse);
  rpc GetUser(GetUserRequest) returns (UserResponse);

  rpc FindUserByEmail(FindUserByEmailRequest) returns (UserResponse) {
    option (sgo.repository_query) = true;
  }

  rpc PurgeStale(PurgeStaleRequest) returns (UserResponse) {
    option (sgo.repository_query) = true;
  }
}

message User {
  string id = 1;
  string name = 2;
  string email = 3;
}

message CreateUserRequest {
  string name = 1;
}

message GetUserRequest {
  string id = 1;
}

message FindUserByEmailRequest {
  string email = 1;
}

message PurgeStaleRequest {
  string older_than = 1;
  string reason = 2;
}

message UserResponse {
  User user = 1;
}
`

func compileRepositoryQueryProto(dir, content string) (protoreflect.FileDescriptor, *sgoproto.File) {
	GinkgoHelper()

	protoDir := filepath.Join(dir, "contract", "pb")
	Expect(os.MkdirAll(protoDir, 0755)).To(Succeed())
	Expect(os.WriteFile(filepath.Join(protoDir, "user.proto"), []byte(content), 0644)).To(Succeed())

	fd, err := sgoproto.Compile(protoDir, "user.proto")
	Expect(err).NotTo(HaveOccurred())
	file, err := sgoproto.Build(fd)
	Expect(err).NotTo(HaveOccurred())

	return fd, file
}

var _ = Describe("RepositoryQueryMethods", func() {
	var dir string

	BeforeEach(func() {
		var err error
		dir, err = os.MkdirTemp("", "sgo-core-repository-query-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })
	})

	It("only derives methods for RPCs marked (sgo.repository_query)", func() {
		fd, file := compileRepositoryQueryProto(dir, repositoryQueryUserProto)

		methods, err := core.RepositoryQueryMethods(fd, file, "user")
		Expect(err).NotTo(HaveOccurred())
		Expect(methods).To(HaveLen(2))

		names := []string{methods[0].Name, methods[1].Name}
		Expect(names).To(ConsistOf("FindByEmail", "PurgeStale"))
	})

	It("strips every occurrence of the entity name from the RPC name", func() {
		fd, file := compileRepositoryQueryProto(dir, repositoryQueryUserProto)

		methods, err := core.RepositoryQueryMethods(fd, file, "user")
		Expect(err).NotTo(HaveOccurred())

		var byEmail core.RepositoryQueryMethod
		for _, m := range methods {
			if m.RPC == "FindUserByEmail" {
				byEmail = m
			}
		}
		Expect(byEmail.Name).To(Equal("FindByEmail"), "FindUserByEmail on entity User should strip to FindByEmail")
	})

	It("keeps an RPC name as-is when it doesn't mention the entity", func() {
		fd, file := compileRepositoryQueryProto(dir, repositoryQueryUserProto)

		methods, err := core.RepositoryQueryMethods(fd, file, "user")
		Expect(err).NotTo(HaveOccurred())

		var purge core.RepositoryQueryMethod
		for _, m := range methods {
			if m.RPC == "PurgeStale" {
				purge = m
			}
		}
		Expect(purge.Name).To(Equal("PurgeStale"), `"PurgeStale" has no "User" in it, so it keeps its own name unchanged`)
	})

	It("flattens the request message's fields to Go parameters in declaration order", func() {
		fd, file := compileRepositoryQueryProto(dir, repositoryQueryUserProto)

		methods, err := core.RepositoryQueryMethods(fd, file, "user")
		Expect(err).NotTo(HaveOccurred())

		var purge core.RepositoryQueryMethod
		for _, m := range methods {
			if m.RPC == "PurgeStale" {
				purge = m
			}
		}
		Expect(purge.Params).To(HaveLen(2))
		Expect(purge.Params[0].GoParam).To(Equal("olderThan"))
		Expect(purge.Params[0].GoType).To(Equal("string"))
		Expect(purge.Params[1].GoParam).To(Equal("reason"))
		Expect(purge.Params[1].GoType).To(Equal("string"))
	})

	It("rejects a message-typed request field with a clear error", func() {
		content := `syntax = "proto3";

package user.v1;

import "sgo/options.proto";

option go_package = "demo/contract/gen/user";

service UserService {
  rpc FindUserByAddress(FindUserByAddressRequest) returns (UserResponse) {
    option (sgo.repository_query) = true;
  }
}

message User {
  string id = 1;
}

message Address {
  string city = 1;
}

message FindUserByAddressRequest {
  Address address = 1;
}

message UserResponse {
  User user = 1;
}
`
		fd, file := compileRepositoryQueryProto(dir, content)

		_, err := core.RepositoryQueryMethods(fd, file, "user")
		Expect(err).To(MatchError(ContainSubstring("requires every request field to be scalar")))
	})

	It("rejects a repeated request field with a clear error", func() {
		content := `syntax = "proto3";

package user.v1;

import "sgo/options.proto";

option go_package = "demo/contract/gen/user";

service UserService {
  rpc FindUsersByTag(FindUsersByTagRequest) returns (UserResponse) {
    option (sgo.repository_query) = true;
  }
}

message User {
  string id = 1;
}

message FindUsersByTagRequest {
  repeated string tags = 1;
}

message UserResponse {
  User user = 1;
}
`
		fd, file := compileRepositoryQueryProto(dir, content)

		_, err := core.RepositoryQueryMethods(fd, file, "user")
		Expect(err).To(MatchError(ContainSubstring("requires every request field to be scalar")))
	})

	It("rejects a derived method name that collides with the port's fixed CRUD methods", func() {
		content := `syntax = "proto3";

package user.v1;

import "sgo/options.proto";

option go_package = "demo/contract/gen/user";

service UserService {
  rpc GetUser(GetUserRequest) returns (UserResponse) {
    option (sgo.repository_query) = true;
  }
}

message User {
  string id = 1;
}

message GetUserRequest {
  string id = 1;
}

message UserResponse {
  User user = 1;
}
`
		fd, file := compileRepositoryQueryProto(dir, content)

		_, err := core.RepositoryQueryMethods(fd, file, "user")
		Expect(err).To(MatchError(ContainSubstring("collides with the port's fixed Get method")))
	})
})

var _ = Describe("GenerateAggregateRepositoryPort with repository_query methods", func() {
	It("adds one interface method per repository_query RPC, beyond the fixed CRUD shape", func() {
		dir, err := os.MkdirTemp("", "sgo-core-repository-query-port-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })

		fd, file := compileRepositoryQueryProto(dir, repositoryQueryUserProto)
		p := core.Paths{Module: "demo", Entity: "user"}
		domainDir := filepath.Join(dir, "internal", "domain", "user")

		Expect(core.GenerateAggregateRepositoryPort(file, fd, p, domainDir)).To(Succeed())

		content, err := os.ReadFile(filepath.Join(domainDir, "repository.go"))
		Expect(err).NotTo(HaveOccurred())
		src := string(content)

		Expect(src).To(ContainSubstring("Create(ctx context.Context, a *User) (*User, error)"))
		Expect(src).To(ContainSubstring("FindByEmail(ctx context.Context, email string) (*User, error)"))
		Expect(src).To(ContainSubstring("PurgeStale(ctx context.Context, olderThan string, reason string) (*User, error)"))
	})

	It("adds no extra methods when no RPC is marked repository_query", func() {
		dir, err := os.MkdirTemp("", "sgo-core-repository-query-port-none-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })

		content := `syntax = "proto3";

package user.v1;

option go_package = "demo/contract/gen/user";

service UserService {
  rpc CreateUser(CreateUserRequest) returns (UserResponse);
}

message User {
  string id = 1;
}

message CreateUserRequest {
  string name = 1;
}

message UserResponse {
  User user = 1;
}
`
		fd, file := compileRepositoryQueryProto(dir, content)
		p := core.Paths{Module: "demo", Entity: "user"}
		domainDir := filepath.Join(dir, "internal", "domain", "user")

		Expect(core.GenerateAggregateRepositoryPort(file, fd, p, domainDir)).To(Succeed())

		read, err := os.ReadFile(filepath.Join(domainDir, "repository.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(read)).To(ContainSubstring("Delete(ctx context.Context, id string) error\n}"), "the interface should close right after Delete, with no extra methods appended")
	})
})
