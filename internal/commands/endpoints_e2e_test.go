package commands_test

// End-to-end coverage for `sgo list endpoints` (PLAN.md Phase 13),
// driving the real compiled binary the same way e2e_test.go does for
// the rest of the command surface.

import (
	"os"
	"path/filepath"
	"regexp"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("sgo list endpoints", func() {
	It("prints every route the generated Gin adapter actually registers", func() {
		root, err := os.MkdirTemp("", "sgo-cli-endpoints-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		projectDir := scaffoldProjectWithService(root)

		out, err := runSgo(projectDir, "list", "endpoints")
		Expect(err).NotTo(HaveOccurred(), out)
		Expect(out).To(ContainSubstring("product:"))

		routesPath := filepath.Join(projectDir, "internal", "infrastructure", "transport", "http", "gin", "product_routes_gen.go")
		routesSrc, err := os.ReadFile(routesPath)
		Expect(err).NotTo(HaveOccurred())

		// Cross-check every printed "VERB path" line against the actual
		// generated registration (engine.VERB("path" — same load-bearing
		// property openapigen/generate_test.go already verifies for the
		// OpenAPI document: this command must not be a second, drifting
		// derivation of what the adapter serves.
		lineRe := regexp.MustCompile(`(?m)^\s+(GET|POST|PUT|DELETE)\s+(\S+)\s+(\w+)$`)
		matches := lineRe.FindAllStringSubmatch(out, -1)
		Expect(matches).To(HaveLen(5), "expected one line per CRUD RPC:\n%s", out)

		for _, m := range matches {
			verb, path, rpc := m[1], m[2], m[3]
			registration := verb + `("` + path + `"`
			Expect(string(routesSrc)).To(ContainSubstring(registration), "printed %s %s (%s) has no matching registration in the generated routes file", verb, path, rpc)
		}

		Expect(out).To(ContainSubstring("POST   /api/v1/products"))
		Expect(out).To(ContainSubstring("GET    /api/v1/products/:id"))
		Expect(out).To(ContainSubstring("DELETE /api/v1/products/:id"))
	})

	It("filters to one service when named", func() {
		root, err := os.MkdirTemp("", "sgo-cli-endpoints-filter-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		projectDir := scaffoldProjectWithService(root)

		out, err := runSgo(projectDir, "generate", "proto", "order")
		Expect(err).NotTo(HaveOccurred(), out)
		out, err = runSgo(projectDir, "generate", "code", "order")
		Expect(err).NotTo(HaveOccurred(), out)

		out, err = runSgo(projectDir, "list", "endpoints", "product")
		Expect(err).NotTo(HaveOccurred(), out)
		Expect(out).To(ContainSubstring("product:"))
		Expect(out).NotTo(ContainSubstring("order:"))
	})

	It("errors clearly for a service not tracked in sgo.yaml", func() {
		root, err := os.MkdirTemp("", "sgo-cli-endpoints-unknown-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		projectDir := scaffoldProjectWithService(root)

		out, err := runSgo(projectDir, "list", "endpoints", "nope")
		Expect(err).To(HaveOccurred())
		Expect(out).To(ContainSubstring(`"nope" not found`))
	})

	It("says so plainly when no services have been generated yet", func() {
		root, err := os.MkdirTemp("", "sgo-cli-endpoints-empty-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		out, err := runSgo(root, "init", "shop", "--http-framework", "gin", "--persistence-mode", "orm")
		Expect(err).NotTo(HaveOccurred(), out)
		projectDir := filepath.Join(root, "shop")

		out, err = runSgo(projectDir, "list", "endpoints")
		Expect(err).NotTo(HaveOccurred(), out)
		Expect(out).To(ContainSubstring("No services yet"))
	})
})
