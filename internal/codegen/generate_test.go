package codegen_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/codegen"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/project"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
	"github.com/wahyurudiyan/sunny-go/internal/config"
)

// scaffoldProject builds a real project (Phase 1's scaffolder) plus a
// starter user.proto, so these tests exercise `sgo init` +
// `sgo generate proto` + `sgo generate code` together end to end. No
// persistence engine is selected — these specs are about the HTTP/gRPC
// wiring, so they run against the in-memory default, fast and isolated
// from any external database. See "with a real Postgres" below for the
// Phase 4-specific persistence path, which needs its own real Postgres
// and its own test isolation (a shared table to reset between runs).
func scaffoldProject(root string) string {
	GinkgoHelper()

	dir := filepath.Join(root, "demo")
	opts := project.Options{
		Name:          "demo",
		Module:        "demo",
		HTTPFramework: config.HTTPFrameworkGin,
	}
	Expect(project.Scaffold(dir, opts)).To(Succeed())
	Expect(sgoproto.GenerateStub(filepath.Join(dir, "contract", "pb"), "user", "demo")).To(Succeed())

	return dir
}

var _ = Describe("GenerateCode", func() {
	var (
		root, dir string
	)

	BeforeEach(func() {
		var err error
		root, err = os.MkdirTemp("", "sgo-codegen-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		dir = scaffoldProject(root)
	})

	It("generates everything that derives from the proto and produces a project that builds", func() {
		cfg, err := config.Load(dir)
		Expect(err).NotTo(HaveOccurred())

		Expect(codegen.GenerateCode(dir, "user", cfg)).To(Succeed())

		for _, f := range []string{
			filepath.Join("contract", "gen", "user", "user.pb.go"),
			filepath.Join("contract", "gen", "user", "user_grpc.pb.go"),
			filepath.Join("internal", "core", "domain", "user", "user_gen.go"),
			filepath.Join("internal", "core", "domain", "user", "user.go"),
			filepath.Join("internal", "core", "port", "in", "user_usecase.go"),
			filepath.Join("internal", "core", "port", "out", "user_repository.go"),
			filepath.Join("internal", "core", "service", "user_service.go"),
			filepath.Join("internal", "adapter", "mapper", "user_mapper_gen.go"),
			filepath.Join("internal", "adapter", "in", "http", "gin", "user_routes_gen.go"),
			filepath.Join("internal", "adapter", "in", "grpc", "user_grpc_server_gen.go"),
			filepath.Join("internal", "adapter", "out", "persistence", "memory", "user_repository_gen.go"),
			filepath.Join("internal", "bootstrap", "wire_gen.go"),
		} {
			Expect(filepath.Join(dir, f)).To(BeAnExistingFile(), f)
		}

		cmd := exec.Command("go", "build", "./...")
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(out))
	})

	It("records the service in sgo.yaml", func() {
		cfg, err := config.Load(dir)
		Expect(err).NotTo(HaveOccurred())

		Expect(codegen.GenerateCode(dir, "user", cfg)).To(Succeed())

		got, err := config.Load(dir)
		Expect(err).NotTo(HaveOccurred())
		Expect(got.Services).To(ConsistOf("user"))
	})

	It("errors clearly when the proto file doesn't exist", func() {
		cfg, err := config.Load(dir)
		Expect(err).NotTo(HaveOccurred())

		err = codegen.GenerateCode(dir, "order", cfg)

		Expect(err).To(MatchError(ContainSubstring("sgo generate proto order")))
	})

	It("preserves hand-written business logic across a second run and still builds", func() {
		cfg, err := config.Load(dir)
		Expect(err).NotTo(HaveOccurred())

		Expect(codegen.GenerateCode(dir, "user", cfg)).To(Succeed())

		servicePath := filepath.Join(dir, "internal", "core", "service", "user_service.go")
		content, err := os.ReadFile(servicePath)
		Expect(err).NotTo(HaveOccurred())
		stub := `panic("sgo: TODO implement GetUser")`
		Expect(string(content)).To(ContainSubstring(stub))
		body := `u, err := s.repo.Get(ctx, req.Id)
			if err != nil {
				return nil, err
			}
			return &user.UserResponse{User: u}, nil`
		updated := strings.Replace(string(content), stub, body, 1)
		Expect(os.WriteFile(servicePath, []byte(updated), 0644)).To(Succeed())

		Expect(codegen.GenerateCode(dir, "user", cfg)).To(Succeed())

		content, err = os.ReadFile(servicePath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("s.repo.Get(ctx, req.Id)"))

		cmd := exec.Command("go", "build", "./...")
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(out))
	})
})

