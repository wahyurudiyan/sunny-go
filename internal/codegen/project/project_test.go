package project_test

import (
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/project"
	"github.com/wahyurudiyan/sunny-go/internal/config"
)

var _ = Describe("ValidateName", func() {
	Context("when the name is empty", func() {
		It("returns an error", func() {
			Expect(project.ValidateName("")).To(MatchError(ContainSubstring("cannot be empty")))
		})
	})

	Context("when the name contains a path separator", func() {
		It("returns an error", func() {
			Expect(project.ValidateName("my/service")).To(HaveOccurred())
		})
	})

	Context("when the name contains a space", func() {
		It("returns an error", func() {
			Expect(project.ValidateName("my service")).To(HaveOccurred())
		})
	})

	Context("when the name is well-formed", func() {
		It("returns no error", func() {
			Expect(project.ValidateName("myservice")).To(Succeed())
		})
	})
})

// BuildOptions is what `sgo init` (parsing comma-separated flags first)
// and the web UI's POST /api/init (already holding typed JSON arrays)
// both call, so a name/engine rejected here is rejected identically by
// both surfaces.
var _ = Describe("BuildOptions", func() {
	It("defaults module to name when module is empty", func() {
		opts, err := project.BuildOptions("shop", "", config.HTTPFrameworkGin, config.PersistenceModeORM, nil, nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(opts.Module).To(Equal("shop"))
	})

	It("keeps an explicitly given module", func() {
		opts, err := project.BuildOptions("shop", "github.com/acme/shop", config.HTTPFrameworkGin, config.PersistenceModeORM, nil, nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(opts.Module).To(Equal("github.com/acme/shop"))
	})

	It("rejects an invalid name before anything else", func() {
		_, err := project.BuildOptions("bad name", "", config.HTTPFrameworkGin, config.PersistenceModeORM, nil, nil, nil)
		Expect(err).To(MatchError(ContainSubstring("invalid characters")))
	})

	It("rejects an invalid HTTP framework", func() {
		_, err := project.BuildOptions("shop", "", config.HTTPFramework("fiber"), config.PersistenceModeORM, nil, nil, nil)
		Expect(err).To(MatchError(ContainSubstring("invalid httpFramework")))
	})

	It("rejects an invalid persistence engine", func() {
		_, err := project.BuildOptions("shop", "", config.HTTPFrameworkGin, config.PersistenceModeORM, []config.PersistenceEngine{"oracle"}, nil, nil)
		Expect(err).To(MatchError(ContainSubstring("invalid persistence engine")))
	})

	It("carries persistence, cache, and search selections through", func() {
		opts, err := project.BuildOptions("shop", "", config.HTTPFrameworkEcho, config.PersistenceModeSelfManaged,
			[]config.PersistenceEngine{config.PersistenceEnginePostgres},
			[]config.CacheEngine{config.CacheEngineRedis},
			[]config.SearchEngine{config.SearchEngineElasticsearch},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(opts.HTTPFramework).To(Equal(config.HTTPFrameworkEcho))
		Expect(opts.Persistence.Mode).To(Equal(config.PersistenceModeSelfManaged))
		Expect(opts.Persistence.Engines).To(ConsistOf(config.PersistenceEnginePostgres))
		Expect(opts.Cache).To(ConsistOf(config.CacheEngineRedis))
		Expect(opts.Search).To(ConsistOf(config.SearchEngineElasticsearch))
	})
})

var _ = Describe("Scaffold", func() {
	var (
		parentDir string
		destDir   string
	)

	BeforeEach(func() {
		var err error
		parentDir, err = os.MkdirTemp("", "sgo-project-test-*")
		Expect(err).NotTo(HaveOccurred())
		destDir = filepath.Join(parentDir, "demo")

		DeferCleanup(func() {
			Expect(os.RemoveAll(parentDir)).To(Succeed())
		})
	})

	Context("when the destination directory already exists", func() {
		It("returns an error and does not touch it", func() {
			Expect(os.MkdirAll(destDir, 0755)).To(Succeed())

			err := project.Scaffold(destDir, project.Options{Name: "demo", Module: "demo"})

			Expect(err).To(MatchError(ContainSubstring("already exists")))
			entries, err := os.ReadDir(destDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).To(BeEmpty())
		})
	})

	Context("with only Postgres selected", func() {
		var opts project.Options

		BeforeEach(func() {
			opts = project.Options{
				Name:          "demo",
				Module:        "demo",
				HTTPFramework: config.HTTPFrameworkGin,
				Persistence: config.Persistence{
					Mode:    config.PersistenceModeORM,
					Engines: []config.PersistenceEngine{config.PersistenceEnginePostgres},
				},
			}

			Expect(project.Scaffold(destDir, opts)).To(Succeed())
		})

		It("writes the core scaffold files", func() {
			for _, f := range []string{"go.mod", "Makefile", filepath.Join("docker", "Dockerfile"), filepath.Join("cmd", "demo", "main.go"), "sgo.yaml"} {
				Expect(filepath.Join(destDir, f)).To(BeAnExistingFile(), f)
			}
		})

		It("creates a persistence adapter directory only for the selected engine", func() {
			Expect(filepath.Join(destDir, "internal", "adapter", "out", "persistence", "postgres")).To(BeADirectory())
			Expect(filepath.Join(destDir, "internal", "adapter", "out", "persistence", "mysql")).NotTo(BeADirectory())
			Expect(filepath.Join(destDir, "internal", "adapter", "out", "persistence", "mongo")).NotTo(BeADirectory())
		})

		It("creates the HTTP adapter directory for the selected framework only", func() {
			Expect(filepath.Join(destDir, "internal", "adapter", "in", "http", "gin")).To(BeADirectory())
			Expect(filepath.Join(destDir, "internal", "adapter", "in", "http", "echo")).NotTo(BeADirectory())
			Expect(filepath.Join(destDir, "internal", "adapter", "in", "http", "chi")).NotTo(BeADirectory())
		})

		It("writes an sgo.yaml that round-trips to the same selections", func() {
			got, err := config.Load(destDir)
			Expect(err).NotTo(HaveOccurred())

			Expect(got.Module).To(Equal(opts.Module))
			Expect(got.HTTPFramework).To(Equal(opts.HTTPFramework))
			Expect(got.Persistence).To(Equal(opts.Persistence))
		})

		It("writes a docker-compose.yml with exactly one service: postgres", func() {
			data, err := os.ReadFile(filepath.Join(destDir, "docker", "docker-compose.yml"))
			Expect(err).NotTo(HaveOccurred())

			var doc struct {
				Services map[string]any `yaml:"services"`
			}
			Expect(yaml.Unmarshal(data, &doc)).To(Succeed())

			Expect(doc.Services).To(HaveLen(1))
			Expect(doc.Services).To(HaveKey("postgres"))
		})

		It("produces a project that builds", func() {
			cmd := exec.Command("go", "build", "./...")
			cmd.Dir = destDir

			out, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(out))
		})
	})

	Context("with Postgres, Redis, and Elasticsearch selected", func() {
		BeforeEach(func() {
			opts := project.Options{
				Name:          "demo",
				Module:        "demo",
				HTTPFramework: config.HTTPFrameworkEcho,
				Persistence: config.Persistence{
					Mode:    config.PersistenceModeORM,
					Engines: []config.PersistenceEngine{config.PersistenceEnginePostgres},
				},
				Cache:  []config.CacheEngine{config.CacheEngineRedis},
				Search: []config.SearchEngine{config.SearchEngineElasticsearch},
			}

			Expect(project.Scaffold(destDir, opts)).To(Succeed())
		})

		It("writes a docker-compose.yml with one service per selected datastore", func() {
			data, err := os.ReadFile(filepath.Join(destDir, "docker", "docker-compose.yml"))
			Expect(err).NotTo(HaveOccurred())

			var doc struct {
				Services map[string]any `yaml:"services"`
			}
			Expect(yaml.Unmarshal(data, &doc)).To(Succeed())

			Expect(doc.Services).To(HaveLen(3))
			Expect(doc.Services).To(HaveKey("postgres"))
			Expect(doc.Services).To(HaveKey("redis"))
			Expect(doc.Services).To(HaveKey("elasticsearch"))
		})

		It("creates cache and search adapter directories for the selected engines", func() {
			Expect(filepath.Join(destDir, "internal", "adapter", "out", "cache", "redis")).To(BeADirectory())
			Expect(filepath.Join(destDir, "internal", "adapter", "out", "search", "elasticsearch")).To(BeADirectory())
		})
	})

	Context("with no datastores selected", func() {
		It("still writes a valid, empty docker-compose.yml", func() {
			opts := project.Options{
				Name:          "demo",
				Module:        "demo",
				HTTPFramework: config.HTTPFrameworkChi,
				Persistence:   config.Persistence{Mode: config.PersistenceModeSelfManaged},
			}
			Expect(project.Scaffold(destDir, opts)).To(Succeed())

			data, err := os.ReadFile(filepath.Join(destDir, "docker", "docker-compose.yml"))
			Expect(err).NotTo(HaveOccurred())

			var doc struct {
				Services map[string]any `yaml:"services"`
			}
			Expect(yaml.Unmarshal(data, &doc)).To(Succeed())
			Expect(doc.Services).To(BeEmpty())
		})
	})

	Context("when no module is given explicitly", func() {
		It("still scaffolds using whatever module the caller resolved", func() {
			opts := project.Options{
				Name:          "demo",
				Module:        "demo",
				HTTPFramework: config.HTTPFrameworkGin,
				Persistence:   config.Persistence{Mode: config.PersistenceModeORM},
			}
			Expect(project.Scaffold(destDir, opts)).To(Succeed())

			data, err := os.ReadFile(filepath.Join(destDir, "go.mod"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(ContainSubstring("module demo"))
		})
	})
})
