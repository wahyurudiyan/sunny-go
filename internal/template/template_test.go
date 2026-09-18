package template_test

import (
	"embed"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/template"
)

//go:embed testdata/*.tmpl
var testdataFS embed.FS

var _ = Describe("Engine", func() {
	var (
		engine *template.Engine
		dir    string
	)

	BeforeEach(func() {
		engine = template.New(testdataFS)

		var err error
		dir, err = os.MkdirTemp("", "sgo-template-test-*")
		Expect(err).NotTo(HaveOccurred())

		DeferCleanup(func() {
			Expect(os.RemoveAll(dir)).To(Succeed())
		})
	})

	Describe("Render", func() {
		Context("when the template and destination are valid", func() {
			It("writes the rendered output to destPath", func() {
				dest := filepath.Join(dir, "greeting.txt")

				err := engine.Render("testdata/greeting.tmpl", struct{ Name string }{Name: "sgo"}, dest)
				Expect(err).NotTo(HaveOccurred())

				got, err := os.ReadFile(dest)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(got)).To(Equal("Hello, sgo!\n"))
			})

			It("creates missing parent directories for destPath", func() {
				dest := filepath.Join(dir, "nested", "deeper", "greeting.txt")

				err := engine.Render("testdata/greeting.tmpl", struct{ Name string }{Name: "sgo"}, dest)
				Expect(err).NotTo(HaveOccurred())

				Expect(dest).To(BeAnExistingFile())
			})
		})

		Context("when the template path doesn't exist in the embedded fs", func() {
			It("returns an error", func() {
				dest := filepath.Join(dir, "out.txt")

				err := engine.Render("testdata/missing.tmpl", nil, dest)

				Expect(err).To(HaveOccurred())
			})
		})

		Context("when the template references a field the data doesn't have", func() {
			It("returns an error instead of writing a partial file", func() {
				dest := filepath.Join(dir, "out.txt")

				err := engine.Render("testdata/greeting.tmpl", struct{ Other string }{Other: "x"}, dest)

				Expect(err).To(HaveOccurred())
			})
		})
	})
})
