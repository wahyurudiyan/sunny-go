package mongogen_test

import (
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/mongogen"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

var _ = Describe("Generate", func() {
	var (
		root, protoDir, destDir string
		fd                      protoreflect.FileDescriptor
		file                    *sgoproto.File
		p                       core.Paths
	)

	BeforeEach(func() {
		var err error
		root, err = os.MkdirTemp("", "sgo-mongogen-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		protoDir = filepath.Join(root, "contract", "pb")
		Expect(sgoproto.GenerateStub(protoDir, "user", "demo")).To(Succeed())
		fd, err = sgoproto.Compile(protoDir, "user.proto")
		Expect(err).NotTo(HaveOccurred())
		file, err = sgoproto.Build(fd)
		Expect(err).NotTo(HaveOccurred())

		p = core.Paths{Module: "demo", Entity: "user"}
		destDir = filepath.Join(root, "internal", "infrastructure", "persistence", "mongo")
	})

	It("writes a repository using bson tags and the official mongo-driver API", func() {
		Expect(mongogen.Generate(file, fd, p, destDir)).To(Succeed())

		content, err := os.ReadFile(filepath.Join(destDir, "user_repository_gen.go"))
		Expect(err).NotTo(HaveOccurred())

		Expect(string(content)).To(ContainSubstring(`bson:"name"`))
		Expect(string(content)).To(ContainSubstring("func NewUserRepository(db *mongodriver.Database) *UserRepository"))
		Expect(string(content)).To(ContainSubstring(`r.collection.FindOne(ctx, bson.M{"_id": id})`))
	})

	// No local MongoDB is available to run this against for real (unlike
	// sqlgen's Postgres and cachegen's Redis specs) — this is a
	// compile-only check: does the generated code actually type-check
	// against the real mongo-driver API, not just look plausible as text.
	It("produces a module that type-checks against the real mongo-driver API", func() {
		Expect(mongogen.Generate(file, fd, p, destDir)).To(Succeed())

		domainDir := filepath.Join(root, "internal", "domain", "user")
		Expect(core.GenerateEventKernel(filepath.Join(root, "internal", "domain", "event"))).To(Succeed())
		Expect(core.GenerateAggregate(file, fd, p, domainDir)).To(Succeed())

		Expect(os.WriteFile(filepath.Join(root, "go.mod"), []byte("module demo\n\ngo 1.22\n"), 0644)).To(Succeed())

		tidy := exec.Command("go", "mod", "tidy")
		tidy.Dir = root
		out, err := tidy.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(out))

		build := exec.Command("go", "build", "./...")
		build.Dir = root
		out, err = build.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(out))
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

	var (
		root, destDir string
		fd            protoreflect.FileDescriptor
		file          *sgoproto.File
		p             core.Paths
	)

	BeforeEach(func() {
		var err error
		root, err = os.MkdirTemp("", "sgo-mongogen-query-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		protoDir := filepath.Join(root, "contract", "pb")
		Expect(os.MkdirAll(protoDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(protoDir, "user.proto"), []byte(content), 0644)).To(Succeed())

		fd, err = sgoproto.Compile(protoDir, "user.proto")
		Expect(err).NotTo(HaveOccurred())
		file, err = sgoproto.Build(fd)
		Expect(err).NotTo(HaveOccurred())

		p = core.Paths{Module: "demo", Entity: "user"}
		destDir = filepath.Join(root, "internal", "infrastructure", "persistence", "mongo")
	})

	It("writes an owned companion file with a panic stub, since Mongo query logic can't be auto-generated", func() {
		Expect(mongogen.Generate(file, fd, p, destDir)).To(Succeed())

		genContent, err := os.ReadFile(filepath.Join(destDir, "user_repository_gen.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(genContent)).NotTo(ContainSubstring("FindByEmail"), "mongogen never auto-implements — that's memgen-only")

		ownedContent, err := os.ReadFile(filepath.Join(destDir, "user_repository.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(ownedContent)).To(ContainSubstring("func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*user.User, error)"))
		Expect(string(ownedContent)).To(ContainSubstring(`panic("sgo: TODO implement FindByEmail")`))
	})

	It("survives a hand-written owned-stub implementation across a second generate", func() {
		Expect(mongogen.Generate(file, fd, p, destDir)).To(Succeed())

		ownedPath := filepath.Join(destDir, "user_repository.go")
		handWritten := `package mongo

import (
	"context"
	"fmt"

	user "demo/internal/domain/user"
)

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	return nil, fmt.Errorf("hand-written: %s", email)
}
`
		Expect(os.WriteFile(ownedPath, []byte(handWritten), 0644)).To(Succeed())

		Expect(mongogen.Generate(file, fd, p, destDir)).To(Succeed())

		after, err := os.ReadFile(ownedPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(after)).To(ContainSubstring("hand-written:"), "the hand-written body must survive regeneration untouched")
	})
})
