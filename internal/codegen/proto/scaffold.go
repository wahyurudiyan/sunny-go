package proto

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sgotemplate "github.com/wahyurudiyan/sunny-go/internal/template"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// GenerateStub writes contract/pb/<name>.proto under protoDir from the
// starter service+CRUD template. It refuses to overwrite an existing
// file — editing a proto is the user's job once it exists.
func GenerateStub(protoDir, name, module string) error {
	path := filepath.Join(protoDir, name+".proto")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", path)
	}

	data := struct {
		Name   string
		Entity string
		Module string
	}{
		Name:   name,
		Entity: title(name),
		Module: module,
	}

	engine := sgotemplate.New(templatesFS)
	if err := engine.Render("templates/service.proto.tmpl", data, path); err != nil {
		return err
	}

	return nil
}

func title(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
