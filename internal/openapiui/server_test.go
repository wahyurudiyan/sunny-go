package openapiui_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/config"
	"github.com/wahyurudiyan/sunny-go/internal/openapiui"
)

const sampleDoc = `openapi: 3.0.3
info:
  title: shop
  version: 0.1.0
paths: {}
`

var _ = Describe("Handler", func() {
	var (
		dir string
		cfg *config.Config
	)

	BeforeEach(func() {
		var err error
		dir, err = os.MkdirTemp("", "sgo-openapiui-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })

		cfg = &config.Config{Module: "shop", OpenAPI: config.OpenAPI{Format: config.OpenAPIFormatYAML}}
	})

	Context("when the project has no generated OpenAPI document yet", func() {
		It("errors clearly instead of starting a server with nothing to show", func() {
			_, err := openapiui.Handler(dir, cfg)
			Expect(err).To(MatchError(ContainSubstring("sgo generate openapi")))
		})
	})

	Context("when docs/openapi.yaml exists", func() {
		BeforeEach(func() {
			Expect(os.MkdirAll(filepath.Join(dir, "docs"), 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(dir, "docs", "openapi.yaml"), []byte(sampleDoc), 0644)).To(Succeed())
		})

		It("serves an index page embedding the Redoc custom element pointed at /openapi.yaml", func() {
			handler, err := openapiui.Handler(dir, cfg)
			Expect(err).NotTo(HaveOccurred())

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusOK))
			body := rec.Body.String()
			Expect(body).To(ContainSubstring(`<redoc spec-url="/openapi.yaml">`))
			Expect(body).To(ContainSubstring(`/redoc.standalone.js`))
			Expect(body).To(ContainSubstring("shop"), "the project's module name should appear in the page title")
		})

		It("serves the real vendored Redoc bundle, not a placeholder", func() {
			handler, err := openapiui.Handler(dir, cfg)
			Expect(err).NotTo(HaveOccurred())

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/redoc.standalone.js", nil)
			handler.ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(rec.Header().Get("Content-Type")).To(ContainSubstring("javascript"))
			Expect(rec.Body.Len()).To(BeNumerically(">", 500_000), "the real Redoc standalone bundle is over 1MB — a placeholder or truncated file would be tiny")
			Expect(rec.Body.String()).To(ContainSubstring("ReDoc"))
		})

		It("serves the project's actual OpenAPI document content, read fresh from disk", func() {
			handler, err := openapiui.Handler(dir, cfg)
			Expect(err).NotTo(HaveOccurred())

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
			handler.ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(rec.Body.String()).To(Equal(sampleDoc))

			// Regenerating the doc while the server is up (no restart)
			// must be reflected on the next request — this is the "reads
			// whatever's on disk" contract, not a cached-at-start copy.
			updated := sampleDoc + "# regenerated\n"
			Expect(os.WriteFile(filepath.Join(dir, "docs", "openapi.yaml"), []byte(updated), 0644)).To(Succeed())

			rec2 := httptest.NewRecorder()
			handler.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
			Expect(rec2.Body.String()).To(Equal(updated))
		})

		It("404s an unknown path instead of falling back to the index", func() {
			handler, err := openapiui.Handler(dir, cfg)
			Expect(err).NotTo(HaveOccurred())

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
			handler.ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusNotFound))
		})
	})

	Context("when sgo.yaml selects the json format", func() {
		It("serves the document at /openapi.json instead", func() {
			cfg.OpenAPI.Format = config.OpenAPIFormatJSON
			Expect(os.MkdirAll(filepath.Join(dir, "docs"), 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(dir, "docs", "openapi.json"), []byte(`{"openapi":"3.0.3"}`), 0644)).To(Succeed())

			handler, err := openapiui.Handler(dir, cfg)
			Expect(err).NotTo(HaveOccurred())

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
			Expect(rec.Body.String()).To(ContainSubstring(`spec-url="/openapi.json"`))

			rec2 := httptest.NewRecorder()
			handler.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
			Expect(rec2.Code).To(Equal(http.StatusOK))
			body, err := io.ReadAll(rec2.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(body)).To(Equal(`{"openapi":"3.0.3"}`))
		})
	})
})
