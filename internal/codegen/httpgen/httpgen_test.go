package httpgen_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/httpgen"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
	"github.com/wahyurudiyan/sunny-go/internal/config"
)

var _ = Describe("Generate", func() {
	var (
		root, protoDir, destDir string
		file                    *sgoproto.File
		p                       core.Paths
	)

	BeforeEach(func() {
		var err error
		root, err = os.MkdirTemp("", "sgo-httpgen-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		protoDir = filepath.Join(root, "contract", "pb")
		file = userFile(protoDir)
		p = core.Paths{Module: "demo", Entity: "user"}
	})

	Context("gin", func() {
		BeforeEach(func() {
			destDir = filepath.Join(root, "internal", "infrastructure", "transport", "http", "gin")
			Expect(httpgen.GenerateServer(config.HTTPFrameworkGin, destDir)).To(Succeed())
			Expect(httpgen.GenerateRoutes(config.HTTPFrameworkGin, file, p, destDir)).To(Succeed())
		})

		It("writes a server wrapping gin.Engine", func() {
			content, err := os.ReadFile(filepath.Join(destDir, "server_gen.go"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("*gin.Engine"))
			Expect(string(content)).To(ContainSubstring("func (s *Server) Start(addr string) error"))
		})

		It("writes route registrations calling the application service directly", func() {
			content, err := os.ReadFile(filepath.Join(destDir, "user_routes_gen.go"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("func RegisterUserRoutes(engine *gin.Engine, svc *app.UserService)"))
			Expect(string(content)).To(ContainSubstring(`engine.GET("/api/v1/users/:id"`))
			Expect(string(content)).To(ContainSubstring(`engine.POST("/api/v1/users"`))
			Expect(string(content)).To(ContainSubstring(`gin.H{"user": resp}`))
			Expect(string(content)).To(ContainSubstring(`gin.H{"success": true}`))
		})
	})

	Context("echo", func() {
		BeforeEach(func() {
			destDir = filepath.Join(root, "internal", "infrastructure", "transport", "http", "echo")
			Expect(httpgen.GenerateServer(config.HTTPFrameworkEcho, destDir)).To(Succeed())
			Expect(httpgen.GenerateRoutes(config.HTTPFrameworkEcho, file, p, destDir)).To(Succeed())
		})

		It("writes a server wrapping echo.Echo", func() {
			content, err := os.ReadFile(filepath.Join(destDir, "server_gen.go"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("*echo.Echo"))
		})

		It("writes route registrations calling the application service directly", func() {
			content, err := os.ReadFile(filepath.Join(destDir, "user_routes_gen.go"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("func RegisterUserRoutes(e *echo.Echo, svc *app.UserService)"))
			Expect(string(content)).To(ContainSubstring(`e.GET("/api/v1/users/:id"`))
		})
	})

	Context("chi", func() {
		BeforeEach(func() {
			destDir = filepath.Join(root, "internal", "infrastructure", "transport", "http", "chi")
			Expect(httpgen.GenerateServer(config.HTTPFrameworkChi, destDir)).To(Succeed())
			Expect(httpgen.GenerateRoutes(config.HTTPFrameworkChi, file, p, destDir)).To(Succeed())
		})

		It("writes a server wrapping chi.Router", func() {
			content, err := os.ReadFile(filepath.Join(destDir, "server_gen.go"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("chi.Router"))
		})

		It("writes route registrations with chi's {id} path syntax", func() {
			content, err := os.ReadFile(filepath.Join(destDir, "user_routes_gen.go"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("func RegisterUserRoutes(r chi.Router, svc *app.UserService)"))
			Expect(string(content)).To(ContainSubstring(`r.Get("/api/v1/users/{id}"`))
			Expect(string(content)).To(ContainSubstring("chi.URLParam(req, \"id\")"))
		})
	})

	It("rejects an unsupported framework", func() {
		err := httpgen.GenerateServer("fiber", filepath.Join(root, "x"))
		Expect(err).To(MatchError(ContainSubstring("unsupported HTTP framework")))
	})
})
