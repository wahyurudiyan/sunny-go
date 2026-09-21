package webui_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/webui"
)

func tempDir() string {
	GinkgoHelper()
	dir, err := os.MkdirTemp("", "sgo-webui-test-*")
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })
	return dir
}

// doJSON drives the server's http.Handler directly (no real listener
// needed) and decodes the JSON body generically, since these specs poke
// at a handful of fields per response rather than the whole shape.
func doJSON(handler http.Handler, method, path string, body any) (*httptest.ResponseRecorder, map[string]any) {
	GinkgoHelper()

	var buf bytes.Buffer
	if body != nil {
		Expect(json.NewEncoder(&buf).Encode(body)).To(Succeed())
	}

	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var parsed map[string]any
	if rec.Body.Len() > 0 {
		Expect(json.Unmarshal(rec.Body.Bytes(), &parsed)).To(Succeed())
	}
	return rec, parsed
}

var _ = Describe("GET /api/state", func() {
	It("reports hasProject=false when there's no sgo.yaml in the served directory", func() {
		h := webui.NewServer(tempDir()).Handler()

		rec, body := doJSON(h, http.MethodGet, "/api/state", nil)

		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(body["hasProject"]).To(BeFalse())
	})
})

var _ = Describe("POST /api/init", func() {
	It("rejects an invalid project name without touching disk", func() {
		dir := tempDir()
		h := webui.NewServer(dir).Handler()

		rec, body := doJSON(h, http.MethodPost, "/api/init", map[string]any{
			"name": "bad name", "httpFramework": "gin", "persistenceMode": "orm",
		})

		Expect(rec.Code).To(Equal(http.StatusBadRequest))
		Expect(body["error"]).To(ContainSubstring("invalid characters"))
	})

	It("scaffolds a project and moves the server's active directory to it", func() {
		dir := tempDir()
		h := webui.NewServer(dir).Handler()

		rec, body := doJSON(h, http.MethodPost, "/api/init", map[string]any{
			"name": "shop", "httpFramework": "echo", "persistenceMode": "orm",
		})
		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(body["hasProject"]).To(BeTrue())

		_, state := doJSON(h, http.MethodGet, "/api/state", nil)
		Expect(state["hasProject"]).To(BeTrue())
		cfg := state["config"].(map[string]any)
		Expect(cfg["httpFramework"]).To(Equal("echo"))
	})
})

var _ = Describe("GET/PUT /api/config", func() {
	var (
		dir string
		h   http.Handler
	)

	BeforeEach(func() {
		dir = tempDir()
		h = webui.NewServer(dir).Handler()
		rec, _ := doJSON(h, http.MethodPost, "/api/init", map[string]any{
			"name": "shop", "httpFramework": "gin", "persistenceMode": "orm",
		})
		Expect(rec.Code).To(Equal(http.StatusOK))
	})

	It("round-trips a valid edit", func() {
		rec, body := doJSON(h, http.MethodGet, "/api/config", nil)
		Expect(rec.Code).To(Equal(http.StatusOK))
		raw := body["yaml"].(string)
		Expect(raw).To(ContainSubstring("httpFramework: gin"))

		edited := raw + "\ncache:\n    - redis\n"
		rec, body = doJSON(h, http.MethodPut, "/api/config", map[string]any{"yaml": edited})
		Expect(rec.Code).To(Equal(http.StatusOK))
		cfg := body["config"].(map[string]any)
		Expect(cfg["cache"]).To(ConsistOf("redis"))
	})

	It("rejects an invalid engine and leaves the file untouched", func() {
		rec, _ := doJSON(h, http.MethodGet, "/api/config", nil)

		rec, body := doJSON(h, http.MethodPut, "/api/config", map[string]any{
			"yaml": "module: shop\nhttpFramework: gin\npersistence:\n    mode: orm\ncache:\n    - memcached\n",
		})
		Expect(rec.Code).To(Equal(http.StatusBadRequest))
		Expect(body["error"]).To(ContainSubstring("invalid cache engine"))

		rec, body = doJSON(h, http.MethodGet, "/api/config", nil)
		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(body["yaml"]).NotTo(ContainSubstring("memcached"))
	})
})

var _ = Describe("the service lifecycle: create proto, edit it, generate code", func() {
	var (
		dir string
		h   http.Handler
	)

	BeforeEach(func() {
		dir = tempDir()
		h = webui.NewServer(dir).Handler()
		rec, _ := doJSON(h, http.MethodPost, "/api/init", map[string]any{
			"name": "shop", "httpFramework": "gin", "persistenceMode": "orm",
		})
		Expect(rec.Code).To(Equal(http.StatusOK))
	})

	It("goes from pending proto to a fully generated service", func() {
		rec, status := doJSON(h, http.MethodPost, "/api/services", map[string]any{"name": "product"})
		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(status["proto"]).To(BeTrue())
		Expect(status["serviceImpl"]).To(BeFalse())

		_, state := doJSON(h, http.MethodGet, "/api/state", nil)
		Expect(state["pendingProtos"]).To(ConsistOf("product"))
		Expect(state["services"]).To(BeNil(), "no services generated yet, the JSON key is omitted entirely")

		rec, body := doJSON(h, http.MethodGet, "/api/services/product/proto", nil)
		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(body["content"]).To(ContainSubstring("message Product"))

		edited := body["content"].(string) + "\n// hand-edited\n"
		rec, _ = doJSON(h, http.MethodPut, "/api/services/product/proto", map[string]any{"content": edited})
		Expect(rec.Code).To(Equal(http.StatusOK))

		rec, status = doJSON(h, http.MethodPost, "/api/services/product/generate", nil)
		Expect(rec.Code).To(Equal(http.StatusOK), "%v", status)
		Expect(status["proto"]).To(BeTrue())
		Expect(status["contractGen"]).To(BeTrue())
		Expect(status["domainEntity"]).To(BeTrue())
		Expect(status["serviceImpl"]).To(BeTrue())

		_, state = doJSON(h, http.MethodGet, "/api/state", nil)
		Expect(state["pendingProtos"]).To(BeNil(), "the service is now tracked in sgo.yaml, no longer pending")
		services := state["services"].([]any)
		Expect(services).To(HaveLen(1))
	})

	It("404s reading a proto that doesn't exist", func() {
		rec, _ := doJSON(h, http.MethodGet, "/api/services/nope/proto", nil)
		Expect(rec.Code).To(Equal(http.StatusNotFound))
	})

	It("404s writing a proto that was never created", func() {
		rec, _ := doJSON(h, http.MethodPut, "/api/services/nope/proto", map[string]any{"content": "x"})
		Expect(rec.Code).To(Equal(http.StatusNotFound))
	})
})
