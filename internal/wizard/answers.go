package wizard

import (
	"github.com/wahyurudiyan/sunny-go/internal/codegen/project"
	"github.com/wahyurudiyan/sunny-go/internal/config"
)

// answers holds the raw values collected from the interactive form.
// Separated from the form itself (wizard.go) so the assembly/validation
// logic can be unit tested without driving a real terminal.
type answers struct {
	Name          string
	Module        string
	HTTPFramework string
	Persistence   string
	Datastores    []string
	EnableRedis   bool
	EnableSearch  bool
}

// assemble turns the raw form answers into validated project.Options.
func (a answers) assemble() (*project.Options, error) {
	if err := project.ValidateName(a.Name); err != nil {
		return nil, err
	}

	module := a.Module
	if module == "" {
		module = a.Name
	}

	engines := make([]config.PersistenceEngine, 0, len(a.Datastores))
	for _, d := range a.Datastores {
		engines = append(engines, config.PersistenceEngine(d))
	}

	var cache []config.CacheEngine
	if a.EnableRedis {
		cache = []config.CacheEngine{config.CacheEngineRedis}
	}

	var search []config.SearchEngine
	if a.EnableSearch {
		search = []config.SearchEngine{config.SearchEngineElasticsearch}
	}

	opts := &project.Options{
		Name:          a.Name,
		Module:        module,
		HTTPFramework: config.HTTPFramework(a.HTTPFramework),
		Persistence: config.Persistence{
			Mode:    config.PersistenceMode(a.Persistence),
			Engines: engines,
		},
		Cache:  cache,
		Search: search,
	}

	cfg := config.Config{
		Module:        opts.Module,
		HTTPFramework: opts.HTTPFramework,
		Persistence:   opts.Persistence,
		Cache:         opts.Cache,
		Search:        opts.Search,
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return opts, nil
}
