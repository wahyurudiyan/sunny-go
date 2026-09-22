package memgen_test

import (
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/memgen"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

var _ = Describe("Generate", func() {
	var (
		root, destDir string
		fd            protoreflect.FileDescriptor
		file          *sgoproto.File
		p             core.Paths
	)

	BeforeEach(func() {
		var err error
		root, err = os.MkdirTemp("", "sgo-memgen-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		destDir = filepath.Join(root, "internal", "adapter", "out", "persistence", "memory")
		p = core.Paths{Module: "demo", Entity: "user"}

		protoDir := filepath.Join(root, "contract", "pb")
		Expect(os.MkdirAll(protoDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(protoDir, "user.proto"), []byte(memgenUserProto), 0644)).To(Succeed())

		fd, err = sgoproto.Compile(protoDir, "user.proto")
		Expect(err).NotTo(HaveOccurred())
		file, err = sgoproto.Build(fd)
		Expect(err).NotTo(HaveOccurred())
	})

	It("writes a repository implementing the fixed CRUD shape", func() {
		Expect(memgen.Generate(file, fd, p, destDir)).To(Succeed())

		content, err := os.ReadFile(filepath.Join(destDir, "user_repository_gen.go"))
		Expect(err).NotTo(HaveOccurred())

		Expect(string(content)).To(ContainSubstring("func NewUserRepository() *UserRepository"))
		Expect(string(content)).To(ContainSubstring("func (r *UserRepository) Create(ctx context.Context, entity *user.User) (*user.User, error)"))
		Expect(string(content)).To(ContainSubstring("func (r *UserRepository) List(ctx context.Context, page, pageSize int32) ([]*user.User, int32, error)"))
	})

	Describe("the generated List method's behavior", func() {
		// This exercises the actual generated code, not just its text —
		// it's the regression test for a real bug found while manually
		// verifying Phase 3 end to end: List(ctx, 0, 0) (the zero-value
		// request a "list everything" HTTP call produces, since Phase 3
		// doesn't parse pagination query params yet) used to return an
		// empty slice instead of everything, because pageSize<=0 was
		// treated as "return nothing" rather than "no limit".
		It("returns every record when called with page=0, pageSize=0", func() {
			moduleDir := buildTestModule(p)

			out := runInModule(moduleDir, `
				ctx := context.Background()
				repo := memory.NewUserRepository()
				repo.Create(ctx, &user.User{Name: "Ada"})
				repo.Create(ctx, &user.User{Name: "Grace"})

				items, total, err := repo.List(ctx, 0, 0)
				if err != nil {
					panic(err)
				}
				fmt.Printf("total=%d len=%d\n", total, len(items))
			`)

			Expect(out).To(ContainSubstring("total=2 len=2"))
		})

		It("still paginates correctly when a page size is given", func() {
			moduleDir := buildTestModule(p)

			out := runInModule(moduleDir, `
				ctx := context.Background()
				repo := memory.NewUserRepository()
				repo.Create(ctx, &user.User{Name: "Ada"})
				repo.Create(ctx, &user.User{Name: "Grace"})
				repo.Create(ctx, &user.User{Name: "Katherine"})

				items, total, err := repo.List(ctx, 0, 2)
				if err != nil {
					panic(err)
				}
				fmt.Printf("total=%d len=%d\n", total, len(items))
			`)

			Expect(out).To(ContainSubstring("total=3 len=2"))
		})
	})
})

var _ = Describe("Generate with repository_query methods", func() {
	const content = `syntax = "proto3";

package user.v1;

import "sgo/options.proto";

option go_package = "demo/contract/gen/user";

service UserService {
  rpc CreateUser(CreateUserRequest) returns (UserResponse);

  rpc FindUserByEmail(FindUserByEmailRequest) returns (UserResponse) {
    option (sgo.repository_query) = true;
  }

  rpc FindUserByNameAndEmail(FindUserByNameAndEmailRequest) returns (UserResponse) {
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

message FindUserByEmailRequest {
  string email = 1;
}

message FindUserByNameAndEmailRequest {
  string name = 1;
  string email = 2;
}

message UserResponse {
  User user = 1;
}
`

	var (
		root, protoDir, destDir string
		fd                      protoreflect.FileDescriptor
		file                    *sgoproto.File
		p                       core.Paths
	)

	BeforeEach(func() {
		var err error
		root, err = os.MkdirTemp("", "sgo-memgen-query-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		destDir = filepath.Join(root, "internal", "infrastructure", "persistence", "memory")
		p = core.Paths{Module: "demo", Entity: "user"}

		protoDir = filepath.Join(root, "contract", "pb")
		Expect(os.MkdirAll(protoDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(protoDir, "user.proto"), []byte(content), 0644)).To(Succeed())

		fd, err = sgoproto.Compile(protoDir, "user.proto")
		Expect(err).NotTo(HaveOccurred())
		file, err = sgoproto.Build(fd)
		Expect(err).NotTo(HaveOccurred())
	})

	It("auto-implements a single-scalar-parameter query as a linear scan in the generated file", func() {
		Expect(memgen.Generate(file, fd, p, destDir)).To(Succeed())

		content, err := os.ReadFile(filepath.Join(destDir, "user_repository_gen.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*user.User, error)"))
		Expect(string(content)).To(ContainSubstring("entity.Email == email"))
	})

	It("falls back to an owned stub for a query it can't confidently map", func() {
		Expect(memgen.Generate(file, fd, p, destDir)).To(Succeed())

		// Not in the generated file...
		genContent, err := os.ReadFile(filepath.Join(destDir, "user_repository_gen.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(genContent)).NotTo(ContainSubstring("FindByNameAndEmail"))

		// ...but stubbed in the owned companion file instead.
		ownedContent, err := os.ReadFile(filepath.Join(destDir, "user_repository.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(ownedContent)).To(ContainSubstring("func (r *UserRepository) FindByNameAndEmail(ctx context.Context, name string, email string) (*user.User, error)"))
		Expect(string(ownedContent)).To(ContainSubstring(`panic("sgo: TODO implement FindByNameAndEmail")`))
	})

	It("survives a hand-written owned-stub implementation across a second generate, and appends a newly added method without touching it", func() {
		Expect(memgen.Generate(file, fd, p, destDir)).To(Succeed())

		ownedPath := filepath.Join(destDir, "user_repository.go")
		handWritten := `package memory

import (
	"context"
	"fmt"

	user "demo/internal/domain/user"
)

func (r *UserRepository) FindByNameAndEmail(ctx context.Context, name string, email string) (*user.User, error) {
	return nil, fmt.Errorf("hand-written: %s/%s", name, email)
}
`
		Expect(os.WriteFile(ownedPath, []byte(handWritten), 0644)).To(Succeed())

		Expect(memgen.Generate(file, fd, p, destDir)).To(Succeed())

		after, err := os.ReadFile(ownedPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(after)).To(ContainSubstring("hand-written:"), "the hand-written body must survive regeneration untouched")
	})

	It("creates no owned companion file when every repository_query method is auto-implementable", func() {
		onlyAutoContent := `syntax = "proto3";

package user.v1;

import "sgo/options.proto";

option go_package = "demo/contract/gen/user";

service UserService {
  rpc CreateUser(CreateUserRequest) returns (UserResponse);

  rpc FindUserByEmail(FindUserByEmailRequest) returns (UserResponse) {
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

message FindUserByEmailRequest {
  string email = 1;
}

message UserResponse {
  User user = 1;
}
`
		Expect(os.WriteFile(filepath.Join(protoDir, "user.proto"), []byte(onlyAutoContent), 0644)).To(Succeed())
		var err error
		fd, err = sgoproto.Compile(protoDir, "user.proto")
		Expect(err).NotTo(HaveOccurred())
		file, err = sgoproto.Build(fd)
		Expect(err).NotTo(HaveOccurred())

		Expect(memgen.Generate(file, fd, p, destDir)).To(Succeed())

		_, err = os.Stat(filepath.Join(destDir, "user_repository.go"))
		Expect(os.IsNotExist(err)).To(BeTrue(), "no owned file should be created when nothing needs stubbing")
	})
})

const memgenUserProto = `syntax = "proto3";

package user.v1;

option go_package = "demo/contract/gen/user";

message User {
  string id = 1;
  string name = 2;
}
`

// buildTestModule assembles a throwaway Go module with p's aggregate
// domain package, its event kernel, and the memory repository generated
// into it, so the generated List method can actually be executed rather
// than just pattern-matched as text.
func buildTestModule(p core.Paths) string {
	GinkgoHelper()

	dir, err := os.MkdirTemp("", "sgo-memgen-module-*")
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })

	Expect(os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module demo\n\ngo 1.22\n"), 0644)).To(Succeed())

	protoDir := filepath.Join(dir, "contract", "pb")
	Expect(os.MkdirAll(protoDir, 0755)).To(Succeed())
	Expect(os.WriteFile(filepath.Join(protoDir, "user.proto"), []byte(memgenUserProto), 0644)).To(Succeed())

	fd, err := sgoproto.Compile(protoDir, "user.proto")
	Expect(err).NotTo(HaveOccurred())
	file, err := sgoproto.Build(fd)
	Expect(err).NotTo(HaveOccurred())

	eventDir := filepath.Join(dir, "internal", "domain", "event")
	Expect(core.GenerateEventKernel(eventDir)).To(Succeed())

	domainDir := filepath.Join(dir, "internal", "domain", "user")
	Expect(core.GenerateAggregate(file, fd, p, domainDir)).To(Succeed())

	memoryDir := filepath.Join(dir, "internal", "infrastructure", "persistence", "memory")
	Expect(memgen.Generate(file, fd, p, memoryDir)).To(Succeed())

	return dir
}

// runInModule writes body into a main() in moduleDir and runs it,
// returning combined stdout+stderr. body can reference the "user" and
// "memory" packages (already imported) plus "context" and "fmt".
func runInModule(moduleDir, body string) string {
	GinkgoHelper()

	main := "package main\n\nimport (\n\t\"context\"\n\t\"fmt\"\n\n\tuser \"demo/internal/domain/user\"\n\tmemory \"demo/internal/infrastructure/persistence/memory\"\n)\n\nfunc main() {\n" + body + "\n}\n"

	mainDir := filepath.Join(moduleDir, "cmd", "harness")
	Expect(os.MkdirAll(mainDir, 0755)).To(Succeed())
	Expect(os.WriteFile(filepath.Join(mainDir, "main.go"), []byte(main), 0644)).To(Succeed())

	cmd := exec.Command("go", "run", "./cmd/harness")
	cmd.Dir = moduleDir
	out, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), string(out))

	return string(out)
}
