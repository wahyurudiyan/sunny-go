package commands_test

// End-to-end coverage for `sgo openapi ui` (PLAN.md Phase 14): starts
// the actual compiled binary as a real background server process and
// drives it over a real socket, the same standard the HTTP CRUD e2e
// suite (generate_test.go) already holds generated adapters to.

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("sgo openapi ui", func() {
	It("errors clearly when no OpenAPI document has been generated yet", func() {
		root, err := os.MkdirTemp("", "sgo-cli-openapi-ui-missing-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		projectDir := scaffoldProjectWithService(root)

		out, err := runSgo(projectDir, "openapi", "ui", "--port", "0")
		Expect(err).To(HaveOccurred())
		Expect(out).To(ContainSubstring("sgo generate openapi"))
	})

	It("serves the real interactive viewer over a real socket, showing the project's actual document", func() {
		root, err := os.MkdirTemp("", "sgo-cli-openapi-ui-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		projectDir := scaffoldProjectWithService(root)

		out, err := runSgo(projectDir, "generate", "openapi")
		Expect(err).NotTo(HaveOccurred(), out)

		server := exec.Command(sgoBinary, "openapi", "ui", "--port", "14749")
		server.Dir = projectDir
		var serverOutput bytes.Buffer
		server.Stdout = &serverOutput
		server.Stderr = &serverOutput
		Expect(server.Start()).To(Succeed())
		DeferCleanup(func() {
			_ = server.Process.Kill()
			_, _ = server.Process.Wait()
		})

		baseURL := "http://127.0.0.1:14749"
		client := &http.Client{Timeout: 2 * time.Second}
		Eventually(func() error {
			_, err := client.Get(baseURL)
			return err
		}, 5*time.Second, 100*time.Millisecond).Should(Succeed(), serverOutput.String())

		indexResp, err := client.Get(baseURL + "/")
		Expect(err).NotTo(HaveOccurred())
		defer indexResp.Body.Close()
		Expect(indexResp.StatusCode).To(Equal(http.StatusOK))
		indexBody, err := io.ReadAll(indexResp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(indexBody)).To(ContainSubstring(`<redoc spec-url="/openapi.yaml">`))
		Expect(string(indexBody)).To(ContainSubstring("/redoc.standalone.js"))

		jsResp, err := client.Get(baseURL + "/redoc.standalone.js")
		Expect(err).NotTo(HaveOccurred())
		defer jsResp.Body.Close()
		Expect(jsResp.StatusCode).To(Equal(http.StatusOK))
		jsBody, err := io.ReadAll(jsResp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(jsBody)).To(BeNumerically(">", 500_000))

		docResp, err := client.Get(baseURL + "/openapi.yaml")
		Expect(err).NotTo(HaveOccurred())
		defer docResp.Body.Close()
		Expect(docResp.StatusCode).To(Equal(http.StatusOK))
		docBody, err := io.ReadAll(docResp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(docBody)).To(ContainSubstring("/api/v1/products"), "the served document must be this project's own real generated routes")
	})
})
