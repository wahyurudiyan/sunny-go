package grpcgen_test

import (
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/grpcgen"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/memgen"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/wiregen"
)

// grpcE2EUserProto follows the exact message shape `sgo generate
// proto`'s starter template produces — the only shape
// core.ClassifyResponse is guaranteed to derive field names from
// correctly.
const grpcE2EUserProto = `syntax = "proto3";

package user.v1;

option go_package = "demo/contract/gen/user";

service UserService {
  rpc CreateUser(CreateUserRequest) returns (UserResponse);
  rpc GetUser(GetUserRequest) returns (UserResponse);
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
  rpc UpdateUser(UpdateUserRequest) returns (UserResponse);
  rpc DeleteUser(DeleteUserRequest) returns (DeleteUserResponse);
}

message User {
  string id = 1;
  string name = 2;
}

message CreateUserRequest {
  string name = 1;
}

message GetUserRequest {
  string id = 1;
}

message ListUsersRequest {
  int32 page = 1;
  int32 page_size = 2;
}

message ListUsersResponse {
  repeated User users = 1;
  int32 total = 2;
}

message UpdateUserRequest {
  string id = 1;
  string name = 2;
}

message DeleteUserRequest {
  string id = 1;
}

message DeleteUserResponse {
  bool success = 1;
}

message UserResponse {
  User user = 1;
}
`

// grpcE2EUserServiceGo is what a developer fills the owned service.go
// stub in with — real CRUD orchestration against the aggregate
// repository, the same shape sgo scaffolds a panic("TODO") stub for.
const grpcE2EUserServiceGo = `package user

import (
	"context"

	domain "demo/internal/domain/user"
	ports "demo/internal/application/ports"
)

type UserService struct {
	repo      domain.UserRepository
	publisher ports.EventPublisher
}

func NewUserService(repo domain.UserRepository, publisher ports.EventPublisher) *UserService {
	return &UserService{repo: repo, publisher: publisher}
}

func (s *UserService) CreateUser(ctx context.Context, req *CreateUserRequest) (*domain.User, error) {
	return s.repo.Create(ctx, &domain.User{Name: req.Name})
}

func (s *UserService) GetUser(ctx context.Context, req *GetUserRequest) (*domain.User, error) {
	return s.repo.Get(ctx, req.Id)
}

func (s *UserService) ListUsers(ctx context.Context, req *ListUsersRequest) ([]*domain.User, error) {
	items, _, err := s.repo.List(ctx, 0, 0)
	return items, err
}

func (s *UserService) UpdateUser(ctx context.Context, req *UpdateUserRequest) (*domain.User, error) {
	return s.repo.Update(ctx, &domain.User{Id: req.Id, Name: req.Name})
}

func (s *UserService) DeleteUser(ctx context.Context, req *DeleteUserRequest) error {
	return s.repo.Delete(ctx, req.Id)
}
`

