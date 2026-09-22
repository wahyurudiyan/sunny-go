package httpgen_test

// End-to-end coverage for proto-defined HTTP paths (PLAN.md Phase 15):
// an RPC carrying a real `(google.api.http)` annotation drives the
// generated Gin route instead of the naming-convention derivation, a
// service's `(sgo.base_path)` override replaces the default "/api/v1"
// prefix for every route on it (annotated or convention-derived alike),
// and a path with more than one named parameter (not just "id") binds
// each one correctly — proven the same way gin_e2e_test.go proves the
// convention-derivation path: a real generated project, a real Gin
// server, real HTTP requests over a real socket.

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

const annotatedE2EUserProto = `syntax = "proto3";

package user.v1;

import "google/api/annotations.proto";
import "sgo/options.proto";

option go_package = "demo/contract/gen/user";

service UserService {
  option (sgo.base_path) = "/v2";

  rpc CreateUser(CreateUserRequest) returns (UserResponse);

  rpc GetUser(GetUserRequest) returns (UserResponse) {
    option (google.api.http) = {
      get: "/accounts/{id}"
    };
  }

  rpc ArchiveUser(ArchiveUserRequest) returns (UserResponse) {
    option (google.api.http) = {
      post: "/accounts/{account_id}/archive/{reason_code}"
    };
  }
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

message ArchiveUserRequest {
  string account_id = 1;
  string reason_code = 2;
}

message UserResponse {
  User user = 1;
}
`

// annotatedE2EUserServiceGo hand-implements the owned service.go stub —
// ArchiveUser deliberately echoes both path-bound fields into the
// returned user's id/name so the test can assert both were bound
// correctly from the URL, not just the first one.
const annotatedE2EUserServiceGo = `package user

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

func (s *UserService) ArchiveUser(ctx context.Context, req *ArchiveUserRequest) (*domain.User, error) {
	return &domain.User{Id: req.AccountId, Name: req.ReasonCode}, nil
}
`

var _ = Describe("proto-defined HTTP paths (google.api.http) and base_path", func() {
	It("routes an annotated RPC to its declared path, binds multiple named path params, and applies base_path to every route on the service", func() {
		root, err := os.MkdirTemp("", "sgo-httpgen-annotated-e2e-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		protoDir := filepath.Join(root, "contract", "pb")
		Expect(os.MkdirAll(protoDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(protoDir, "user.proto"), []byte(annotatedE2EUserProto), 0644)).To(Succeed())

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
		Expect(os.WriteFile(filepath.Join(appDir, "service.go"), []byte(annotatedE2EUserServiceGo), 0644)).To(Succeed())

		Expect(memgen.Generate(file, fd, p, memoryDir)).To(Succeed())
		Expect(httpgen.GenerateServer(config.HTTPFrameworkGin, ginDir)).To(Succeed())
		Expect(httpgen.GenerateRoutes(config.HTTPFrameworkGin, fd, file, p, ginDir)).To(Succeed())

		// Regression guard, checked directly against the generated
		// source before even trying to run it: the annotated routes use
		// their declared paths (base_path-prefixed), and the
		// unannotated CreateUser still uses the naming-convention path
		// (also base_path-prefixed) — never "/api/v1".
		routesSrc, err := os.ReadFile(filepath.Join(ginDir, "user_routes_gen.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(routesSrc)).To(ContainSubstring(`engine.POST("/v2/users"`))
		Expect(string(routesSrc)).To(ContainSubstring(`engine.GET("/v2/accounts/:id"`))
		Expect(string(routesSrc)).To(ContainSubstring(`engine.POST("/v2/accounts/:account_id/archive/:reason_code"`))
		Expect(string(routesSrc)).NotTo(ContainSubstring("/api/v1"))

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
	createResp, err := http.Post(ts.URL+"/v2/users", "application/json", bytes.NewReader(createBody))
	if err != nil {
		panic(err)
	}
	var created map[string]map[string]any
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		panic(err)
	}
	createResp.Body.Close()
	id, _ := created["user"]["id"].(string)
	fmt.Printf("create status=%d id=%s\n", createResp.StatusCode, id)

	getResp, err := http.Get(ts.URL + "/v2/accounts/" + id)
	if err != nil {
		panic(err)
	}
	var got map[string]map[string]any
	if err := json.NewDecoder(getResp.Body).Decode(&got); err != nil {
		panic(err)
	}
	getResp.Body.Close()
	fmt.Printf("get status=%d name=%v\n", getResp.StatusCode, got["user"]["name"])

	archiveResp, err := http.Post(ts.URL+"/v2/accounts/"+id+"/archive/duplicate", "application/json", nil)
	if err != nil {
		panic(err)
	}
	var archived map[string]map[string]any
	if err := json.NewDecoder(archiveResp.Body).Decode(&archived); err != nil {
		panic(err)
	}
	archiveResp.Body.Close()
	fmt.Printf("archive status=%d account_id=%v reason_code=%v\n", archiveResp.StatusCode, archived["user"]["id"], archived["user"]["name"])
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
		Expect(result).To(ContainSubstring("create status=200 id=user-1"))
		Expect(result).To(ContainSubstring("get status=200 name=Ada"))
		Expect(result).To(ContainSubstring("archive status=200 account_id=user-1 reason_code=duplicate"))
	})
})
