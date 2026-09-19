package searchgen_test

import (
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/searchgen"
)

var _ = Describe("ImportPath", func() {
	It("joins the module path with the Elasticsearch adapter's package location", func() {
		Expect(searchgen.ImportPath("demo")).To(Equal("demo/internal/adapter/out/search/elasticsearch"))
	})
})

var _ = Describe("Generate", func() {
	var root string

	BeforeEach(func() {
		var err error
		root, err = os.MkdirTemp("", "sgo-searchgen-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })
	})

	It("writes the Search port", func() {
		portDir := filepath.Join(root, "internal", "core", "port", "out")
		Expect(searchgen.GeneratePort(portDir)).To(Succeed())

		content, err := os.ReadFile(filepath.Join(portDir, "search.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("type Search interface"))
		Expect(string(content)).To(ContainSubstring("type SearchResult struct"))
	})

	It("writes the Elasticsearch adapter", func() {
		destDir := filepath.Join(root, "internal", "adapter", "out", "search", "elasticsearch")
		Expect(searchgen.GenerateElasticsearch("demo", destDir)).To(Succeed())

		content, err := os.ReadFile(filepath.Join(destDir, "search_gen.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("github.com/elastic/go-elasticsearch/v8"))
		Expect(string(content)).To(ContainSubstring("func (s *Search) Index(ctx context.Context, id string, document map[string]any) error"))
	})

	// No local Elasticsearch is available to run this against for real
	// (unlike sqlgen's Postgres and cachegen's Redis specs, and unlike
	// Docker, which also isn't available in this environment) — this is
	// a compile-only check: does the generated code actually type-check
	// against the real go-elasticsearch client and esapi package.
	It("produces a module that type-checks against the real go-elasticsearch API", func() {
		portDir := filepath.Join(root, "internal", "core", "port", "out")
		Expect(searchgen.GeneratePort(portDir)).To(Succeed())

		destDir := filepath.Join(root, "internal", "adapter", "out", "search", "elasticsearch")
		Expect(searchgen.GenerateElasticsearch("demo", destDir)).To(Succeed())

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