// This spec proves the whole DDD infrastructure-layer stack — domain
// aggregate, event kernel, application service/ports, memory
// repository, the wire<->app mapper, and the gRPC adapter — compiles
// together and actually serves real gRPC calls end to end, over a real
// in-process listener with the real generated client stub, not just
// that the generator's output looks right in isolation
// (ARCHITECTURE.md §17, task #37).
var _ = Describe("the gRPC adapter wired to the full DDD stack", func() {
	It("serves Create/Get/List/Update/Delete over real gRPC calls against the generated application service and memory repository", func() {
		root, err := os.MkdirTemp("", "sgo-grpcgen-e2e-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		protoDir := filepath.Join(root, "contract", "pb")
		Expect(os.MkdirAll(protoDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(protoDir, "user.proto"), []byte(grpcE2EUserProto), 0644)).To(Succeed())

		fd, err := sgoproto.Compile(protoDir, "user.proto")
		Expect(err).NotTo(HaveOccurred())
		file, err := sgoproto.Build(fd)
		Expect(err).NotTo(HaveOccurred())

		p := core.Paths{Module: "demo", Entity: "user"}
		eventDir := filepath.Join(root, "internal", "domain", "event")
		domainDir := filepath.Join(root, "internal", "domain", "user")
		appDir := filepath.Join(root, "internal", "application", "user")
		portsDir := filepath.Join(root, "internal", "application", "ports")
		memoryDir := filepath.Join(root, "internal", "infrastructure", "persistence", "memory")
		mapperDir := filepath.Join(root, "internal", "infrastructure", "transport")
		grpcDir := filepath.Join(root, "internal", "infrastructure", "transport", "grpc")
		wireDir := filepath.Join(root, "contract", "gen", "user")

		Expect(core.GenerateEventKernel(eventDir)).To(Succeed())
		Expect(core.GenerateAggregate(file, fd, p, domainDir)).To(Succeed())
		Expect(core.GenerateAggregateRepositoryPort(file, fd, p, domainDir)).To(Succeed())
		Expect(core.GenerateDomainErrors(fd, p, domainDir)).To(Succeed())
		Expect(core.GenerateCommandsAndQueries(file, fd, p, appDir)).To(Succeed())
		Expect(core.GenerateEventPublisher(p, portsDir)).To(Succeed())
		Expect(core.GenerateApplicationService(file, fd, p, appDir)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(appDir, "service.go"), []byte(grpcE2EUserServiceGo), 0644)).To(Succeed())

		Expect(memgen.Generate(file, fd, p, memoryDir)).To(Succeed())
		Expect(wiregen.Generate(fd, wireDir)).To(Succeed())
		Expect(core.GenerateInfraMapper(file, p, mapperDir)).To(Succeed())
		Expect(grpcgen.Generate(file, p, grpcDir)).To(Succeed())

		modDir := filepath.Join(root, "gobuild")
		Expect(os.MkdirAll(modDir, 0755)).To(Succeed())
		repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
		Expect(err).NotTo(HaveOccurred())
		goMod := "module demo\n\ngo 1.26.0\n\nrequire github.com/wahyurudiyan/sunny-go v0.0.0-00010101000000-000000000000\n\nreplace github.com/wahyurudiyan/sunny-go => " + repoRoot + "\n"
		Expect(os.WriteFile(filepath.Join(modDir, "go.mod"), []byte(goMod), 0644)).To(Succeed())

		copyDir := func(src, dstRel string) {
			GinkgoHelper()
			entries, err := os.ReadDir(src)
			Expect(err).NotTo(HaveOccurred())
			dst := filepath.Join(modDir, dstRel)
			Expect(os.MkdirAll(dst, 0755)).To(Succeed())
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				data, err := os.ReadFile(filepath.Join(src, e.Name()))
				Expect(err).NotTo(HaveOccurred())
				Expect(os.WriteFile(filepath.Join(dst, e.Name()), data, 0644)).To(Succeed())
			}
		}

		copyDir(eventDir, "internal/domain/event")
		copyDir(domainDir, "internal/domain/user")
		copyDir(portsDir, "internal/application/ports")
		copyDir(appDir, "internal/application/user")
		copyDir(memoryDir, "internal/infrastructure/persistence/memory")
		copyDir(mapperDir, "internal/infrastructure/transport")
		copyDir(grpcDir, "internal/infrastructure/transport/grpc")
		copyDir(wireDir, "contract/gen/user")

		mainSrc := `package main

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	app "demo/internal/application/user"
	"demo/internal/application/ports"
	wire "demo/contract/gen/user"
	grpcadapter "demo/internal/infrastructure/transport/grpc"
	memory "demo/internal/infrastructure/persistence/memory"
)

func main() {
	repo := memory.NewUserRepository()
	svc := app.NewUserService(repo, ports.NoopEventPublisher{})

	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	grpcadapter.RegisterUserServiceServer(server, svc)
	go func() {
		if err := server.Serve(lis); err != nil {
			panic(err)
		}
	}()
	defer server.Stop()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	client := wire.NewUserServiceClient(conn)
	ctx := context.Background()

	created, err := client.CreateUser(ctx, &wire.CreateUserRequest{Name: "Ada"})
	if err != nil {
		panic(err)
	}
	fmt.Printf("create id=%s name=%s\n", created.GetUser().GetId(), created.GetUser().GetName())

	got, err := client.GetUser(ctx, &wire.GetUserRequest{Id: created.GetUser().GetId()})
	if err != nil {
		panic(err)
	}
	fmt.Printf("get name=%s\n", got.GetUser().GetName())

	list, err := client.ListUsers(ctx, &wire.ListUsersRequest{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("list count=%d total=%d\n", len(list.GetUsers()), list.GetTotal())

	updated, err := client.UpdateUser(ctx, &wire.UpdateUserRequest{Id: created.GetUser().GetId(), Name: "Ada Lovelace"})
	if err != nil {
		panic(err)
	}
	fmt.Printf("update name=%s\n", updated.GetUser().GetName())

	deleted, err := client.DeleteUser(ctx, &wire.DeleteUserRequest{Id: created.GetUser().GetId()})
	if err != nil {
		panic(err)
	}
	fmt.Printf("delete success=%v\n", deleted.GetSuccess())

	_, err = client.GetUser(ctx, &wire.GetUserRequest{Id: created.GetUser().GetId()})
	fmt.Printf("get-after-delete error=%v\n", err != nil)
}
`
		Expect(os.WriteFile(filepath.Join(modDir, "main.go"), []byte(mainSrc), 0644)).To(Succeed())

		tidy := exec.Command("go", "mod", "tidy")
		tidy.Dir = modDir
		out, err := tidy.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(out))

		cmd := exec.Command("go", "run", ".")
		cmd.Dir = modDir
		out, err = cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(out))

		result := string(out)
		Expect(result).To(ContainSubstring("create id=user-1 name=Ada"))
		Expect(result).To(ContainSubstring("get name=Ada"))
		Expect(result).To(ContainSubstring("list count=1 total=1"))
		Expect(result).To(ContainSubstring("update name=Ada Lovelace"))
		Expect(result).To(ContainSubstring("delete success=true"))
		Expect(result).To(ContainSubstring("get-after-delete error=true"))
	})
})

