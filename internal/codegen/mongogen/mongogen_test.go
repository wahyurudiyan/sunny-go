package mongogen_test

import (
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/mongogen"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

var _ = Describe("Generate", func() {
	var (
		root, protoDir, destDir string
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
		fd, err := sgoproto.Compile(protoDir, "user.proto")
		Expect(err).NotTo(HaveOccurred())
		file, err = sgoproto.Build(fd)
		Expect(err).NotTo(HaveOccurred())

		p = core.Paths{Module: "demo", Entity: "user"}
		destDir = filepath.Join(root, "internal", "infrastructure", "persistence", "mongo")
	})

	It("writes a repository using bson tags and the official mongo-driver API", func() {
		Expect(mongogen.Generate(file, p, destDir)).To(Succeed())

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
		Expect(mongogen.Generate(file, p, destDir)).To(Succeed())

		fd, err := sgoproto.Compile(protoDir, "user.proto")
		Expect(err).NotTo(HaveOccurred())

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
