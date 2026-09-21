// Package webui implements `sgo ui` (ARCHITECTURE.md §11): a
// localhost-only HTTP server exposing a small JSON API under /api/* plus
// an embedded static frontend. Handlers call internal/codegen,
// internal/codegen/project, and internal/config directly — the same
// functions internal/commands' Cobra commands call — so this is a
// second frontend over the same engine as the CLI, never a separate
// code path (Decision #8).
package webui

import (
	"fmt"
	"net/http"
	"sync"
)

// Server holds the one piece of mutable state the web UI needs beyond
// what's already on disk: which directory it's currently serving. It
// starts as the directory `sgo ui` was run from. A successful
// POST /api/init moves it to the newly scaffolded project directory, so
// the dashboard that follows operates on the project just created
// without restarting the server — mirroring `sgo init myservice && cd
// myservice` in a single long-running process.
type Server struct {
	mu  sync.Mutex
	dir string
}

// NewServer returns a Server initially serving dir.
func NewServer(dir string) *Server {
	return &Server{dir: dir}
}

func (s *Server) projectDir() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dir
}

func (s *Server) setProjectDir(dir string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dir = dir
}

// Handler returns the server's http.Handler: the JSON API under /api/*
// plus the embedded static frontend for everything else.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.registerAPI(mux)
	mux.Handle("/", staticHandler())
	return mux
}

// Serve binds to 127.0.0.1:port, serving dir, and blocks until the
// server stops or errors.
func Serve(dir string, port int) error {
	s := NewServer(dir)
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	return http.ListenAndServe(addr, s.Handler())
}
