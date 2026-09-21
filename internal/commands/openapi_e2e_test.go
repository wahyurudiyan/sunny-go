package commands_test

// End-to-end coverage for `sgo generate openapi` and `sgo openapi
// validate` (PLAN.md Phase 8), driving the real compiled binary the
// same way e2e_test.go does for the rest of the command surface.

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// scaffoldProjectWithService inits a project and generates one service
// through the real CLI — the common setup every spec below starts from.
func scaffoldProjectWithService(root string) string {
	GinkgoHelper()

	out, err := runSgo(root, "init", "shop", "--http-framework", "gin", "--persistence-mode", "orm")
	Expect(err).NotTo(HaveOccurred(), out)

	projectDir := filepath.Join(root, "shop")

	out, err = runSgo(projectDir, "generate", "proto", "product")
	Expect(err).NotTo(HaveOccurred(), out)

	out, err = runSgo(projectDir, "generate", "code", "product")
	Expect(err).NotTo(HaveOccurred(), out)

	return projectDir
}

var _ = Describe("sgo generate openapi", func() {
	It("writes docs/openapi.yaml by default and it validates", func() {
		root, err := os.MkdirTemp("", "sgo-cli-openapi-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		projectDir := scaffoldProjectWithService(root)

		out, err := runSgo(projectDir, "generate", "openapi")
		Expect(err).NotTo(HaveOccurred(), out)
		Expect(out).To(ContainSubstring("Generated"))
		Expect(out).To(ContainSubstring(filepath.Join("docs", "openapi.yaml")))

		Expect(filepath.Join(projectDir, "docs", "openapi.yaml")).To(BeAnExistingFile())
	})

	It("--version/--format override sgo.yaml for one run without persisting the change", func() {
		root, err := os.MkdirTemp("", "sgo-cli-openapi-override-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		projectDir := scaffoldProjectWithService(root)

		sgoYAMLBefore, err := os.ReadFile(filepath.Join(projectDir, "sgo.yaml"))
		Expect(err).NotTo(HaveOccurred())

		out, err := runSgo(projectDir, "generate", "openapi", "--version", "3.1", "--format", "json")
		Expect(err).NotTo(HaveOccurred(), out)

		Expect(filepath.Join(projectDir, "docs", "openapi.json")).To(BeAnExistingFile())

		content, err := os.ReadFile(filepath.Join(projectDir, "docs", "openapi.json"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring(`"openapi": "3.1.0"`))

		sgoYAMLAfter, err := os.ReadFile(filepath.Join(projectDir, "sgo.yaml"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(sgoYAMLAfter)).To(Equal(string(sgoYAMLBefore)), "a one-off --version/--format override must not rewrite sgo.yaml")
	})

	It("errors clearly outside an sgo project", func() {
		root, err := os.MkdirTemp("", "sgo-cli-openapi-noproject-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		out, err := runSgo(root, "generate", "openapi")
		Expect(err).To(HaveOccurred())
		Expect(out).To(ContainSubstring("not an sgo project"))
	})
})

var _ = Describe("sgo openapi validate", func() {
	It("validates the project's own generated doc with no path given", func() {
		root, err := os.MkdirTemp("", "sgo-cli-openapi-validate-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		projectDir := scaffoldProjectWithService(root)

		out, err := runSgo(projectDir, "generate", "openapi")
		Expect(err).NotTo(HaveOccurred(), out)

		out, err = runSgo(projectDir, "openapi", "validate")
		Expect(err).NotTo(HaveOccurred(), out)
		Expect(out).To(ContainSubstring("valid OpenAPI document"))
	})

	It("validates an explicit path outside any sgo project", func() {
		root, err := os.MkdirTemp("", "sgo-cli-openapi-standalone-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		docPath := filepath.Join(root, "hand-written.json")
		Expect(os.WriteFile(docPath, []byte(`{"openapi":"3.0.3","info":{"title":"t","version":"1"},"paths":{}}`), 0644)).To(Succeed())

		out, err := runSgo(root, "openapi", "validate", docPath)
		Expect(err).NotTo(HaveOccurred(), out)
		Expect(out).To(ContainSubstring("valid OpenAPI document"))
	})

	It("exits non-zero with a clear error on an invalid document", func() {
		root, err := os.MkdirTemp("", "sgo-cli-openapi-invalid-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		docPath := filepath.Join(root, "broken.json")
		Expect(os.WriteFile(docPath, []byte(`{"openapi":"3.0.3","info":{"title":"t"},"paths":{}}`), 0644)).To(Succeed())

		out, err := runSgo(root, "openapi", "validate", docPath)
		Expect(err).To(HaveOccurred())
		Expect(out).To(ContainSubstring("not valid OpenAPI"))
	})

	It("errors clearly when no path is given and there's no sgo project", func() {
		root, err := os.MkdirTemp("", "sgo-cli-openapi-validate-noproject-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		out, err := runSgo(root, "openapi", "validate")
		Expect(err).To(HaveOccurred())
		Expect(out).To(ContainSubstring("not an sgo project"))
	})
})
