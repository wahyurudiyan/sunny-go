package config_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/config"
)

func validConfig() *config.Config {
	return &config.Config{
		Module:        "github.com/acme/myservice",
		HTTPFramework: config.HTTPFrameworkGin,
		Persistence: config.Persistence{
			Mode:    config.PersistenceModeORM,
			Engines: []config.PersistenceEngine{config.PersistenceEnginePostgres},
		},
		Cache:  []config.CacheEngine{config.CacheEngineRedis},
		Search: []config.SearchEngine{config.SearchEngineElasticsearch},
		OpenAPI: config.OpenAPI{
			Version: config.OpenAPIVersion30,
			Format:  config.OpenAPIFormatYAML,
		},
	}
}

var _ = Describe("Config", func() {
	Describe("Validate", func() {
		Context("when the config is well-formed", func() {
			It("returns no error", func() {
				Expect(validConfig().Validate()).To(Succeed())
			})
		})

		Context("when the module path is missing", func() {
			It("returns an error", func() {
				cfg := validConfig()
				cfg.Module = ""

				Expect(cfg.Validate()).To(MatchError(ContainSubstring("module is required")))
			})
		})

		Context("when the HTTP framework is unrecognized", func() {
			It("returns an error naming the invalid value", func() {
				cfg := validConfig()
				cfg.HTTPFramework = "fiber"

				Expect(cfg.Validate()).To(MatchError(ContainSubstring("httpFramework")))
			})
		})

		Context("when the persistence mode is unrecognized", func() {
			It("returns an error naming the invalid value", func() {
				cfg := validConfig()
				cfg.Persistence.Mode = "raw-sql"

				Expect(cfg.Validate()).To(MatchError(ContainSubstring("persistence.mode")))
			})
		})

		Context("when a persistence engine is unrecognized", func() {
			It("returns an error naming the invalid engine", func() {
				cfg := validConfig()
				cfg.Persistence.Engines = []config.PersistenceEngine{"oracle"}

				Expect(cfg.Validate()).To(MatchError(ContainSubstring(`"oracle"`)))
			})
		})

		Context("when a cache engine is unrecognized", func() {
			It("returns an error naming the invalid engine", func() {
				cfg := validConfig()
				cfg.Cache = []config.CacheEngine{"memcached"}

				Expect(cfg.Validate()).To(MatchError(ContainSubstring(`"memcached"`)))
			})
		})

		Context("when a search engine is unrecognized", func() {
			It("returns an error naming the invalid engine", func() {
				cfg := validConfig()
				cfg.Search = []config.SearchEngine{"solr"}

				Expect(cfg.Validate()).To(MatchError(ContainSubstring(`"solr"`)))
			})
		})

		Context("when the OpenAPI version is unrecognized", func() {
			It("returns an error naming the invalid value", func() {
				cfg := validConfig()
				cfg.OpenAPI.Version = "2.0"

				Expect(cfg.Validate()).To(MatchError(ContainSubstring("openapi.version")))
			})
		})

		Context("when the OpenAPI format is unrecognized", func() {
			It("returns an error naming the invalid value", func() {
				cfg := validConfig()
				cfg.OpenAPI.Format = "xml"

				Expect(cfg.Validate()).To(MatchError(ContainSubstring("openapi.format")))
			})
		})
	})

	Describe("Save and Load", func() {
		var dir string

		BeforeEach(func() {
			var err error
			dir, err = os.MkdirTemp("", "sgo-config-test-*")
			Expect(err).NotTo(HaveOccurred())

			DeferCleanup(func() {
				Expect(os.RemoveAll(dir)).To(Succeed())
			})
		})

		Context("when a config is saved and then loaded back", func() {
			It("round-trips to an equal config", func() {
				want := validConfig()

				Expect(want.Save(dir)).To(Succeed())

				got, err := config.Load(dir)
				Expect(err).NotTo(HaveOccurred())
				Expect(got).To(Equal(want))
			})

			It("writes sgo.yaml into the given directory", func() {
				Expect(validConfig().Save(dir)).To(Succeed())

				Expect(filepath.Join(dir, config.FileName)).To(BeAnExistingFile())
			})
		})

		Context("when saving to a directory that doesn't exist", func() {
			It("returns an error", func() {
				missing := filepath.Join(dir, "does-not-exist")

				Expect(validConfig().Save(missing)).To(HaveOccurred())
			})
		})

		Context("when no sgo.yaml exists in the directory", func() {
			It("returns an error", func() {
				_, err := config.Load(dir)

				Expect(err).To(HaveOccurred())
			})
		})
	})
})