const userServiceImplementation = `package service

import (
	"context"

	user "demo/internal/core/domain/user"
	out "demo/internal/core/port/out"
)

type UserService struct {
	repo out.UserRepository
}

func NewUserService(repo out.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) CreateUser(ctx context.Context, req *user.CreateUserRequest) (*user.UserResponse, error) {
	created, err := s.repo.Create(ctx, &user.User{Name: req.Name, Description: req.Description})
	if err != nil {
		return nil, err
	}
	return &user.UserResponse{User: created}, nil
}

func (s *UserService) GetUser(ctx context.Context, req *user.GetUserRequest) (*user.UserResponse, error) {
	u, err := s.repo.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &user.UserResponse{User: u}, nil
}

func (s *UserService) ListUsers(ctx context.Context, req *user.ListUsersRequest) (*user.ListUsersResponse, error) {
	users, total, err := s.repo.List(ctx, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}
	return &user.ListUsersResponse{Users: users, Total: total}, nil
}

func (s *UserService) UpdateUser(ctx context.Context, req *user.UpdateUserRequest) (*user.UserResponse, error) {
	updated, err := s.repo.Update(ctx, &user.User{Id: req.Id, Name: req.Name, Description: req.Description})
	if err != nil {
		return nil, err
	}
	return &user.UserResponse{User: updated}, nil
}

func (s *UserService) DeleteUser(ctx context.Context, req *user.DeleteUserRequest) (*user.DeleteUserResponse, error) {
	if err := s.repo.Delete(ctx, req.Id); err != nil {
		return nil, err
	}
	return &user.DeleteUserResponse{Success: true}, nil
}
`

// Describes the actual exit criterion for this phase: a generated
// project starts a real HTTP server whose routes match the proto's
// CRUD-naming-derived convention, and a real gRPC server on a separate
// port, both backed by the same service instance. Runs the compiled
// binary as a subprocess and drives it over real HTTP rather than
// calling generated code in-process, since that's the only way to
// verify the whole wire-up (bootstrap -> adapters -> service ->
// in-memory repo) actually works together.
var _ = Describe("a generated and implemented project, running for real", func() {
	It("serves a working HTTP CRUD API backed by the in-memory repository", func() {
		root, err := os.MkdirTemp("", "sgo-codegen-runtime-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		dir := scaffoldProject(root)

		cfg, err := config.Load(dir)
		Expect(err).NotTo(HaveOccurred())
		Expect(codegen.GenerateCode(dir, "user", cfg)).To(Succeed())

		servicePath := filepath.Join(dir, "internal", "core", "service", "user_service.go")
		Expect(os.WriteFile(servicePath, []byte(userServiceImplementation), 0644)).To(Succeed())

		binPath := filepath.Join(dir, "bin", "demo")
		buildCmd := exec.Command("go", "build", "-o", binPath, "./cmd/demo")
		buildCmd.Dir = dir
		out, err := buildCmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(out))

		server := exec.Command(binPath)
		var serverOutput bytes.Buffer
		server.Stdout = &serverOutput
		server.Stderr = &serverOutput
		Expect(server.Start()).To(Succeed())
		DeferCleanup(func() {
			_ = server.Process.Kill()
			_, _ = server.Process.Wait()
		})

		baseURL := "http://127.0.0.1:8080/api/v1/users"
		client := &http.Client{Timeout: 2 * time.Second}
		Eventually(func() error {
			_, err := client.Get(baseURL)
			return err
		}, 5*time.Second, 100*time.Millisecond).Should(Succeed(), serverOutput.String())

		createResp := postJSON(client, baseURL, `{"name":"Ada","description":"engineer"}`)
		Expect(createResp["user"].(map[string]any)["name"]).To(Equal("Ada"))
		id := createResp["user"].(map[string]any)["id"].(string)
		Expect(id).NotTo(BeEmpty())

		postJSON(client, baseURL, `{"name":"Grace","description":"admiral"}`)

		list := getJSON(client, baseURL)
		Expect(list["total"]).To(Equal(2.0))
		Expect(list["users"]).To(HaveLen(2), "regression check: List must return records, not an empty slice, for the default (unpaginated) request")

		got := getJSON(client, baseURL+"/"+id)
		Expect(got["user"].(map[string]any)["name"]).To(Equal("Ada"))

		updated := putJSON(client, baseURL+"/"+id, `{"name":"Ada L","description":"senior engineer"}`)
		Expect(updated["user"].(map[string]any)["name"]).To(Equal("Ada L"))

		req, err := http.NewRequest(http.MethodDelete, baseURL+"/"+id, nil)
		Expect(err).NotTo(HaveOccurred())
		delResp, err := client.Do(req)
		Expect(err).NotTo(HaveOccurred())
		Expect(delResp.StatusCode).To(Equal(http.StatusOK))

		afterDelete := getJSON(client, baseURL)
		Expect(afterDelete["total"]).To(Equal(1.0))
	})
})

