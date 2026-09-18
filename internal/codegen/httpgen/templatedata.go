package httpgen

import (
	"strings"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

// templateRoute is the per-route view every framework template renders
// from. TitleVerb ("Get"/"Post"/"Put"/"Delete") is only used by chi,
// whose convenience methods are named that way; gin and echo use Verb
// directly, since their route methods are named GET/POST/PUT/DELETE.
type templateRoute struct {
	Verb      string
	TitleVerb string
	Path      string
	HasBody   bool
	HasID     bool
	Name      string
	Input     string
	Output    string
}

// routesData is what every <framework>_routes.go.tmpl renders from.
type routesData struct {
	Entity           string
	DomainPkg        string
	DomainImportPath string
	PortInImportPath string
	Routes           []templateRoute
}

func buildRoutesData(f *sgoproto.File, p core.Paths, idPlaceholder string) routesData {
	routes := BuildRoutes(f, p.Entity)

	data := routesData{
		Entity:           entityTitle(p.Entity),
		DomainPkg:        p.Entity,
		DomainImportPath: p.DomainImportPath(),
		PortInImportPath: p.PortInImportPath(),
	}

	for _, r := range routes {
		data.Routes = append(data.Routes, templateRoute{
			Verb:      r.Verb,
			TitleVerb: titleVerb(r.Verb),
			Path:      r.FullPath(idPlaceholder),
			HasBody:   r.HasBody,
			HasID:     r.HasID,
			Name:      r.Method.Name,
			Input:     r.Method.Input,
			Output:    r.Method.Output,
		})
	}

	return data
}

func titleVerb(verb string) string {
	if verb == "" {
		return verb
	}
	return strings.ToUpper(verb[:1]) + strings.ToLower(verb[1:])
}

func entityTitle(name string) string {
	if name == "" {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}
