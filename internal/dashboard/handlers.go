package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func (s *Server) registerAPI(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/config", s.handleListConfig)
	mux.HandleFunc("GET /api/config/{key}", s.handleGetConfig)
	mux.HandleFunc("PUT /api/config/{key}", s.handlePutConfig)
	mux.HandleFunc("GET /api/events", s.handleEvents)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

// configListItem is GET /api/config's shape — deliberately no Value
// field. Secret values are masked by default (ARCHITECTURE.md §15): a
// value only ever crosses the wire once a client asks for that one key
// by name, via GET /api/config/{key}, an explicit reveal.
type configListItem struct {
	Key      string `json:"key"`
	Source   string `json:"source"`
	Writable bool   `json:"writable"`
}

func (s *Server) handleListConfig(w http.ResponseWriter, r *http.Request) {
	_, values, _, err := mergeSources(r.Context(), s.sources)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	items := make([]configListItem, len(values))
	for i, v := range values {
		items[i] = configListItem{Key: v.Key, Source: v.Source, Writable: v.Writable}
	}
	writeJSON(w, http.StatusOK, items)
}

type configItem struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Source   string `json:"source"`
	Writable bool   `json:"writable"`
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	_, values, _, err := mergeSources(r.Context(), s.sources)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	for _, v := range values {
		if v.Key == key {
			writeJSON(w, http.StatusOK, configItem{Key: v.Key, Value: v.Value, Source: v.Source, Writable: v.Writable})
			return
		}
	}
	writeError(w, http.StatusNotFound, fmt.Errorf("dashboard: no such config key %q", key))
}

type putConfigRequest struct {
	Value string `json:"value"`
}

// handlePutConfig writes key's new value back through whichever source
// declared it, then restarts the supervised child with the freshly
// merged environment so the edit takes effect immediately. A key only
// set in the real process environment (mergeSources reports it as
// source "env") has no owner and is rejected — writing its .env line,
// if any, wouldn't change what the child actually sees.
func (s *Server) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	var req putConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	ctx := r.Context()

	_, _, owner, err := mergeSources(ctx, s.sources)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	src, ok := owner[key]
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("dashboard: %q isn't writable — no configured source declares it, or it's set in the real environment", key))
		return
	}

	if err := src.Write(ctx, key, req.Value); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	env, _, _, err := mergeSources(ctx, s.sources)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := s.sup.Restart(env); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.hub.broadcast()

	writeJSON(w, http.StatusOK, configItem{Key: key, Value: req.Value, Source: src.Name(), Writable: true})
}

// handleEvents is a Server-Sent Events stream telling connected
// browsers when to refetch GET /api/config — after any PUT, from any
// client. It never carries a value itself (see configListItem).
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("dashboard: streaming unsupported"))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ch, cancel := s.hub.subscribe()
	defer cancel()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ch:
			fmt.Fprint(w, "data: refresh\n\n")
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprint(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}
