// Package wizard implements the interactive selection UI for `sgo init`
// (ARCHITECTURE.md §10): project name/module, HTTP framework,
// persistence mode, and datastores, collected through an animated
// terminal form (charmbracelet/huh) instead of flags.
package wizard

import (
	"fmt"

	"github.com/charmbracelet/huh"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/project"
	"github.com/wahyurudiyan/sunny-go/internal/config"
)

// Run walks the user through the interactive wizard and returns the
// resulting scaffolding options. defaultName pre-fills the project name
// field when the caller already has one (e.g. `sgo init myservice` with
// no other flags).
func Run(defaultName string) (*project.Options, error) {
	a := answers{
		Name:          defaultName,
		HTTPFramework: string(config.HTTPFrameworkGin),
		Persistence:   string(config.PersistenceModeORM),
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Project name").
				Value(&a.Name).
				Validate(project.ValidateName),
			huh.NewInput().
				Title("Go module path").
				Description("Leave blank to use the project name").
				Placeholder("github.com/you/"+defaultName).
				Value(&a.Module),
		),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("HTTP framework").
				Options(
					huh.NewOption("Gin", string(config.HTTPFrameworkGin)),
					huh.NewOption("Echo", string(config.HTTPFrameworkEcho)),
					huh.NewOption("Chi", string(config.HTTPFrameworkChi)),
				).
				Value(&a.HTTPFramework),
			huh.NewSelect[string]().
				Title("Persistence mode").
				Options(
					huh.NewOption("ORM", string(config.PersistenceModeORM)),
					huh.NewOption("Self-managed", string(config.PersistenceModeSelfManaged)),
				).
				Value(&a.Persistence),
		),
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Datastores").
				Description("Space to toggle, enter to continue").
				Options(
					huh.NewOption("PostgreSQL", string(config.PersistenceEnginePostgres)),
					huh.NewOption("MySQL", string(config.PersistenceEngineMySQL)),
					huh.NewOption("MongoDB", string(config.PersistenceEngineMongo)),
				).
				Value(&a.Datastores),
			huh.NewConfirm().
				Title("Enable Redis cache?").
				Value(&a.EnableRedis),
			huh.NewConfirm().
				Title("Enable Elasticsearch search?").
				Value(&a.EnableSearch),
		),
	).WithTheme(huh.ThemeCharm())

	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("wizard cancelled: %w", err)
	}

	return a.assemble()
}
