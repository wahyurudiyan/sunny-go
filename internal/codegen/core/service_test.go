package core_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

var _ = Describe("safe regeneration", func() {
	var (
		root, protoDir, serviceDir, servicePath string
		file                                    *sgoproto.File
		p                                       core.Paths
	)

	BeforeEach(func() {
		var err error
		root, err = os.MkdirTemp("", "sgo-core-service-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		protoDir = filepath.Join(root, "contract", "pb")
		Expect(os.MkdirAll(protoDir, 0755)).To(Succeed())
		serviceDir = filepath.Join(root, "internal", "core", "service")
		servicePath = filepath.Join(serviceDir, "user_service.go")

		file = userFile(protoDir)
		p = core.Paths{Module: "demo", Entity: "user"}
	})

	Context("on first generation", func() {
		It("creates the service file with a panic-stub for every usecase method", func() {
			Expect(core.GenerateService(file, p, serviceDir)).To(Succeed())

			content, err := os.ReadFile(servicePath)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(content)).To(ContainSubstring("type UserService struct"))
			Expect(string(content)).To(ContainSubstring("func NewUserService(repo out.UserRepository) *UserService"))
			for _, method := range []string{"CreateUser", "GetUser", "ListUsers", "UpdateUser", "DeleteUser"} {
				Expect(string(content)).To(ContainSubstring("func (s *UserService) " + method))
				Expect(string(content)).To(ContainSubstring(`panic("sgo: TODO implement ` + method + `")`))
			}
		})
	})

	Context("when regenerated with no changes to the proto", func() {
		It("preserves a hand-written method body across a second generate run", func() {
			Expect(core.GenerateService(file, p, serviceDir)).To(Succeed())

			implementBusinessLogic(servicePath, "GetUser", `return s.repo.Get(ctx, req.Id)`)

			Expect(core.GenerateService(file, p, serviceDir)).To(Succeed())

			content, err := os.ReadFile(servicePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("return s.repo.Get(ctx, req.Id)"))
			Expect(string(content)).NotTo(ContainSubstring(`panic("sgo: TODO implement GetUser")`))
		})

		It("does not touch the constructor or struct a developer may have customized", func() {
			Expect(core.GenerateService(file, p, serviceDir)).To(Succeed())

			content, err := os.ReadFile(servicePath)
			Expect(err).NotTo(HaveOccurred())
			customized := strings.Replace(string(content),
				"type UserService struct {\n\trepo out.UserRepository\n}",
				"type UserService struct {\n\trepo   out.UserRepository\n\tlogger string // hand-added dependency\n}",
				1)
			Expect(customized).To(ContainSubstring("hand-added dependency"))
			Expect(os.WriteFile(servicePath, []byte(customized), 0644)).To(Succeed())

			Expect(core.GenerateService(file, p, serviceDir)).To(Succeed())

			content, err = os.ReadFile(servicePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("hand-added dependency"))
		})
	})

	Context("when the proto gains a new RPC", func() {
		It("appends a stub for the new method and preserves the hand-written body of an existing one", func() {
			Expect(core.GenerateService(file, p, serviceDir)).To(Succeed())
			implementBusinessLogic(servicePath, "GetUser", `return s.repo.Get(ctx, req.Id)`)

			grown := withMethod(file, sgoproto.Method{Name: "ArchiveUser", Input: "DeleteUserRequest", Output: "DeleteUserResponse"})
			Expect(core.GenerateService(grown, p, serviceDir)).To(Succeed())

			content, err := os.ReadFile(servicePath)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(content)).To(ContainSubstring("return s.repo.Get(ctx, req.Id)"), "existing business logic must survive")
			Expect(string(content)).To(ContainSubstring("func (s *UserService) ArchiveUser"), "new method must be appended")
			Expect(string(content)).To(ContainSubstring(`panic("sgo: TODO implement ArchiveUser")`))

			assertValidGo(servicePath)
		})
	})

	Context("when the proto loses an RPC that was already implemented", func() {
		It("leaves the implementation in place with a warning comment, never deletes it", func() {
			Expect(core.GenerateService(file, p, serviceDir)).To(Succeed())
			implementBusinessLogic(servicePath, "DeleteUser", `return s.repo.Delete(ctx, req.Id)`)

			shrunk := withoutMethod(file, "DeleteUser")
			Expect(core.GenerateService(shrunk, p, serviceDir)).To(Succeed())

			content, err := os.ReadFile(servicePath)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(content)).To(ContainSubstring("return s.repo.Delete(ctx, req.Id)"), "orphaned implementation must survive")
			Expect(string(content)).To(ContainSubstring("// sgo: DeleteUser is no longer part of UserUseCase; remove if unused"))

			assertValidGo(servicePath)
		})

		It("doesn't duplicate the warning comment on a third regeneration", func() {
			Expect(core.GenerateService(file, p, serviceDir)).To(Succeed())
			implementBusinessLogic(servicePath, "DeleteUser", `return s.repo.Delete(ctx, req.Id)`)

			shrunk := withoutMethod(file, "DeleteUser")
			Expect(core.GenerateService(shrunk, p, serviceDir)).To(Succeed())
			Expect(core.GenerateService(shrunk, p, serviceDir)).To(Succeed())

			content, err := os.ReadFile(servicePath)
			Expect(err).NotTo(HaveOccurred())

			Expect(strings.Count(string(content), "// sgo: DeleteUser is no longer part of UserUseCase; remove if unused")).To(Equal(1))
		})
	})
})

// implementBusinessLogic replaces the named method's panic-stub body
// with body, simulating a developer implementing it.
func implementBusinessLogic(path, method, body string) {
	GinkgoHelper()

	content, err := os.ReadFile(path)
	Expect(err).NotTo(HaveOccurred())

	stub := `panic("sgo: TODO implement ` + method + `")`
	Expect(string(content)).To(ContainSubstring(stub))

	updated := strings.Replace(string(content), stub, body, 1)
	Expect(os.WriteFile(path, []byte(updated), 0644)).To(Succeed())
}

// withMethod returns a copy of f's IR with an extra method appended to
// its (only) service, simulating the proto gaining an RPC.
func withMethod(f *sgoproto.File, m sgoproto.Method) *sgoproto.File {
	clone := *f
	svc := f.Services[0]
	svc.Methods = append(append([]sgoproto.Method{}, svc.Methods...), m)
	clone.Services = []sgoproto.Service{svc}
	return &clone
}

// withoutMethod returns a copy of f's IR with the named method removed
// from its (only) service, simulating the proto losing an RPC.
func withoutMethod(f *sgoproto.File, name string) *sgoproto.File {
	clone := *f
	svc := f.Services[0]
	var kept []sgoproto.Method
	for _, m := range svc.Methods {
		if m.Name != name {
			kept = append(kept, m)
		}
	}
	svc.Methods = kept
	clone.Services = []sgoproto.Service{svc}
	return &clone
}

func assertValidGo(path string) {
	GinkgoHelper()

	src, err := os.ReadFile(path)
	Expect(err).NotTo(HaveOccurred())

	_, err = parser.ParseFile(token.NewFileSet(), path, src, parser.AllErrors)
	Expect(err).NotTo(HaveOccurred(), "regenerated file must still be valid Go")
}
