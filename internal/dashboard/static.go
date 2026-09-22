package dashboard

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static
var staticFS embed.FS

// staticHandler serves the embedded frontend (static/index.html,
// app.js, style.css) — plain HTML/CSS/JS, no build step or npm
// dependency, matching internal/webui's own static frontend.
func staticHandler() http.Handler {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		// staticFS is embedded at build time; a missing "static"
		// subdirectory would fail every build, not just at runtime.
		panic(err)
	}
	return http.FileServer(http.FS(sub))
}
