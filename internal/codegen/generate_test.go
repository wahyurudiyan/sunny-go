package codegen_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/codegen"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/project"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
	"github.com/wahyurudiyan/sunny-go/internal/config"
)

// scaffoldProject builds a real project (Phase 1's scaffolder) plus a
// starter user.proto, so these tests exercise `sgo init` +
// `sgo generate proto` + `sgo generate code` together end to end.
func scaffoldProject(root string) string {
	GinkgoHelper()

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
