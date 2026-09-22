package dashboard_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/dashboard"
	"github.com/wahyurudiyan/sunny-go/internal/run"
	"github.com/wahyurudiyan/sunny-go/internal/run/envsource"
)

// doJSON drives the server's http.Handler directly (no real listener
// needed for anything but the SSE spec below) and decodes the JSON
// body generically.
func doJSON(handler http.Handler, method, path string, body any) (*httptest.ResponseRecorder, any) {
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

	var parsed any
	if rec.Body.Len() > 0 {
		Expect(json.Unmarshal(rec.Body.Bytes(), &parsed)).To(Succeed())
	}
	return rec, parsed
}

var _ = Describe("dashboard API", func() {
	var (
		envPath string
		sup     *run.Supervisor
		out     *safeBuffer
		srv     *dashboard.Server
	)

	BeforeEach(func() {
		dir, err := os.MkdirTemp("", "sgo-dashboard-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })

		envPath = filepath.Join(dir, ".env")
		Expect(os.WriteFile(envPath, []byte("GREETING=hello\n"), 0644)).To(Succeed())

		out = &safeBuffer{}
		sup = &run.Supervisor{
			ProjectDir: "../run/testdata/fixture",
			CmdDir:     "cmd/testapp",
			Stdout:     out,
			Stderr:     out,
		}
		Expect(sup.Start(append(os.Environ(), "GREETING=hello"))).To(Succeed())
		DeferCleanup(func() { Expect(sup.Stop()).To(Succeed()) })

		srv = dashboard.NewServer(sup, []envsource.Source{&envsource.DotEnvSource{Path: envPath}})
	})

	Describe("GET /api/config", func() {
		It("lists every key any source declares, without its value", func() {
			_, body := doJSON(srv.Handler(), http.MethodGet, "/api/config", nil)

			items := body.([]any)
			Expect(items).To(HaveLen(1))
			item := items[0].(map[string]any)
			Expect(item["key"]).To(Equal("GREETING"))
			Expect(item["source"]).To(Equal("dotenv"))
			Expect(item["writable"]).To(BeTrue())
			Expect(item).NotTo(HaveKey("value"))
		})
	})

	Describe("GET /api/config/{key}", func() {
		It("reveals the key's current value on request", func() {
			rec, body := doJSON(srv.Handler(), http.MethodGet, "/api/config/GREETING", nil)

			Expect(rec.Code).To(Equal(http.StatusOK))
			item := body.(map[string]any)
			Expect(item["value"]).To(Equal("hello"))
			Expect(item["source"]).To(Equal("dotenv"))
		})

		It("404s for a key no source declares", func() {
			rec, _ := doJSON(srv.Handler(), http.MethodGet, "/api/config/NOPE", nil)
			Expect(rec.Code).To(Equal(http.StatusNotFound))
		})
	})

	Describe("PUT /api/config/{key}", func() {
		It("writes the new value back to its source and restarts the child", func() {
			Eventually(out.String, 10*time.Second, 50*time.Millisecond).Should(ContainSubstring("GREETING=hello"))

			rec, body := doJSON(srv.Handler(), http.MethodPut, "/api/config/GREETING", map[string]string{"value": "world"})
			Expect(rec.Code).To(Equal(http.StatusOK))
			item := body.(map[string]any)
			Expect(item["value"]).To(Equal("world"))

			content, err := os.ReadFile(envPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("GREETING=world\n"))

			Eventually(out.String, 10*time.Second, 50*time.Millisecond).Should(ContainSubstring("GREETING=world"))
		})

		It("404s a key with no owning source, e.g. one only set in the real environment", func() {
			rec, _ := doJSON(srv.Handler(), http.MethodPut, "/api/config/PATH", map[string]string{"value": "/nope"})
			Expect(rec.Code).To(Equal(http.StatusNotFound))
		})
	})

	Describe("GET /api/events", func() {
		It("tells a connected client to refresh after a PUT changes config", func() {
			ts := httptest.NewServer(srv.Handler())
			DeferCleanup(ts.Close)

			req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/events", nil)
			Expect(err).NotTo(HaveOccurred())

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func() { Expect(resp.Body.Close()).To(Succeed()) })
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			lines := make(chan string, 16)
			go func() {
				scanner := bufio.NewScanner(resp.Body)
				for scanner.Scan() {
					lines <- scanner.Text()
				}
			}()

			putReq, err := http.NewRequest(http.MethodPut, ts.URL+"/api/config/GREETING",
				bytes.NewBufferString(`{"value":"world"}`))
			Expect(err).NotTo(HaveOccurred())
			putReq.Header.Set("Content-Type", "application/json")
			putResp, err := http.DefaultClient.Do(putReq)
			Expect(err).NotTo(HaveOccurred())
			Expect(putResp.Body.Close()).To(Succeed())
			Expect(putResp.StatusCode).To(Equal(http.StatusOK))

			Eventually(lines, 5*time.Second).Should(Receive(ContainSubstring("data: refresh")))
		})
	})
})
