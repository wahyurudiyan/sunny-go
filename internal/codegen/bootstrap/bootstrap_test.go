package bootstrap_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/bootstrap"
	"github.com/wahyurudiyan/sunny-go/internal/config"
)

func baseCfg() *config.Config {
	return &config.Config{
		Module:        "demo",
		HTTPFramework: config.HTTPFrameworkGin,
		Persistence:   config.Persistence{Mode: config.PersistenceModeORM},
	}
}

var _ = Describe("Generate", func() {
	var (
		root, destDir string
	)

	BeforeEach(func() {
		var err error
		root, err = os.MkdirTemp("", "sgo-bootstrap-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		destDir = filepath.Join(root, "internal", "infrastructure", "bootstrap")
	})

	Context("with no services yet (fresh `sgo init`)", func() {
		It("still writes a valid, buildable Run that starts empty servers", func() {
			cfg := baseCfg()
			Expect(bootstrap.Generate(cfg, destDir)).To(Succeed())

			content, err := os.ReadFile(filepath.Join(destDir, "wire_gen.go"))
			Expect(err).NotTo(HaveOccurred())

			Expect(string(content)).NotTo(ContainSubstring("grpcadapter"), "no per-entity imports needed with zero services")
			Expect(string(content)).NotTo(ContainSubstring(`memory "demo`))
			Expect(string(content)).To(ContainSubstring("func Run(ctx context.Context) error"))

			assertValidGo(filepath.Join(destDir, "wire_gen.go"))
		})
	})

	Context("with services on record and no persistence engine selected", func() {
		It("wires every service to the in-memory repository", func() {
			cfg := baseCfg()
			cfg.Services = []string{"user", "order"}
			Expect(bootstrap.Generate(cfg, destDir)).To(Succeed())

			content, err := os.ReadFile(filepath.Join(destDir, "wire_gen.go"))
			Expect(err).NotTo(HaveOccurred())

			for _, entity := range []struct{ Title, Lower string }{{"User", "user"}, {"Order", "order"}} {
				Expect(string(content)).To(ContainSubstring("memory.New" + entity.Title + "Repository()"))
				Expect(string(content)).To(ContainSubstring(entity.Lower + "app.New" + entity.Title + "Service("))
				Expect(string(content)).To(ContainSubstring("httpadapter.Register" + entity.Title + "Routes("))
				Expect(string(content)).To(ContainSubstring("grpcadapter.Register" + entity.Title + "ServiceServer("))
			}
			Expect(string(content)).To(ContainSubstring("publisher := ports.NoopEventPublisher{}"))

			assertValidGo(filepath.Join(destDir, "wire_gen.go"))
		})
	})

	Context("when a persistence engine is selected", func() {
		It("wires postgres in ORM mode: Connect(), AutoMigrate(db), and New<Entity>Repository(db)", func() {
			cfg := baseCfg()
			cfg.Services = []string{"user"}
			cfg.Persistence.Engines = []config.PersistenceEngine{config.PersistenceEnginePostgres}
			Expect(bootstrap.Generate(cfg, destDir)).To(Succeed())

			content, err := os.ReadFile(filepath.Join(destDir, "wire_gen.go"))
			Expect(err).NotTo(HaveOccurred())

			Expect(string(content)).To(ContainSubstring(`postgres "demo/internal/infrastructure/persistence/postgres"`))
			Expect(string(content)).To(ContainSubstring("db, err := postgres.Connect()"))
			Expect(string(content)).To(ContainSubstring("postgres.AutoMigrate(db)"))
			Expect(string(content)).To(ContainSubstring("postgres.NewUserRepository(db)"))
			Expect(string(content)).NotTo(ContainSubstring("memory."))

			assertValidGo(filepath.Join(destDir, "wire_gen.go"))
		})

		It("wires postgres in self-managed mode with a ctx-taking Connect/AutoMigrate", func() {
			cfg := baseCfg()
			cfg.Persistence.Mode = config.PersistenceModeSelfManaged
			cfg.Services = []string{"user"}
			cfg.Persistence.Engines = []config.PersistenceEngine{config.PersistenceEnginePostgres}
			Expect(bootstrap.Generate(cfg, destDir)).To(Succeed())

			content, err := os.ReadFile(filepath.Join(destDir, "wire_gen.go"))
			Expect(err).NotTo(HaveOccurred())

			Expect(string(content)).To(ContainSubstring("db, err := postgres.Connect(ctx)"))
			Expect(string(content)).To(ContainSubstring("postgres.AutoMigrate(ctx, db)"))

			assertValidGo(filepath.Join(destDir, "wire_gen.go"))
		})

		It("wires mongo with no AutoMigrate call (schemaless)", func() {
			cfg := baseCfg()
			cfg.Services = []string{"user"}
			cfg.Persistence.Engines = []config.PersistenceEngine{config.PersistenceEngineMongo}
			Expect(bootstrap.Generate(cfg, destDir)).To(Succeed())

			content, err := os.ReadFile(filepath.Join(destDir, "wire_gen.go"))
			Expect(err).NotTo(HaveOccurred())

			Expect(string(content)).To(ContainSubstring("db, err := mongo.Connect(ctx)"))
			Expect(string(content)).To(ContainSubstring("mongo.NewUserRepository(db)"))
			Expect(string(content)).NotTo(ContainSubstring("AutoMigrate"))

			assertValidGo(filepath.Join(destDir, "wire_gen.go"))
		})
	})

	Context("when cache and search are selected", func() {
		It("connects both clients without wiring them into any service", func() {
			cfg := baseCfg()
			cfg.Services = []string{"user"}
			cfg.Cache = []config.CacheEngine{config.CacheEngineRedis}
			cfg.Search = []config.SearchEngine{config.SearchEngineElasticsearch}
			Expect(bootstrap.Generate(cfg, destDir)).To(Succeed())

			content, err := os.ReadFile(filepath.Join(destDir, "wire_gen.go"))
			Expect(err).NotTo(HaveOccurred())

			Expect(string(content)).To(ContainSubstring(`cacheadapter "demo/internal/adapter/out/cache/redis"`))
			Expect(string(content)).To(ContainSubstring("cacheadapter.Connect(ctx)"))
			Expect(string(content)).To(ContainSubstring(`searchadapter "demo/internal/adapter/out/search/elasticsearch"`))
			Expect(string(content)).To(ContainSubstring("searchadapter.Connect()"))
			Expect(string(content)).NotTo(ContainSubstring("userapp.NewUserService(cache"), "cache/search must not appear in the service constructor call")

			assertValidGo(filepath.Join(destDir, "wire_gen.go"))
		})
	})

	Context("per HTTP framework", func() {
		It("uses the gin server's Engine field", func() {
			cfg := baseCfg()
			cfg.Services = []string{"user"}
			Expect(bootstrap.Generate(cfg, destDir)).To(Succeed())
			content, err := os.ReadFile(filepath.Join(destDir, "wire_gen.go"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("httpServer.Engine"))
		})

		It("uses the echo server's Echo field", func() {
			cfg := baseCfg()
			cfg.HTTPFramework = config.HTTPFrameworkEcho
			cfg.Services = []string{"user"}
			Expect(bootstrap.Generate(cfg, destDir)).To(Succeed())
			content, err := os.ReadFile(filepath.Join(destDir, "wire_gen.go"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("httpServer.Echo"))
		})

		It("uses the chi server's Router field", func() {
			cfg := baseCfg()
			cfg.HTTPFramework = config.HTTPFrameworkChi
			cfg.Services = []string{"user"}
			Expect(bootstrap.Generate(cfg, destDir)).To(Succeed())
			content, err := os.ReadFile(filepath.Join(destDir, "wire_gen.go"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("httpServer.Router"))
		})
	})
})

func assertValidGo(path string) {
	GinkgoHelper()

	src, err := os.ReadFile(path)
	Expect(err).NotTo(HaveOccurred())

	_, err = parser.ParseFile(token.NewFileSet(), path, src, parser.AllErrors)
	Expect(err).NotTo(HaveOccurred())
}
