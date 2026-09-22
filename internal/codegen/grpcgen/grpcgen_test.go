package grpcgen_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/grpcgen"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

var _ = Describe("Generate", func() {
	var (
		root, protoDir, destDir string
	)

	BeforeEach(func() {
		var err error
		root, err = os.MkdirTemp("", "sgo-grpcgen-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		protoDir = filepath.Join(root, "contract", "pb")
		Expect(sgoproto.GenerateStub(protoDir, "user", "demo")).To(Succeed())

		destDir = filepath.Join(root, "internal", "infrastructure", "transport", "grpc")
	})

	It("writes a server embedding UnimplementedUserServiceServer by value and delegating to the application service", func() {
		fd, err := sgoproto.Compile(protoDir, "user.proto")
		Expect(err).NotTo(HaveOccurred())
		file, err := sgoproto.Build(fd)
		Expect(err).NotTo(HaveOccurred())

		p := core.Paths{Module: "demo", Entity: "user"}
		Expect(grpcgen.Generate(file, p, destDir)).To(Succeed())

		content, err := os.ReadFile(filepath.Join(destDir, "user_grpc_server_gen.go"))
		Expect(err).NotTo(HaveOccurred())

		Expect(string(content)).To(ContainSubstring("wire.UnimplementedUserServiceServer"))
		Expect(string(content)).To(ContainSubstring("func RegisterUserServiceServer(s grpc.ServiceRegistrar, svc *app.UserService)"))
		Expect(string(content)).To(ContainSubstring("func (h *userServiceServer) CreateUser(ctx context.Context, req *wire.CreateUserRequest) (*wire.UserResponse, error)"))
		Expect(string(content)).To(ContainSubstring("mapper.CreateUserRequestToApp(req)"))
		Expect(string(content)).To(ContainSubstring("mapper.UserFromDomain(resp)"))
		Expect(string(content)).To(ContainSubstring("User: mapper.UserFromDomain(resp)"))
	})

	It("errors clearly when the proto declares no service", func() {
		serviceless := &sgoproto.File{Path: "empty.proto"}

		p := core.Paths{Module: "demo", Entity: "user"}
		err := grpcgen.Generate(serviceless, p, destDir)

		Expect(err).To(MatchError(ContainSubstring("declares no service")))
	})
})
