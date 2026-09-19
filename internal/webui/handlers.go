package webui

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/wahyurudiyan/sunny-go/internal/codegen"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/project"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
	"github.com/wahyurudiyan/sunny-go/internal/config"
)

func (s *Server) registerAPI(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/state", s.handleState)
	mux.HandleFunc("POST /api/init", s.handleInit)
	mux.HandleFunc("GET /api/config", s.handleGetConfig)
	mux.HandleFunc("PUT /api/config", s.handlePutConfig)
	mux.HandleFunc("POST /api/services", s.handleCreateService)
	mux.HandleFunc("GET /api/services/{name}/proto", s.handleGetProto)
	mux.HandleFunc("PUT /api/services/{name}/proto", s.handlePutProto)
	mux.HandleFunc("POST /api/services/{name}/generate", s.handleGenerate)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

// stateResponse is GET /api/state's shape: either "no project here yet,
// show the create-project form" or the full dashboard payload.
type stateResponse struct {
	HasProject    bool                    `json:"hasProject"`
	ProjectDir    string                  `json:"projectDir,omitempty"`
	Config        *config.Config          `json:"config,omitempty"`
	Services      []codegen.ServiceStatus `json:"services,omitempty"`
	PendingProtos []string                `json:"pendingProtos,omitempty"`
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	dir := s.projectDir()

	cfg, err := config.Load(dir)
	if err != nil {
		writeJSON(w, http.StatusOK, stateResponse{HasProject: false})
		return
	}

	services := make([]codegen.ServiceStatus, 0, len(cfg.Services))
	for _, name := range cfg.Services {
		services = append(services, codegen.Status(dir, name))
	}

	writeJSON(w, http.StatusOK, stateResponse{
		HasProject:    true,
		ProjectDir:    dir,
		Config:        cfg,
		Services:      services,
		PendingProtos: pendingProtos(dir, cfg),
	})
}

// pendingProtos lists contract/pb/*.proto files not yet recorded in
// cfg.Services — created via POST /api/services (`sgo generate proto`'s
// equivalent) but not yet run through generate code, the same gap
// `sgo generate proto` alone leaves in the CLI (only `generate code`
// appends to sgo.yaml's services list).
func pendingProtos(dir string, cfg *config.Config) []string {
	tracked := make(map[string]bool, len(cfg.Services))
	for _, name := range cfg.Services {
		tracked[name] = true
	}

	matches, err := filepath.Glob(filepath.Join(dir, "contract", "pb", "*.proto"))
	if err != nil {
		return nil
	}

	var pending []string
	for _, m := range matches {
		name := strings.TrimSuffix(filepath.Base(m), ".proto")
		if !tracked[name] {
			pending = append(pending, name)
		}
	}
	sort.Strings(pending)
	return pending
}

type initRequest struct {
	Name            string   `json:"name"`
	Module          string   `json:"module"`
	HTTPFramework   string   `json:"httpFramework"`
	PersistenceMode string   `json:"persistenceMode"`
	DB              []string `json:"db"`
	Cache           []string `json:"cache"`
	Search          []string `json:"search"`
}

func toEngines[T ~string](raw []string) []T {
	out := make([]T, len(raw))
	for i, v := range raw {
		out[i] = T(v)
	}
	return out
}

func (s *Server) handleInit(w http.ResponseWriter, r *http.Request) {
	var req initRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	opts, err := project.BuildOptions(
		req.Name, req.Module,
		config.HTTPFramework(req.HTTPFramework),
		config.PersistenceMode(req.PersistenceMode),
		toEngines[config.PersistenceEngine](req.DB),
		toEngines[config.CacheEngine](req.Cache),
		toEngines[config.SearchEngine](req.Search),
	)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	destDir := filepath.Join(s.projectDir(), opts.Name)
	if err := project.Scaffold(destDir, *opts); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	s.setProjectDir(destDir)

	cfg, err := config.Load(destDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, stateResponse{HasProject: true, ProjectDir: destDir, Config: cfg})
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	dir := s.projectDir()

	raw, err := os.ReadFile(filepath.Join(dir, config.FileName))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}

	cfg, err := config.Load(dir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"yaml": string(raw), "config": cfg})
}

type putConfigRequest struct {
	YAML string `json:"yaml"`
}

// handlePutConfig lets the dashboard's sgo.yaml editor save hand-edited
// YAML directly — validated the same way config.Load + Validate always
// is, then re-saved through Config.Save so the file is re-marshaled
// canonically, same as every other sgo.yaml write in this codebase.
func (s *Server) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	var req putConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var cfg config.Config
	if err := yaml.Unmarshal([]byte(req.YAML), &cfg); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := cfg.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	dir := s.projectDir()
	if err := cfg.Save(dir); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	raw, err := os.ReadFile(filepath.Join(dir, config.FileName))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"yaml": string(raw), "config": &cfg})
}

type createServiceRequest struct {
	Name string `json:"name"`
}

// handleCreateService is `sgo generate proto <name>`'s equivalent: it
// only writes the starter proto stub, same as the CLI command of the
// same name — it does not touch sgo.yaml (see pendingProtos above).
func (s *Server) handleCreateService(w http.ResponseWriter, r *http.Request) {
	dir := s.projectDir()

	cfg, err := config.Load(dir)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var req createServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := sgoproto.GenerateStub(filepath.Join(dir, "contract", "pb"), req.Name, cfg.Module); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusOK, codegen.Status(dir, req.Name))
}

func (s *Server) handleGetProto(w http.ResponseWriter, r *http.Request) {
	dir := s.projectDir()
	name := r.PathValue("name")

	content, err := os.ReadFile(filepath.Join(dir, "contract", "pb", name+".proto"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"content": string(content)})
}

type putProtoRequest struct {
	Content string `json:"content"`
}

// handlePutProto is the hand-edit step the README's quick start
// describes ("edit contract/pb/user.proto") done in the browser instead
// of a text editor — it only ever overwrites a proto that
// POST /api/services (== `sgo generate proto`) already created, never
// creates one itself, matching that command's own "refuses to overwrite"
// / create-once semantics from the other direction.
func (s *Server) handlePutProto(w http.ResponseWriter, r *http.Request) {
	dir := s.projectDir()
	name := r.PathValue("name")
	path := filepath.Join(dir, "contract", "pb", name+".proto")

	if _, err := os.Stat(path); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}

	var req putProtoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := os.WriteFile(path, []byte(req.Content), 0644); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"content": req.Content})
}

// handleGenerate is `sgo generate code <name>`'s equivalent, calling the
// exact same codegen.GenerateCode the CLI command calls.
func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	dir := s.projectDir()
	name := r.PathValue("name")

	cfg, err := config.Load(dir)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := codegen.GenerateCode(dir, name, cfg); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusOK, codegen.Status(dir, name))
}
