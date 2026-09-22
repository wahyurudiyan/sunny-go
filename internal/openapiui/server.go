package openapiui

import (
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/openapigen"
	"github.com/wahyurudiyan/sunny-go/internal/config"
)

var indexTemplate = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html>
<head>
  <title>{{.Title}} — API Reference</title>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1"/>
  <style>body { margin: 0; padding: 0; }</style>
</head>
<body>
  <redoc spec-url="{{.SpecURL}}"></redoc>
  <script src="/redoc.standalone.js"></script>
</body>
</html>
`))

// Handler returns the http.Handler serving the doc viewer for
// projectDir's docs/openapi.<ext> (per cfg.OpenAPI.Format). Errors up
// front — before starting any server — if that document doesn't exist
// yet, the same "generate it first" contract `sgo openapi validate`
// already has (ARCHITECTURE.md §13/§19).
func Handler(projectDir string, cfg *config.Config) (http.Handler, error) {
	docPath := openapigen.Path(projectDir, cfg.OpenAPI.Format)
	if _, err := os.Stat(docPath); err != nil {
		return nil, fmt.Errorf("no OpenAPI document at %s — run `sgo generate openapi` first", docPath)
	}

	docName := openapigen.DocsBaseName + "." + openapigen.Ext(cfg.OpenAPI.Format)
	specURL := "/" + docName

	mux := http.NewServeMux()

	mux.HandleFunc("/"+docName, func(w http.ResponseWriter, r *http.Request) {
		// Served fresh from disk on every request, not cached at server
		// start — so a `sgo generate openapi` run while the server is up
		// shows up on the next browser refresh, no restart needed.
		http.ServeFile(w, r, docPath)
	})

	mux.HandleFunc("/redoc.standalone.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		data, err := vendorFS.ReadFile("vendor/redoc.standalone.js")
		if err != nil {
			// vendorFS is embedded at build time; this can only fail if
			// the embed itself is broken, which would fail every build,
			// not just at runtime.
			panic(err)
		}
		_, _ = w.Write(data)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = indexTemplate.Execute(w, struct {
			Title   string
			SpecURL string
		}{Title: cfg.Module, SpecURL: specURL})
	})

	return mux, nil
}

// Serve binds to 127.0.0.1:port, serving projectDir's OpenAPI document
// through the embedded Redoc viewer, and blocks until the server stops
// or errors. 127.0.0.1-only, no auth — the same stance `sgo ui` and
// `sgo run --debug`'s config dashboard already take, since neither ever
// listens on anything but loopback.
func Serve(projectDir string, cfg *config.Config, port int) error {
	handler, err := Handler(projectDir, cfg)
	if err != nil {
		return err
	}

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	return http.ListenAndServe(addr, handler)
}