// grpcE2ESensitiveUserProto adds an (sgo.obfuscate_visible) field to the
// same shape above, to prove gRPC masking (ARCHITECTURE.md §22) over a
// real client/server round trip, not just that infra_mapper_gen.go.tmpl
// renders the right source text.
const grpcE2ESensitiveUserProto = `syntax = "proto3";

package user.v1;

import "sgo/options.proto";

option go_package = "demo/contract/gen/user";

service UserService {
  rpc CreateUser(CreateUserRequest) returns (UserResponse);
  rpc GetUser(GetUserRequest) returns (UserResponse);
}

message User {
  string id = 1;
  string name = 2;
  string email = 3 [(sgo.obfuscate_visible) = 3];
}

message CreateUserRequest {
  string name = 1;
  string email = 2;
}

message GetUserRequest {
  string id = 1;
}

message UserResponse {
  User user = 1;
}
`

const grpcE2ESensitiveUserServiceGo = `package user

import (
	"context"

	domain "demo/internal/domain/user"
	ports "demo/internal/application/ports"
)

type UserService struct {
	repo      domain.UserRepository
	publisher ports.EventPublisher
}

func NewUserService(repo domain.UserRepository, publisher ports.EventPublisher) *UserService {
	return &UserService{repo: repo, publisher: publisher}
}

func (s *UserService) CreateUser(ctx context.Context, req *CreateUserRequest) (*domain.User, error) {
	return s.repo.Create(ctx, &domain.User{Name: req.Name, Email: req.Email})
}

func (s *UserService) GetUser(ctx context.Context, req *GetUserRequest) (*domain.User, error) {
	return s.repo.Get(ctx, req.Id)
}
`