// Complements the in-memory runtime test above with the Phase 4 path:
// a project whose sgo.yaml actually selects Postgres must get a
// generated project wired to the real adapter (not memory), and data
// must survive the server process restarting — proving real durable
// persistence, not just a working HTTP round trip. Needs its own real,
// locally reachable Postgres and its own table cleanup, since it shares
// sqlgen's own test suite's default connection target.
var _ = Describe("a generated project with Postgres selected, running for real", func() {
	It("persists data across a server restart", func() {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:5432", 500*time.Millisecond)
		if err != nil {
			Skip("no local Postgres reachable on 127.0.0.1:5432: " + err.Error())
			return
		}
		conn.Close()

		dropCmd := exec.Command("psql", "-h", "127.0.0.1", "-U", "postgres", "-d", "postgres", "-c", "DROP TABLE IF EXISTS users;")
		dropCmd.Env = append(os.Environ(), "PGPASSWORD=postgres")
		_ = dropCmd.Run()

		root, err := os.MkdirTemp("", "sgo-codegen-postgres-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		dir := filepath.Join(root, "demo")
		opts := project.Options{
			Name:          "demo",
			Module:        "demo",
			HTTPFramework: config.HTTPFrameworkGin,
			Persistence: config.Persistence{
				Mode:    config.PersistenceModeORM,
				Engines: []config.PersistenceEngine{config.PersistenceEnginePostgres},
			},
		}
		Expect(project.Scaffold(dir, opts)).To(Succeed())
		Expect(sgoproto.GenerateStub(filepath.Join(dir, "contract", "pb"), "user", "demo")).To(Succeed())

		cfg, err := config.Load(dir)
		Expect(err).NotTo(HaveOccurred())
		Expect(codegen.GenerateCode(dir, "user", cfg)).To(Succeed())

		Expect(filepath.Join(dir, "internal", "adapter", "out", "persistence", "postgres", "user_repository_gen.go")).To(BeAnExistingFile())

		wireContent, err := os.ReadFile(filepath.Join(dir, "internal", "bootstrap", "wire_gen.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(wireContent)).To(ContainSubstring("postgres.NewUserRepository(db)"), "must use the real adapter, not memory, once Postgres is selected")

		servicePath := filepath.Join(dir, "internal", "core", "service", "user_service.go")
		Expect(os.WriteFile(servicePath, []byte(userServiceImplementation), 0644)).To(Succeed())

		binPath := filepath.Join(dir, "bin", "demo")
		buildCmd := exec.Command("go", "build", "-o", binPath, "./cmd/demo")
		buildCmd.Dir = dir
		out, err := buildCmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(out))

		baseURL := "http://127.0.0.1:8080/api/v1/users"
		client := &http.Client{Timeout: 2 * time.Second}

		firstRun := exec.Command(binPath)
		var firstOutput bytes.Buffer
		firstRun.Stdout = &firstOutput
		firstRun.Stderr = &firstOutput
		Expect(firstRun.Start()).To(Succeed())

		Eventually(func() error {
			_, err := client.Get(baseURL)
			return err
		}, 5*time.Second, 100*time.Millisecond).Should(Succeed(), firstOutput.String())

		created := postJSON(client, baseURL, `{"name":"Ada","description":"engineer"}`)
		id := created["user"].(map[string]any)["id"].(string)
		Expect(id).NotTo(BeEmpty())

		Expect(firstRun.Process.Kill()).To(Succeed())
		_, _ = firstRun.Process.Wait()

		secondRun := exec.Command(binPath)
		var secondOutput bytes.Buffer
		secondRun.Stdout = &secondOutput
		secondRun.Stderr = &secondOutput
		Expect(secondRun.Start()).To(Succeed())
		DeferCleanup(func() {
			_ = secondRun.Process.Kill()
			_, _ = secondRun.Process.Wait()
		})

		Eventually(func() error {
			_, err := client.Get(baseURL)
			return err
		}, 5*time.Second, 100*time.Millisecond).Should(Succeed(), secondOutput.String())

		got := getJSON(client, baseURL+"/"+id)
		Expect(got["user"].(map[string]any)["name"]).To(Equal("Ada"), "data must survive the process restarting — this is what makes it real persistence, not the Phase 3 in-memory adapter")
	})
})

func postJSON(client *http.Client, url, body string) map[string]any {
	GinkgoHelper()
	resp, err := client.Post(url, "application/json", strings.NewReader(body))
	Expect(err).NotTo(HaveOccurred())
	return decodeJSON(resp)
}

func putJSON(client *http.Client, url, body string) map[string]any {
	GinkgoHelper()
	req, err := http.NewRequest(http.MethodPut, url, strings.NewReader(body))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	Expect(err).NotTo(HaveOccurred())
	return decodeJSON(resp)
}

func getJSON(client *http.Client, url string) map[string]any {
	GinkgoHelper()
	resp, err := client.Get(url)
	Expect(err).NotTo(HaveOccurred())
	return decodeJSON(resp)
}

func decodeJSON(resp *http.Response) map[string]any {
	GinkgoHelper()
	defer resp.Body.Close()

	var out map[string]any
	Expect(json.NewDecoder(resp.Body).Decode(&out)).To(Succeed())
	Expect(resp.StatusCode).To(Equal(http.StatusOK), fmt.Sprintf("body: %+v", out))
	return out
}
