// Package codegen orchestrates `sgo generate code`: compiling a proto
// file and driving every generator in its subpackages (proto, wiregen,
// core) to produce contract/gen, the hexagonal core, and the mapper for
// one entity.
package codegen

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/core"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/wiregen"
	"github.com/wahyurudiyan/sunny-go/internal/config"
)

// GenerateCode reads projectDir/contract/pb/<name>.proto and (re)generates
// everything that derives from it: contract/gen, the domain entity, the
// usecase/repository ports, the service skeleton (safely — see
// core.GenerateService), and the wire↔domain mapper. It then runs
// `go mod tidy` in projectDir so the newly imported protobuf/grpc
// dependencies are picked up, and records name in sgo.yaml's services
// list.
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

	if err := core.GenerateDomain(file, p, filepath.Join(projectDir, "internal", "core", "domain", name)); err != nil {
		return err
	}
	if err := core.GenerateUsecasePort(file, p, filepath.Join(projectDir, "internal", "core", "port", "in")); err != nil {
		return err
	}
	if err := core.GenerateRepositoryPort(p, filepath.Join(projectDir, "internal", "core", "port", "out")); err != nil {
		return err
	}
	if err := core.GenerateService(file, p, filepath.Join(projectDir, "internal", "core", "service")); err != nil {
		return err
	}
	if err := core.GenerateMapper(file, p, filepath.Join(projectDir, "internal", "adapter", "mapper")); err != nil {
		return err
	}

	addService(cfg, name)
	if err := cfg.Save(projectDir); err != nil {
		return err
	}

	return tidyModule(projectDir)
}

func addService(cfg *config.Config, name string) {
	for _, s := range cfg.Services {
		if s == name {
			return
		}
	}
	cfg.Services = append(cfg.Services, name)
}

func tidyModule(projectDir string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = projectDir

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go mod tidy failed: %w\n%s", err, out)
	}

	return nil
}
