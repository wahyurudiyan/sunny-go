// Package dashboard implements the `sgo run --debug` config dashboard
// (ARCHITECTURE.md §15): a 127.0.0.1-only JSON API plus an embedded
// static frontend, alongside internal/webui in spirit. It lists every
// config key the running service's envsource.Sources declare, each
// one's effective source and whether it's writable, and lets editing a
// writable value write it back through its owning Source, then
// restarts the supervised child so the change takes effect.
package dashboard

import (
	"fmt"
	"net/http"

	"github.com/wahyurudiyan/sunny-go/internal/run"
	"github.com/wahyurudiyan/sunny-go/internal/run/envsource"
)

// Server holds what the dashboard needs beyond the sources themselves:
// the supervisor it restarts after a config write, and the hub that
// tells connected browsers to refetch when that happens.
type Server struct {
	sup     *run.Supervisor
	sources []envsource.Source
	hub     *hub
}

// NewServer returns a Server driving sup and reading/writing through
// sources. sup must already be running — `sgo run --debug` starts it
// before serving; the dashboard only ever restarts it, never starts it
// from cold.
func NewServer(sup *run.Supervisor, sources []envsource.Source) *Server {
	return &Server{sup: sup, sources: sources, hub: newHub()}
}

// Handler returns the dashboard's http.Handler: the JSON API under
// /api/* plus the embedded static frontend for everything else.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.registerAPI(mux)
	mux.Handle("/", staticHandler())
	return mux
}

// Serve binds to 127.0.0.1:port and blocks until the server stops or
// errors — 127.0.0.1-only, no auth, the same stance internal/webui
// takes (ARCHITECTURE.md §11).
func Serve(port int, sup *run.Supervisor, sources []envsource.Source) error {
	s := NewServer(sup, sources)
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	return http.ListenAndServe(addr, s.Handler())
}
