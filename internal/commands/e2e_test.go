package commands_test

// End-to-end coverage for the actual `sgo` command surface (PLAN.md
// Phase 7): `sgo init` -> `sgo generate proto` -> edit the proto by hand
// -> `sgo generate code` -> `go build` the generated project. Everything
// else in this repo tests internal/codegen's Go API directly; this is
// the one place that drives the compiled binary as a subprocess, the
// same way a real user would, so flag parsing, the wizard/non-wizard
// branch, and each command's user-facing output are actually exercised
// rather than assumed to match their underlying codegen calls.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// sgoBinary is built once for the whole suite rather than per spec —
// `go build` of the CLI itself takes a few seconds, and nothing here
// mutates the binary.
var sgoBinary string

var _ = BeforeSuite(func() {
	buildDir, err := os.MkdirTemp("", "sgo-cli-build-*")
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() { Expect(os.RemoveAll(buildDir)).To(Succeed()) })

	sgoBinary = filepath.Join(buildDir, "sgo")

	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	Expect(err).NotTo(HaveOccurred())

	build := exec.Command("go", "build", "-o", sgoBinary, "./cmd")
	build.Dir = repoRoot
	out, err := build.CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), string(out))
})

// runSgo runs the built sgo binary with args in dir, with an empty
// (non-TTY) stdin so `sgo init` always takes the flag-driven path
// instead of launching the interactive wizard.
func runSgo(dir string, args ...string) (string, error) {
	GinkgoHelper()
	cmd := exec.Command(sgoBinary, args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader("")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

var _ = Describe("the sgo CLI, driven as a real subprocess", func() {
	It("inits a project, generates a service, edits the proto, regenerates, and the result builds", func() {
		root, err := os.MkdirTemp("", "sgo-cli-e2e-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		out, err := runSgo(root, "init", "demo",
			"--http-framework", "gin",
			"--persistence-mode", "orm",
			"--db", "postgres",
			"--cache", "redis",
			"--search", "elasticsearch",
		)
		Expect(err).NotTo(HaveOccurred(), out)
		Expect(out).To(ContainSubstring("Created demo"))

		projectDir := filepath.Join(root, "demo")
		Expect(filepath.Join(projectDir, "sgo.yaml")).To(BeAnExistingFile())

		out, err = runSgo(projectDir, "generate", "proto", "product")
		Expect(err).NotTo(HaveOccurred(), out)
		Expect(out).To(ContainSubstring("Created contract/pb/product.proto"))

		// A real hand-edit between generate proto and generate code, the
		// same workflow the README's quick start describes.
		protoPath := filepath.Join(projectDir, "contract", "pb", "product.proto")
		content, err := os.ReadFile(protoPath)
		Expect(err).NotTo(HaveOccurred())
		original := "message Product {\n  string id = 1;\n  string name = 2;\n  string description = 3;\n}"
		Expect(string(content)).To(ContainSubstring(original), "starter Product message shape changed, update this test's edit to match")
		edited := strings.Replace(string(content), original,
			"message Product {\n  string id = 1;\n  string name = 2;\n  string description = 3;\n  double price = 4;\n}", 1)
		Expect(os.WriteFile(protoPath, []byte(edited), 0644)).To(Succeed())

		out, err = runSgo(projectDir, "generate", "code", "product")
		Expect(err).NotTo(HaveOccurred(), out)
		Expect(out).To(ContainSubstring("Generated code for product"))

		domainContent, err := os.ReadFile(filepath.Join(projectDir, "internal", "core", "domain", "product", "product_gen.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(domainContent)).To(ContainSubstring("Price"), "the hand-edited price field should have made it into the generated domain struct")

		out, err = runSgo(projectDir, "list", "services")
		Expect(err).NotTo(HaveOccurred(), out)
		Expect(out).To(ContainSubstring("product"))

		build := exec.Command("go", "build", "./...")
		build.Dir = projectDir
		buildOut, err := build.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(buildOut))
	})

	It("exits non-zero with a clear message when generate code runs outside an sgo project", func() {
		root, err := os.MkdirTemp("", "sgo-cli-e2e-noproject-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		out, err := runSgo(root, "generate", "code", "user")

		Expect(err).To(HaveOccurred())
		Expect(out).To(ContainSubstring("not an sgo project"))
	})

	It("refuses to init a project whose directory already exists", func() {
		root, err := os.MkdirTemp("", "sgo-cli-e2e-existing-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		out, err := runSgo(root, "init", "demo", "--http-framework", "gin")
		Expect(err).NotTo(HaveOccurred(), out)

		out, err = runSgo(root, "init", "demo", "--http-framework", "gin")
		Expect(err).To(HaveOccurred())
		Expect(out).To(ContainSubstring("already exists"))
	})
})
