package wizard

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/config"
)

func validAnswers() answers {
	return answers{
		Name:          "demo",
		HTTPFramework: string(config.HTTPFrameworkGin),
		Persistence:   string(config.PersistenceModeORM),
		Datastores:    []string{string(config.PersistenceEnginePostgres)},
		EnableRedis:   true,
		EnableSearch:  true,
	}
}

var _ = Describe("answers.assemble", func() {
	Context("when every field is well-formed", func() {
		It("builds matching project.Options", func() {
			opts, err := validAnswers().assemble()

			Expect(err).NotTo(HaveOccurred())
			Expect(opts.Name).To(Equal("demo"))
			Expect(opts.Module).To(Equal("demo"))
			Expect(opts.HTTPFramework).To(Equal(config.HTTPFrameworkGin))
			Expect(opts.Persistence.Mode).To(Equal(config.PersistenceModeORM))
			Expect(opts.Persistence.Engines).To(ConsistOf(config.PersistenceEnginePostgres))
			Expect(opts.Cache).To(ConsistOf(config.CacheEngineRedis))
			Expect(opts.Search).To(ConsistOf(config.SearchEngineElasticsearch))
		})
	})

	Context("when the module is left blank", func() {
		It("defaults the module to the project name", func() {
			a := validAnswers()
			a.Module = ""

			opts, err := a.assemble()

			Expect(err).NotTo(HaveOccurred())
			Expect(opts.Module).To(Equal("demo"))
		})
	})

	Context("when a module is given explicitly", func() {
		It("uses it instead of the project name", func() {
			a := validAnswers()
			a.Module = "github.com/acme/demo"

			opts, err := a.assemble()

			Expect(err).NotTo(HaveOccurred())
			Expect(opts.Module).To(Equal("github.com/acme/demo"))
		})
	})

	Context("when Redis and Elasticsearch are declined", func() {
		It("leaves cache and search empty rather than nil-vs-empty ambiguous", func() {
			a := validAnswers()
			a.EnableRedis = false
			a.EnableSearch = false

			opts, err := a.assemble()

			Expect(err).NotTo(HaveOccurred())
			Expect(opts.Cache).To(BeEmpty())
			Expect(opts.Search).To(BeEmpty())
		})
	})

	Context("when the project name is invalid", func() {
		It("returns an error without needing to consult config.Validate", func() {
			a := validAnswers()
			a.Name = "bad name"

			_, err := a.assemble()

			Expect(err).To(HaveOccurred())
		})
	})

	Context("when the HTTP framework isn't one of the offered options", func() {
		It("returns an error", func() {
			a := validAnswers()
			a.HTTPFramework = "fiber"

			_, err := a.assemble()

			Expect(err).To(HaveOccurred())
		})
	})
})