var _ = Describe("the gRPC adapter masks an (sgo.obfuscate_visible) field over a real client/server round trip", func() {
	It("returns a masked email over gRPC while a direct repository read still shows the real one", func() {
		root, err := os.MkdirTemp("", "sgo-grpcgen-mask-e2e-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		protoDir := filepath.Join(root, "contract", "pb")
		Expect(os.MkdirAll(protoDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(protoDir, "user.proto"), []byte(grpcE2ESensitiveUserProto), 0644)).To(Succeed())

		fd, err := sgoproto.Compile(protoDir, "user.proto")
		Expect(err).NotTo(HaveOccurred())
		file, err := sgoproto.Build(fd)
		Expect(err).NotTo(HaveOccurred())

		p := core.Paths{Module: "demo", Entity: "user"}
		eventDir := filepath.Join(root, "internal", "domain", "event")
		maskDir := filepath.Join(root, "internal", "domain", "mask")
		domainDir := filepath.Join(root, "internal", "domain", "user")
		appDir := filepath.Join(root, "internal", "application", "user")
		portsDir := filepath.Join(root, "internal", "application", "ports")
		memoryDir := filepath.Join(root, "internal", "infrastructure", "persistence", "memory")
		mapperDir := filepath.Join(root, "internal", "infrastructure", "transport")
		grpcDir := filepath.Join(root, "internal", "infrastructure", "transport", "grpc")
		wireDir := filepath.Join(root, "contract", "gen", "user")

		Expect(core.GenerateEventKernel(eventDir)).To(Succeed())
		Expect(core.GenerateMaskKernel(maskDir)).To(Succeed())
		Expect(core.GenerateAggregate(file, fd, p, domainDir)).To(Succeed())
		Expect(core.GenerateAggregateRepositoryPort(file, fd, p, domainDir)).To(Succeed())
		Expect(core.GenerateDomainErrors(fd, p, domainDir)).To(Succeed())
		Expect(core.GenerateCommandsAndQueries(file, fd, p, appDir)).To(Succeed())
		Expect(core.GenerateEventPublisher(p, portsDir)).To(Succeed())
		Expect(core.GenerateApplicationService(file, fd, p, appDir)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(appDir, "service.go"), []byte(grpcE2ESensitiveUserServiceGo), 0644)).To(Succeed())

		Expect(memgen.Generate(file, fd, p, memoryDir)).To(Succeed())
		Expect(wiregen.Generate(fd, wireDir)).To(Succeed())
		Expect(core.GenerateInfraMapper(file, p, mapperDir)).To(Succeed())
		Expect(grpcgen.Generate(file, p, grpcDir)).To(Succeed())

		infraMapperSrc, err := os.ReadFile(filepath.Join(mapperDir, "user_mapper_gen.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(infraMapperSrc)).To(ContainSubstring("mask.Obfuscate(d.Email, 3)"))

		modDir := filepath.Join(root, "gobuild")
		Expect(os.MkdirAll(modDir, 0755)).To(Succeed())
		repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
		Expect(err).NotTo(HaveOccurred())
		goMod := "module demo\n\ngo 1.26.0\n\nrequire github.com/wahyurudiyan/sunny-go v0.0.0-00010101000000-000000000000\n\nreplace github.com/wahyurudiyan/sunny-go => " + repoRoot + "\n"
		Expect(os.WriteFile(filepath.Join(modDir, "go.mod"), []byte(goMod), 0644)).To(Succeed())

		copyDir := func(src, dstRel string) {
			GinkgoHelper()
			entries, err := os.ReadDir(src)
			Expect(err).NotTo(HaveOccurred())
			dst := filepath.Join(modDir, dstRel)
			Expect(os.MkdirAll(dst, 0755)).To(Succeed())
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				data, err := os.ReadFile(filepath.Join(src, e.Name()))
				Expect(err).NotTo(HaveOccurred())
				Expect(os.WriteFile(filepath.Join(dst, e.Name()), data, 0644)).To(Succeed())
			}
		}

		copyDir(eventDir, "internal/domain/event")
		copyDir(maskDir, "internal/domain/mask")
		copyDir(domainDir, "internal/domain/user")
		copyDir(portsDir, "internal/application/ports")
		copyDir(appDir, "internal/application/user")
		copyDir(memoryDir, "internal/infrastructure/persistence/memory")
		copyDir(mapperDir, "internal/infrastructure/transport")
		copyDir(grpcDir, "internal/infrastructure/transport/grpc")
		copyDir(wireDir, "contract/gen/user")

		mainSrc := `package main

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	app "demo/internal/application/user"
	"demo/internal/application/ports"
	wire "demo/contract/gen/user"
	grpcadapter "demo/internal/infrastructure/transport/grpc"
	memory "demo/internal/infrastructure/persistence/memory"
)

func main() {
	repo := memory.NewUserRepository()
	svc := app.NewUserService(repo, ports.NoopEventPublisher{})

	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	grpcadapter.RegisterUserServiceServer(server, svc)
	go func() {
		if err := server.Serve(lis); err != nil {
			panic(err)
		}
	}()
	defer server.Stop()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	client := wire.NewUserServiceClient(conn)
	ctx := context.Background()

	created, err := client.CreateUser(ctx, &wire.CreateUserRequest{Name: "Ada", Email: "ada@example.com"})
	if err != nil {
		panic(err)
	}
	fmt.Printf("create email=%s\n", created.GetUser().GetEmail())

	got, err := client.GetUser(ctx, &wire.GetUserRequest{Id: created.GetUser().GetId()})
	if err != nil {
		panic(err)
	}
	fmt.Printf("get email=%s\n", got.GetUser().GetEmail())

	// Direct domain-layer read, bypassing gRPC/the mapper entirely —
	// proves the real value was actually stored, not masked at rest.
	stored, err := repo.Get(ctx, created.GetUser().GetId())
	if err != nil {
		panic(err)
	}
	fmt.Printf("stored email=%s\n", stored.Email)
}
`
		Expect(os.WriteFile(filepath.Join(modDir, "main.go"), []byte(mainSrc), 0644)).To(Succeed())

		tidy := exec.Command("go", "mod", "tidy")
		tidy.Dir = modDir
		out, err := tidy.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(out))

		cmd := exec.Command("go", "run", ".")
		cmd.Dir = modDir
		out, err = cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(out))

		result := string(out)
		Expect(result).To(ContainSubstring("create email=ada*****"))
		Expect(result).To(ContainSubstring("get email=ada*****"))
		Expect(result).To(ContainSubstring("stored email=ada@example.com"))
	})
})
