package memgen_test

import (
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/memgen"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

var _ = Describe("Generate", func() {
	var (
		root, destDir string
		p             core.Paths
	)

	BeforeEach(func() {
		var err error
		root, err = os.MkdirTemp("", "sgo-memgen-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		destDir = filepath.Join(root, "internal", "adapter", "out", "persistence", "memory")
		p = core.Paths{Module: "demo", Entity: "user"}
	})

	It("writes a repository implementing the fixed CRUD shape", func() {
		Expect(memgen.Generate(p, destDir)).To(Succeed())

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

// buildTestModule assembles a throwaway Go module with p's domain
// package and memory repository generated into it, so the generated
// List method can actually be executed rather than just pattern-matched
// as text.
func buildTestModule(p core.Paths) string {
	GinkgoHelper()

	dir, err := os.MkdirTemp("", "sgo-memgen-module-*")
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })

	Expect(os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module demo\n\ngo 1.22\n"), 0644)).To(Succeed())

	file := &sgoproto.File{
		Messages: []sgoproto.Message{{
			Name: "User",
			Fields: []sgoproto.Field{
				{Name: "id", GoName: "Id", Kind: sgoproto.KindString},
				{Name: "name", GoName: "Name", Kind: sgoproto.KindString},
			},
		}},
	}

	domainDir := filepath.Join(dir, "internal", "core", "domain", "user")
	Expect(core.GenerateDomain(file, p, domainDir)).To(Succeed())

	memoryDir := filepath.Join(dir, "internal", "adapter", "out", "persistence", "memory")
	Expect(memgen.Generate(p, memoryDir)).To(Succeed())

	return dir
}

// runInModule writes body into a main() in moduleDir and runs it,
// returning combined stdout+stderr. body can reference the "user" and
// "memory" packages (already imported) plus "context" and "fmt".
func runInModule(moduleDir, body string) string {
	GinkgoHelper()

	main := "package main\n\nimport (\n\t\"context\"\n\t\"fmt\"\n\n\tuser \"demo/internal/core/domain/user\"\n\tmemory \"demo/internal/adapter/out/persistence/memory\"\n)\n\nfunc main() {\n" + body + "\n}\n"

	mainDir := filepath.Join(moduleDir, "cmd", "harness")
	Expect(os.MkdirAll(mainDir, 0755)).To(Succeed())
	Expect(os.WriteFile(filepath.Join(mainDir, "main.go"), []byte(main), 0644)).To(Succeed())

	cmd := exec.Command("go", "run", "./cmd/harness")
	cmd.Dir = moduleDir
	out, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), string(out))

	return string(out)
}
