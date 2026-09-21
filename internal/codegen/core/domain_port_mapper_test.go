package core_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

// userFile compiles the real `sgo generate proto user` starter template
// and builds its IR, so these tests exercise the actual generator
// pipeline rather than hand-built fixtures that could drift from it.
func userFile(protoDir string) *sgoproto.File {
	GinkgoHelper()

	Expect(sgoproto.GenerateStub(protoDir, "user", "demo")).To(Succeed())

	fd, err := sgoproto.Compile(protoDir, "user.proto")
	Expect(err).NotTo(HaveOccurred())

	file, err := sgoproto.Build(fd)
	Expect(err).NotTo(HaveOccurred())

	return file
}

var _ = Describe("GenerateDomain", func() {
	var (
		root, protoDir, domainDir string
		file                      *sgoproto.File
		p                         core.Paths
	)

	BeforeEach(func() {
		var err error
		root, err = os.MkdirTemp("", "sgo-core-domain-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		protoDir = filepath.Join(root, "contract", "pb")
		Expect(os.MkdirAll(protoDir, 0755)).To(Succeed())
		domainDir = filepath.Join(root, "internal", "core", "domain", "user")

		file = userFile(protoDir)
		p = core.Paths{Module: "demo", Entity: "user"}
	})

	It("writes structs for every message into user_gen.go", func() {
		Expect(core.GenerateDomain(file, p, domainDir)).To(Succeed())

		content, err := os.ReadFile(filepath.Join(domainDir, "user_gen.go"))
		Expect(err).NotTo(HaveOccurred())

		Expect(string(content)).To(ContainSubstring("type User struct"))
		Expect(string(content)).To(ContainSubstring("Id string"))
		Expect(string(content)).To(ContainSubstring("type CreateUserRequest struct"))
		Expect(string(content)).To(ContainSubstring("type ListUsersResponse struct"))
	})

	It("creates the owned user.go once and never overwrites it", func() {
		Expect(core.GenerateDomain(file, p, domainDir)).To(Succeed())

		ownedPath := filepath.Join(domainDir, "user.go")
		Expect(os.WriteFile(ownedPath, []byte("package user\n\n// hand-written invariant\nfunc (u *User) Valid() bool { return u.Id != \"\" }\n"), 0644)).To(Succeed())

		Expect(core.GenerateDomain(file, p, domainDir)).To(Succeed())

		content, err := os.ReadFile(ownedPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("hand-written invariant"))
	})
})

var _ = Describe("GenerateUsecasePort and GenerateRepositoryPort", func() {
	var (
		root, protoDir string
		file           *sgoproto.File
		p              core.Paths
	)

	BeforeEach(func() {
		var err error
		root, err = os.MkdirTemp("", "sgo-core-port-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		protoDir = filepath.Join(root, "contract", "pb")
		Expect(os.MkdirAll(protoDir, 0755)).To(Succeed())

		file = userFile(protoDir)
		p = core.Paths{Module: "demo", Entity: "user"}
	})

	It("generates a usecase interface with one method per RPC", func() {
		dir := filepath.Join(root, "internal", "core", "port", "in")
		Expect(core.GenerateUsecasePort(file, p, dir)).To(Succeed())

		content, err := os.ReadFile(filepath.Join(dir, "user_usecase.go"))
		Expect(err).NotTo(HaveOccurred())

		Expect(string(content)).To(ContainSubstring("type UserUseCase interface"))
		Expect(string(content)).To(ContainSubstring("CreateUser(ctx context.Context, req *user.CreateUserRequest) (*user.UserResponse, error)"))
		Expect(string(content)).To(ContainSubstring("DeleteUser(ctx context.Context, req *user.DeleteUserRequest) (*user.DeleteUserResponse, error)"))
	})

	It("generates a fixed CRUD repository interface", func() {
		dir := filepath.Join(root, "internal", "core", "port", "out")
		Expect(core.GenerateRepositoryPort(p, dir)).To(Succeed())

		content, err := os.ReadFile(filepath.Join(dir, "user_repository.go"))
		Expect(err).NotTo(HaveOccurred())

		Expect(string(content)).To(ContainSubstring("type UserRepository interface"))
		Expect(string(content)).To(ContainSubstring("Create(ctx context.Context, entity *user.User) (*user.User, error)"))
		Expect(string(content)).To(ContainSubstring("Delete(ctx context.Context, id string) error"))
	})
})

var _ = Describe("GenerateMapper", func() {
	var (
		root, protoDir, mapperDir string
		file                      *sgoproto.File
		p                         core.Paths
	)

	BeforeEach(func() {
		var err error
		root, err = os.MkdirTemp("", "sgo-core-mapper-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		protoDir = filepath.Join(root, "contract", "pb")
		Expect(os.MkdirAll(protoDir, 0755)).To(Succeed())
		mapperDir = filepath.Join(root, "internal", "adapter", "mapper")

		file = userFile(protoDir)
		p = core.Paths{Module: "demo", Entity: "user"}
	})

	It("generates ToDomain/FromDomain for every message", func() {
		Expect(core.GenerateMapper(file, p, mapperDir)).To(Succeed())

		content, err := os.ReadFile(filepath.Join(mapperDir, "user_mapper_gen.go"))
		Expect(err).NotTo(HaveOccurred())

		Expect(string(content)).To(ContainSubstring("func UserToDomain(w *wire.User) *domain.User"))
		Expect(string(content)).To(ContainSubstring("func UserFromDomain(d *domain.User) *wire.User"))
	})

	It("maps a repeated message field with a loop, not a direct assignment", func() {
		Expect(core.GenerateMapper(file, p, mapperDir)).To(Succeed())

		content, err := os.ReadFile(filepath.Join(mapperDir, "user_mapper_gen.go"))
		Expect(err).NotTo(HaveOccurred())

		Expect(string(content)).To(ContainSubstring("func ListUsersResponseToDomain"))
		Expect(string(content)).To(ContainSubstring("for i, v := range w.GetUsers()"))
		Expect(string(content)).To(ContainSubstring("UserToDomain(v)"))
	})
})
