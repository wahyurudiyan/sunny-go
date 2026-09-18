package bootstrap_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/bootstrap"
	"github.com/wahyurudiyan/sunny-go/internal/config"
)

var _ = Describe("Generate", func() {
	var (
		root, destDir string
	)

	BeforeEach(func() {
		var err error
		root, err = os.MkdirTemp("", "sgo-bootstrap-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		destDir = filepath.Join(root, "internal", "bootstrap")
	})

	Context("with no services yet (fresh `sgo init`)", func() {
		It("still writes a valid, buildable Run that starts empty servers", func() {
			Expect(bootstrap.Generate("demo", config.HTTPFrameworkGin, nil, destDir)).To(Succeed())

			content, err := os.ReadFile(filepath.Join(destDir, "wire_gen.go"))
			Expect(err).NotTo(HaveOccurred())

			Expect(string(content)).NotTo(ContainSubstring("grpcadapter"), "no per-entity imports needed with zero services")
			Expect(string(content)).NotTo(ContainSubstring(`memory "demo`))
			Expect(string(content)).To(ContainSubstring("func Run(ctx context.Context) error"))

			assertValidGo(filepath.Join(destDir, "wire_gen.go"))
		})
	})

	Context("with services on record", func() {
		It("wires every service, not just one", func() {
			Expect(bootstrap.Generate("demo", config.HTTPFrameworkGin, []string{"user", "order"}, destDir)).To(Succeed())

			content, err := os.ReadFile(filepath.Join(destDir, "wire_gen.go"))
			Expect(err).NotTo(HaveOccurred())

			for _, entity := range []string{"User", "Order"} {
				Expect(string(content)).To(ContainSubstring("memory.New" + entity + "Repository()"))
				Expect(string(content)).To(ContainSubstring("service.New" + entity + "Service("))
				Expect(string(content)).To(ContainSubstring("httpadapter.Register" + entity + "Routes("))
				Expect(string(content)).To(ContainSubstring("grpcadapter.Register" + entity + "ServiceServer("))
			}

			assertValidGo(filepath.Join(destDir, "wire_gen.go"))
		})
	})

	Context("per HTTP framework", func() {
		It("uses the gin server's Engine field", func() {
			Expect(bootstrap.Generate("demo", config.HTTPFrameworkGin, []string{"user"}, destDir)).To(Succeed())
			content, err := os.ReadFile(filepath.Join(destDir, "wire_gen.go"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("httpServer.Engine"))
		})

		It("uses the echo server's Echo field", func() {
			Expect(bootstrap.Generate("demo", config.HTTPFrameworkEcho, []string{"user"}, destDir)).To(Succeed())
			content, err := os.ReadFile(filepath.Join(destDir, "wire_gen.go"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("httpServer.Echo"))
		})

		It("uses the chi server's Router field", func() {
			Expect(bootstrap.Generate("demo", config.HTTPFrameworkChi, []string{"user"}, destDir)).To(Succeed())
			content, err := os.ReadFile(filepath.Join(destDir, "wire_gen.go"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("httpServer.Router"))
		})
	})
})

func assertValidGo(path string) {
	GinkgoHelper()

	src, err := os.ReadFile(path)
	Expect(err).NotTo(HaveOccurred())

	_, err = parser.ParseFile(token.NewFileSet(), path, src, parser.AllErrors)
	Expect(err).NotTo(HaveOccurred())
}
