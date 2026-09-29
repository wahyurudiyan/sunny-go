// Package codegen orchestrates `sgo generate code`: compiling a proto
// file and driving every generator in its subpackages (proto, wiregen,
// core, httpgen, grpcgen, memgen, bootstrap) to produce contract/gen,
// the DDD domain/application/infrastructure layers (ARCHITECTURE.md
// §17), the HTTP/gRPC adapters, a runnable in-memory repository, and the
// composition root for one entity.
package codegen

import (
	"fmt"
	"os"
	"path/filepath"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/bootstrap"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/cachegen"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/gengo"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/grpcgen"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/httpgen"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/memgen"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/mongogen"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/searchgen"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/sqlgen"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/wiregen"
	"github.com/wahyurudiyan/sunny-go/internal/config"
)

// GenerateCode reads projectDir/contract/pb/<name>.proto and (re)generates
// everything that derives from it: contract/gen, the domain aggregate
// and its repository port, the application layer's CQRS command/query
// DTOs and service (safely — see core.GenerateApplicationService), the
// wire↔domain and wire↔application mappers, the HTTP routes for the
// project's chosen framework, the gRPC server adapter, a default
// in-memory repository, and the bootstrap composition root (rewired for
// every service on record, not just this one). It then runs
// `go mod tidy` in projectDir so newly imported dependencies are picked
// up, and records name in sgo.yaml's services list.
func GenerateCode(projectDir, name string, cfg *config.Config) error {
	protoDir := filepath.Join(projectDir, "contract", "pb")
	protoRel := name + ".proto"

	if _, err := os.Stat(filepath.Join(protoDir, protoRel)); os.IsNotExist(err) {
		return fmt.Errorf("proto file not found: contract/pb/%s. Run `sgo generate proto %s` first", protoRel, name)
	}

	fd, err := sgoproto.Compile(protoDir, protoRel)
	if err != nil {
		return err
	}

	file, err := sgoproto.Build(fd)
	if err != nil {
		return err
	}

	if err := wiregen.Generate(fd, filepath.Join(projectDir, "contract", "gen", name)); err != nil {
		return err
	}

	p := core.Paths{Module: cfg.Module, Entity: name}

	domainDir := filepath.Join(projectDir, "internal", "domain", name)
	if err := core.GenerateEventKernel(filepath.Join(projectDir, "internal", "domain", "event")); err != nil {
		return err
	}
	if err := core.GenerateMaskKernel(filepath.Join(projectDir, "internal", "domain", "mask")); err != nil {
		return err
	}
	if err := core.GenerateAggregate(file, fd, p, domainDir); err != nil {
		return err
	}
	if err := core.GenerateAggregateRepositoryPort(file, fd, p, domainDir); err != nil {
		return err
	}
	if err := core.GenerateDomainErrors(fd, p, domainDir); err != nil {
		return err
	}

	appDir := filepath.Join(projectDir, "internal", "application", name)
	if err := core.GenerateCommandsAndQueries(file, fd, p, appDir); err != nil {
		return err
	}
	if err := core.GenerateEventPublisher(p, filepath.Join(projectDir, "internal", "application", "ports")); err != nil {
		return err
	}
	if err := core.GenerateApplicationService(file, fd, p, appDir); err != nil {
		return err
	}

	if err := core.GenerateInfraMapper(file, p, filepath.Join(projectDir, "internal", "infrastructure", "transport")); err != nil {
		return err
	}

	httpDir := filepath.Join(projectDir, "internal", "infrastructure", "transport", "http", string(cfg.HTTPFramework))
	if err := httpgen.GenerateServer(cfg.HTTPFramework, httpDir); err != nil {
		return err
	}
	if err := httpgen.GenerateRoutes(cfg.HTTPFramework, fd, file, p, httpDir); err != nil {
		return err
	}

	if err := grpcgen.Generate(file, p, filepath.Join(projectDir, "internal", "infrastructure", "transport", "grpc")); err != nil {
		return err
	}

	if err := memgen.Generate(file, fd, p, filepath.Join(projectDir, "internal", "infrastructure", "persistence", "memory")); err != nil {
		return err
	}

	if err := generateRealPersistence(cfg, file, fd, p, projectDir); err != nil {
		return err
	}

	if err := generateCacheAndSearch(cfg, projectDir); err != nil {
		return err
	}

	addService(cfg, name)

	if err := bootstrap.Generate(cfg, filepath.Join(projectDir, "internal", "infrastructure", "bootstrap")); err != nil {
		return err
	}

	if err := cfg.Save(projectDir); err != nil {
		return err
	}

	return gengo.TidyModule(projectDir)
}

// generateRealPersistence writes the real repository adapter for the
// project's first selected persistence engine, if any — on top of the
// in-memory one, which is always generated regardless (Decision #15).
// bootstrap.Generate decides which one wire_gen.go actually uses.
func generateRealPersistence(cfg *config.Config, file *sgoproto.File, fd protoreflect.FileDescriptor, p core.Paths, projectDir string) error {
	if len(cfg.Persistence.Engines) == 0 {
		return nil
	}

	engine := cfg.Persistence.Engines[0]
	switch {
	case sqlgen.Supports(engine):
		destDir := filepath.Join(projectDir, "internal", "infrastructure", "persistence", string(engine))
		return sqlgen.Generate(engine, cfg.Persistence.Mode, file, fd, p, destDir)
	case engine == config.PersistenceEngineMongo:
		destDir := filepath.Join(projectDir, "internal", "infrastructure", "persistence", "mongo")
		return mongogen.Generate(file, fd, p, destDir)
	default:
		return fmt.Errorf("no persistence generator for engine %q", engine)
	}
}

// generateCacheAndSearch writes the project-scoped (not per-entity)
// cache/search ports and their adapters if sgo.yaml selects them.
// Idempotent — safe to call on every `sgo generate code` run regardless
// of which entity triggered it.
func generateCacheAndSearch(cfg *config.Config, projectDir string) error {
	portDir := filepath.Join(projectDir, "internal", "core", "port", "out")

	for _, c := range cfg.Cache {
		if c != config.CacheEngineRedis {
			continue
		}
		if err := cachegen.GeneratePort(portDir); err != nil {
			return err
		}
		destDir := filepath.Join(projectDir, "internal", "adapter", "out", "cache", "redis")
		if err := cachegen.GenerateRedis(destDir); err != nil {
			return err
		}
	}

	for _, s := range cfg.Search {
		if s != config.SearchEngineElasticsearch {
			continue
		}
		if err := searchgen.GeneratePort(portDir); err != nil {
			return err
		}
		destDir := filepath.Join(projectDir, "internal", "adapter", "out", "search", "elasticsearch")
		if err := searchgen.GenerateElasticsearch(cfg.Module, destDir); err != nil {
			return err
		}
	}

	return nil
}

func addService(cfg *config.Config, name string) {
	for _, s := range cfg.Services {
		if s == name {
			return
		}
	}
	cfg.Services = append(cfg.Services, name)
}
