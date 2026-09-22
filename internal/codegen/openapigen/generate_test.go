package openapigen_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"

	"github.com/wahyurudiyan/sunny-go/internal/codegen"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/openapigen"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/project"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
	"github.com/wahyurudiyan/sunny-go/internal/config"
)

// scaffoldFullProject runs the real pipeline `sgo init` + `sgo generate
// proto` + `sgo generate code` runs — not just a proto stub — so this
// spec can cross-check the OpenAPI doc against the *actual* generated
// Gin route file, not against internal IR the doc-builder itself
// produced. That's the honest version of "the routes in a generated
// docs/openapi.yaml match the project" (PLAN.md Phase 8 checklist),
// independent of openapigen.Build's own internals.
func scaffoldFullProject(root, entity string) (string, *config.Config) {
	GinkgoHelper()

	opts := project.Options{
		Name:          "demo",
		Module:        "demo",
		HTTPFramework: config.HTTPFrameworkGin,
		Persistence:   config.Persistence{Mode: config.PersistenceModeORM},
		OpenAPI:       config.DefaultOpenAPI(),
	}
	dir := filepath.Join(root, "demo")
	Expect(project.Scaffold(dir, opts)).To(Succeed())
	Expect(proto.GenerateStub(filepath.Join(dir, "contract", "pb"), entity, "demo")).To(Succeed())

	cfg, err := config.Load(dir)
	Expect(err).NotTo(HaveOccurred())
	Expect(codegen.GenerateCode(dir, entity, cfg)).To(Succeed())

	cfg, err = config.Load(dir)
	Expect(err).NotTo(HaveOccurred())
	return dir, cfg
}

var _ = Describe("Generate", func() {
	var root string

	BeforeEach(func() {
		var err error
		root, err = os.MkdirTemp("", "sgo-openapigen-generate-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })
	})

	It("writes docs/openapi.yaml and self-validates it", func() {
		dir, cfg := scaffoldFullProject(root, "product")

		path, err := openapigen.Generate(cfg, dir)
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(Equal(filepath.Join(dir, "docs", "openapi.yaml")))
		Expect(path).To(BeAnExistingFile())

		data, err := os.ReadFile(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(openapigen.Validate(data)).To(Succeed())
	})

	It("writes docs/openapi.json when the project selected the json format", func() {
		dir, cfg := scaffoldFullProject(root, "product")
		cfg.OpenAPI.Format = config.OpenAPIFormatJSON

		path, err := openapigen.Generate(cfg, dir)
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(Equal(filepath.Join(dir, "docs", "openapi.json")))
		Expect(path).To(BeAnExistingFile())
	})

	It("its paths match the real generated Gin route registrations, verb for verb, path for path", func() {
		dir, cfg := scaffoldFullProject(root, "product")

		path, err := openapigen.Generate(cfg, dir)
		Expect(err).NotTo(HaveOccurred())

		data, err := os.ReadFile(path)
		Expect(err).NotTo(HaveOccurred())

		var doc struct {
			Paths map[string]map[string]any `yaml:"paths"`
		}
		Expect(yaml.Unmarshal(data, &doc)).To(Succeed())

		routesSrc, err := os.ReadFile(filepath.Join(dir, "internal", "infrastructure", "transport", "http", "gin", "product_routes_gen.go"))
		Expect(err).NotTo(HaveOccurred())

		verbToGin := map[string]string{"get": "GET", "post": "POST", "put": "PUT", "delete": "DELETE"}

		operationCount := 0
		for oasPath, methods := range doc.Paths {
			ginPath := strings.ReplaceAll(oasPath, "{id}", ":id")
			for verb := range methods {
				ginVerb, ok := verbToGin[verb]
				if !ok {
					continue
				}
				operationCount++
				registration := fmt.Sprintf("engine.%s(%q,", ginVerb, ginPath)
				Expect(string(routesSrc)).To(ContainSubstring(registration),
					"expected the generated route file to register %s %s (OpenAPI doc says it should exist)", ginVerb, ginPath)
			}
		}

		// The CRUD starter template always produces exactly 5 RPCs.
		Expect(operationCount).To(Equal(5))
	})

	It("fails without writing anything when the project has no registered services", func() {
		opts := project.Options{
			Name:          "empty",
			Module:        "empty",
			HTTPFramework: config.HTTPFrameworkGin,
			Persistence:   config.Persistence{Mode: config.PersistenceModeORM},
			OpenAPI:       config.DefaultOpenAPI(),
		}
		dir := filepath.Join(root, "empty")
		Expect(project.Scaffold(dir, opts)).To(Succeed())

		cfg, err := config.Load(dir)
		Expect(err).NotTo(HaveOccurred())

		path, err := openapigen.Generate(cfg, dir)
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(BeAnExistingFile())

		data, err := os.ReadFile(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(openapigen.Validate(data)).To(Succeed(), "an empty paths/components document is still a valid OpenAPI document")
	})
})
