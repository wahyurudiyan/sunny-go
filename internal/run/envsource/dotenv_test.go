package envsource_test

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/run/envsource"
)

var _ = Describe("DotEnvSource", func() {
	var (
		dir  string
		path string
		src  *envsource.DotEnvSource
		ctx  context.Context
	)

	BeforeEach(func() {
		var err error
		dir, err = os.MkdirTemp("", "sgo-envsource-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })

		path = filepath.Join(dir, ".env")
		src = &envsource.DotEnvSource{Path: path}
		ctx = context.Background()
	})

	It("reports its name as dotenv", func() {
		Expect(src.Name()).To(Equal("dotenv"))
	})

	Describe("Fetch", func() {
		It("returns no values, and no error, when the file doesn't exist", func() {
			values, err := src.Fetch(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(BeEmpty())
		})

		It("parses KEY=VALUE lines, skipping comments and blank lines", func() {
			content := "# a comment\n\nPORT=8080\nDATABASE_URL=postgres://localhost/app\n"
			Expect(os.WriteFile(path, []byte(content), 0644)).To(Succeed())

			values, err := src.Fetch(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(ConsistOf(
				envsource.Value{Key: "PORT", Value: "8080", Source: "dotenv"},
				envsource.Value{Key: "DATABASE_URL", Value: "postgres://localhost/app", Source: "dotenv"},
			))
		})

		It("strips matching quotes from a value", func() {
			Expect(os.WriteFile(path, []byte(`GREETING="hello world"`+"\n"+`NAME='ada'`+"\n"), 0644)).To(Succeed())

			values, err := src.Fetch(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(ConsistOf(
				envsource.Value{Key: "GREETING", Value: "hello world", Source: "dotenv"},
				envsource.Value{Key: "NAME", Value: "ada", Source: "dotenv"},
			))
		})

		It("understands an 'export KEY=VALUE' line", func() {
			Expect(os.WriteFile(path, []byte("export TOKEN=abc123\n"), 0644)).To(Succeed())

			values, err := src.Fetch(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(ConsistOf(envsource.Value{Key: "TOKEN", Value: "abc123", Source: "dotenv"}))
		})
	})

	Describe("Write", func() {
		It("creates the file when it doesn't exist yet", func() {
			Expect(src.Write(ctx, "PORT", "8080")).To(Succeed())

			content, err := os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("PORT=8080\n"))
		})

		It("updates an existing key in place, preserving every other line's order and content", func() {
			original := "# top comment\nFOO=1\nPORT=8080\nBAR=2\n"
			Expect(os.WriteFile(path, []byte(original), 0644)).To(Succeed())

			Expect(src.Write(ctx, "PORT", "9090")).To(Succeed())

			content, err := os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("# top comment\nFOO=1\nPORT=9090\nBAR=2\n"))
		})

		It("appends a new key at the end when it isn't already present", func() {
			Expect(os.WriteFile(path, []byte("FOO=1\n"), 0644)).To(Succeed())

			Expect(src.Write(ctx, "BAR", "2")).To(Succeed())

			content, err := os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("FOO=1\nBAR=2\n"))
		})

		It("round-trips: a written value is what the next Fetch reports", func() {
			Expect(src.Write(ctx, "PORT", "8080")).To(Succeed())

			values, err := src.Fetch(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(ConsistOf(envsource.Value{Key: "PORT", Value: "8080", Source: "dotenv"}))
		})
	})
})
