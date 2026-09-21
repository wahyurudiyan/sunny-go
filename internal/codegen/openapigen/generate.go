package openapigen

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wahyurudiyan/sunny-go/internal/config"
)

// DocsDir and DocsBaseName make up docs/openapi.<ext> — the fixed,
// always-overwritten output location, the same "generated, never
// hand-edited" contract contract/gen and wire_gen.go already have.
const (
	DocsDir      = "docs"
	DocsBaseName = "openapi"
)

// Path returns the doc path Generate writes for a project using format.
func Path(projectDir string, format config.OpenAPIFormat) string {
	return filepath.Join(projectDir, DocsDir, DocsBaseName+"."+Ext(format))
}

// Generate builds a fresh OpenAPI document from every service
// registered in cfg.Services, encodes it per cfg.OpenAPI, writes it to
// docs/openapi.<ext>, and immediately validates the written bytes
// against the real OpenAPI meta-schema — a safety net catching a
// generator bug before it ships an invalid document, not just a
// user-facing feature (ARCHITECTURE.md §13). Returns the path written.
func Generate(cfg *config.Config, projectDir string) (string, error) {
	doc, err := Build(cfg, projectDir)
	if err != nil {
		return "", err
	}

	data, err := Encode(doc, cfg.OpenAPI.Version, cfg.OpenAPI.Format)
	if err != nil {
		return "", err
	}

	if err := Validate(data); err != nil {
		return "", fmt.Errorf("openapigen: generated an invalid document, not writing it — this is a bug in sgo itself, please report it: %w", err)
	}

	path := Path(projectDir, cfg.OpenAPI.Format)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", fmt.Errorf("openapigen: creating %s: %w", DocsDir, err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("openapigen: writing %s: %w", path, err)
	}

	return path, nil
}
