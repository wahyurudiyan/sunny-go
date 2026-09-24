package httpgen_test

import (
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/httpgen"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/memgen"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
	"github.com/wahyurudiyan/sunny-go/internal/config"
)

// ginE2EUserProto follows the exact message shape `sgo generate proto`'s
// starter template produces (service.proto.tmpl) — the only shape
// classifyResponse (templatedata.go) is guaranteed to derive JSON field
// names from correctly.
const ginE2EUserProto = `syntax = "proto3";

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

// userServiceGo is what a developer fills the owned service.go stub in
// with — real CRUD orchestration against the aggregate repository, the
// same shape sgo scaffolds a panic("TODO") stub for.
const userServiceGo = `package user

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

// This spec proves the whole new DDD infrastructure-layer stack — domain
// aggregate, event kernel, application service/ports, memory repository,
// and the Gin HTTP adapter — compiles together and actually serves real
// HTTP requests end to end, not just that each generator's output looks
// right in isolation (ARCHITECTURE.md §17, task #37).
var _ = Describe("the Gin HTTP adapter wired to the full DDD stack", func() {
	It("serves Create/Get/List/Update/Delete over real HTTP requests against the generated application service and memory repository", func() {
		root, err := os.MkdirTemp("", "sgo-httpgen-e2e-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		protoDir := filepath.Join(root, "contract", "pb")
		Expect(os.MkdirAll(protoDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(protoDir, "user.proto"), []byte(ginE2EUserProto), 0644)).To(Succeed())

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
		ginDir := filepath.Join(root, "internal", "infrastructure", "transport", "http", "gin")

		Expect(core.GenerateEventKernel(eventDir)).To(Succeed())
		Expect(core.GenerateAggregate(file, fd, p, domainDir)).To(Succeed())
		Expect(core.GenerateAggregateRepositoryPort(file, fd, p, domainDir)).To(Succeed())
		Expect(core.GenerateDomainErrors(fd, p, domainDir)).To(Succeed())
		Expect(core.GenerateCommandsAndQueries(file, fd, p, appDir)).To(Succeed())
		Expect(core.GenerateEventPublisher(p, portsDir)).To(Succeed())
		Expect(core.GenerateApplicationService(file, fd, p, appDir)).To(Succeed())
		// Overwrite the generated panic("TODO") skeleton with a real
		// implementation, simulating a developer filling in the owned
		// file sgo never touches again once it exists.
		Expect(os.WriteFile(filepath.Join(appDir, "service.go"), []byte(userServiceGo), 0644)).To(Succeed())

		Expect(memgen.Generate(file, fd, p, memoryDir)).To(Succeed())
		Expect(httpgen.GenerateServer(config.HTTPFrameworkGin, ginDir)).To(Succeed())
		Expect(httpgen.GenerateRoutes(config.HTTPFrameworkGin, fd, file, p, ginDir)).To(Succeed())

		modDir := filepath.Join(root, "gobuild")
		Expect(os.MkdirAll(modDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(modDir, "go.mod"), []byte("module demo\n\ngo 1.22\n"), 0644)).To(Succeed())

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
		copyDir(ginDir, "internal/infrastructure/transport/http/gin")

		mainSrc := `package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"

	app "demo/internal/application/user"
	"demo/internal/application/ports"
	httpadapter "demo/internal/infrastructure/transport/http/gin"
	memory "demo/internal/infrastructure/persistence/memory"
)

func main() {
	gin.SetMode(gin.TestMode)
	repo := memory.NewUserRepository()
	svc := app.NewUserService(repo, ports.NoopEventPublisher{})

	engine := gin.New()
	httpadapter.RegisterUserRoutes(engine, svc)

	ts := httptest.NewServer(engine)
	defer ts.Close()

	createBody, _ := json.Marshal(map[string]string{"name": "Ada"})
	createResp, err := http.Post(ts.URL+"/api/v1/users", "application/json", bytes.NewReader(createBody))
	if err != nil {
		panic(err)
	}
	var created map[string]map[string]any
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		panic(err)
	}
	createResp.Body.Close()
	id, _ := created["user"]["id"].(string)
	fmt.Printf("create status=%d id=%s name=%v\n", createResp.StatusCode, id, created["user"]["name"])

	getResp, err := http.Get(ts.URL + "/api/v1/users/" + id)
	if err != nil {
		panic(err)
	}
	var got map[string]map[string]any
	if err := json.NewDecoder(getResp.Body).Decode(&got); err != nil {
		panic(err)
	}
	getResp.Body.Close()
	fmt.Printf("get status=%d name=%v\n", getResp.StatusCode, got["user"]["name"])

	listResp, err := http.Get(ts.URL + "/api/v1/users")
	if err != nil {
		panic(err)
	}
	var list map[string]any
	if err := json.NewDecoder(listResp.Body).Decode(&list); err != nil {
		panic(err)
	}
	listResp.Body.Close()
	users, _ := list["users"].([]any)
	fmt.Printf("list status=%d count=%d total=%v\n", listResp.StatusCode, len(users), list["total"])

	updateBody, _ := json.Marshal(map[string]string{"name": "Ada Lovelace"})
	updateReq, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/users/"+id, bytes.NewReader(updateBody))
	updateResp, err := http.DefaultClient.Do(updateReq)
	if err != nil {
		panic(err)
	}
	var updated map[string]map[string]any
	if err := json.NewDecoder(updateResp.Body).Decode(&updated); err != nil {
		panic(err)
	}
	updateResp.Body.Close()
	fmt.Printf("update status=%d name=%v\n", updateResp.StatusCode, updated["user"]["name"])

	delReq, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/users/"+id, nil)
	delResp, err := http.DefaultClient.Do(delReq)
	if err != nil {
		panic(err)
	}
	var deleted map[string]any
	if err := json.NewDecoder(delResp.Body).Decode(&deleted); err != nil {
		panic(err)
	}
	delResp.Body.Close()
	fmt.Printf("delete status=%d success=%v\n", delResp.StatusCode, deleted["success"])

	afterResp, err := http.Get(ts.URL + "/api/v1/users/" + id)
	if err != nil {
		panic(err)
	}
	fmt.Printf("get-after-delete status=%d\n", afterResp.StatusCode)
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
		Expect(result).To(ContainSubstring("create status=200 id=user-1 name=Ada"))
		Expect(result).To(ContainSubstring("get status=200 name=Ada"))
		Expect(result).To(ContainSubstring("list status=200 count=1 total=1"))
		Expect(result).To(ContainSubstring("update status=200 name=Ada Lovelace"))
		Expect(result).To(ContainSubstring("delete status=200 success=true"))
		Expect(result).To(ContainSubstring("get-after-delete status=500"))
	})
})

// ginE2ESensitiveUserProto adds an (sgo.obfuscate_visible) field to the
// same shape above, to prove HTTP masking (ARCHITECTURE.md §22) over a
// real request/response round trip — specifically the non-destructive
// MarshalJSON shadow-struct approach, since the Gin adapter serializes
// the exact *domain.User pointer the in-memory repository still holds
// (httpgen's gin_routes.go.tmpl: `c.JSON(http.StatusOK, gin.H{"user":
// resp})`), not a copy.
const ginE2ESensitiveUserProto = `syntax = "proto3";

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
  string id_number = 3 [(sgo.obfuscate_visible) = 3];
}

message CreateUserRequest {
  string name = 1;
  string id_number = 2;
}

message GetUserRequest {
  string id = 1;
}

message UserResponse {
  User user = 1;
}
`

const sensitiveUserServiceGo = `package user

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
	return s.repo.Create(ctx, &domain.User{Name: req.Name, IdNumber: req.IdNumber})
}

func (s *UserService) GetUser(ctx context.Context, req *GetUserRequest) (*domain.User, error) {
	return s.repo.Get(ctx, req.Id)
}
`

var _ = Describe("the Gin HTTP adapter masks an (sgo.obfuscate_visible) field, without mutating the stored value", func() {
	It("returns a masked id_number over HTTP while the in-memory repository still holds the real one", func() {
		root, err := os.MkdirTemp("", "sgo-httpgen-mask-e2e-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		protoDir := filepath.Join(root, "contract", "pb")
		Expect(os.MkdirAll(protoDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(protoDir, "user.proto"), []byte(ginE2ESensitiveUserProto), 0644)).To(Succeed())

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
		ginDir := filepath.Join(root, "internal", "infrastructure", "transport", "http", "gin")

		Expect(core.GenerateEventKernel(eventDir)).To(Succeed())
		Expect(core.GenerateMaskKernel(maskDir)).To(Succeed())
		Expect(core.GenerateAggregate(file, fd, p, domainDir)).To(Succeed())
		Expect(core.GenerateAggregateRepositoryPort(file, fd, p, domainDir)).To(Succeed())
		Expect(core.GenerateDomainErrors(fd, p, domainDir)).To(Succeed())
		Expect(core.GenerateCommandsAndQueries(file, fd, p, appDir)).To(Succeed())
		Expect(core.GenerateEventPublisher(p, portsDir)).To(Succeed())
		Expect(core.GenerateApplicationService(file, fd, p, appDir)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(appDir, "service.go"), []byte(sensitiveUserServiceGo), 0644)).To(Succeed())

		Expect(memgen.Generate(file, fd, p, memoryDir)).To(Succeed())
		Expect(httpgen.GenerateServer(config.HTTPFrameworkGin, ginDir)).To(Succeed())
		Expect(httpgen.GenerateRoutes(config.HTTPFrameworkGin, fd, file, p, ginDir)).To(Succeed())

		domainSrc, err := os.ReadFile(filepath.Join(domainDir, "user_gen.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(domainSrc)).To(ContainSubstring("func (a *User) MarshalJSON() ([]byte, error)"))

		modDir := filepath.Join(root, "gobuild")
		Expect(os.MkdirAll(modDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(modDir, "go.mod"), []byte("module demo\n\ngo 1.22\n"), 0644)).To(Succeed())

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
		copyDir(ginDir, "internal/infrastructure/transport/http/gin")

		mainSrc := `package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"

	app "demo/internal/application/user"
	"demo/internal/application/ports"
	httpadapter "demo/internal/infrastructure/transport/http/gin"
	memory "demo/internal/infrastructure/persistence/memory"
)

func main() {
	gin.SetMode(gin.TestMode)
	repo := memory.NewUserRepository()
	svc := app.NewUserService(repo, ports.NoopEventPublisher{})

	engine := gin.New()
	httpadapter.RegisterUserRoutes(engine, svc)

	ts := httptest.NewServer(engine)
	defer ts.Close()

	createBody, _ := json.Marshal(map[string]string{"name": "Ada", "id_number": "123456789"})
	createResp, err := http.Post(ts.URL+"/api/v1/users", "application/json", bytes.NewReader(createBody))
	if err != nil {
		panic(err)
	}
	var created map[string]map[string]any
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		panic(err)
	}
	createResp.Body.Close()
	id, _ := created["user"]["id"].(string)
	fmt.Printf("create id_number=%v\n", created["user"]["id_number"])

	getResp, err := http.Get(ts.URL + "/api/v1/users/" + id)
	if err != nil {
		panic(err)
	}
	var got map[string]map[string]any
	if err := json.NewDecoder(getResp.Body).Decode(&got); err != nil {
		panic(err)
	}
	getResp.Body.Close()
	fmt.Printf("get id_number=%v\n", got["user"]["id_number"])

	// Direct in-memory repository read, bypassing HTTP/JSON entirely —
	// proves the stored value was never mutated by serializing it.
	stored, err := repo.Get(nil, id)
	if err != nil {
		panic(err)
	}
	fmt.Printf("stored id_number=%s\n", stored.IdNumber)
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
		Expect(result).To(ContainSubstring("create id_number=123*****"))
		Expect(result).To(ContainSubstring("get id_number=123*****"))
		Expect(result).To(ContainSubstring("stored id_number=123456789"))
	})
})
