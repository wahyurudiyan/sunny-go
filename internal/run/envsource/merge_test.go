package envsource_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/run/envsource"
)

var _ = Describe("Merge", func() {
	var ctx context.Context

	BeforeEach(func() { ctx = context.Background() })

	It("includes every source's fetched values", func() {
		dir, err := os.MkdirTemp("", "sgo-envsource-merge-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })

		path := filepath.Join(dir, ".env")
		Expect(os.WriteFile(path, []byte("SGO_MERGE_TEST_KEY=from-dotenv\n"), 0644)).To(Succeed())

		env, err := envsource.Merge(ctx, []envsource.Source{&envsource.DotEnvSource{Path: path}})
		Expect(err).NotTo(HaveOccurred())
		Expect(env).To(ContainElement("SGO_MERGE_TEST_KEY=from-dotenv"))
	})

	It("lets a real environment variable win over the same key from a source", func() {
		Expect(os.Setenv("SGO_MERGE_TEST_KEY", "from-os-env")).To(Succeed())
		DeferCleanup(func() { Expect(os.Unsetenv("SGO_MERGE_TEST_KEY")).To(Succeed()) })

		dir, err := os.MkdirTemp("", "sgo-envsource-merge-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })

		path := filepath.Join(dir, ".env")
		Expect(os.WriteFile(path, []byte("SGO_MERGE_TEST_KEY=from-dotenv\n"), 0644)).To(Succeed())

		env, err := envsource.Merge(ctx, []envsource.Source{&envsource.DotEnvSource{Path: path}})
		Expect(err).NotTo(HaveOccurred())

		// exec.Cmd keeps only the last occurrence of a duplicate key —
		// what matters is that "from-os-env" is the last of the two.
		var lastIndex, sourceIndex int
		for i, kv := range env {
			if kv == "SGO_MERGE_TEST_KEY=from-os-env" {
				lastIndex = i
			}
			if kv == "SGO_MERGE_TEST_KEY=from-dotenv" {
				sourceIndex = i
			}
		}
		Expect(lastIndex).To(BeNumerically(">", sourceIndex))
	})

	It("propagates a source's Fetch error", func() {
		_, err := envsource.Merge(ctx, []envsource.Source{&failingSource{}})
		Expect(err).To(HaveOccurred())
	})
})

var errFetchFailed = errors.New("fetch failed")

type failingSource struct{}

func (failingSource) Name() string                                     { return "failing" }
func (failingSource) Fetch(context.Context) ([]envsource.Value, error) { return nil, errFetchFailed }
func (failingSource) Write(context.Context, string, string) error      { return nil }
