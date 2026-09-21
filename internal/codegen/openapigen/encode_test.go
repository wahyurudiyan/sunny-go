package openapigen_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/openapigen"
	"github.com/wahyurudiyan/sunny-go/internal/config"
)

func minimalDoc() *openapigen.Document {
	return &openapigen.Document{
		Info:  openapigen.Info{Title: "demo", Version: "0.1.0"},
		Paths: map[string]*openapigen.PathItem{},
	}
}

var _ = Describe("Encode", func() {
	DescribeTable("stamps the exact openapi version string for each family",
		func(version config.OpenAPIVersion, want string) {
			doc := minimalDoc()
			_, err := openapigen.Encode(doc, version, config.OpenAPIFormatJSON)
			Expect(err).NotTo(HaveOccurred())
			Expect(doc.OpenAPI).To(Equal(want))
		},
		Entry("3.0", config.OpenAPIVersion30, "3.0.3"),
		Entry("3.1", config.OpenAPIVersion31, "3.1.0"),
	)

	It("encodes valid JSON that decodes back to an equivalent document", func() {
		data, err := openapigen.Encode(minimalDoc(), config.OpenAPIVersion30, config.OpenAPIFormatJSON)
		Expect(err).NotTo(HaveOccurred())

		var decoded map[string]any
		Expect(json.Unmarshal(data, &decoded)).To(Succeed())
		Expect(decoded["openapi"]).To(Equal("3.0.3"))
		Expect(decoded["info"]).To(HaveKeyWithValue("title", "demo"))
	})

	It("encodes valid YAML that decodes back to an equivalent document", func() {
		data, err := openapigen.Encode(minimalDoc(), config.OpenAPIVersion31, config.OpenAPIFormatYAML)
		Expect(err).NotTo(HaveOccurred())

		var decoded map[string]any
		Expect(yaml.Unmarshal(data, &decoded)).To(Succeed())
		Expect(decoded["openapi"]).To(Equal("3.1.0"))
	})

	It("rejects an unsupported version", func() {
		_, err := openapigen.Encode(minimalDoc(), "2.0", config.OpenAPIFormatJSON)
		Expect(err).To(MatchError(ContainSubstring("unsupported OpenAPI version")))
	})

	It("rejects an unsupported format", func() {
		_, err := openapigen.Encode(minimalDoc(), config.OpenAPIVersion30, "xml")
		Expect(err).To(MatchError(ContainSubstring("unsupported format")))
	})
})

var _ = Describe("Ext", func() {
	It("returns yaml for the yaml format", func() {
		Expect(openapigen.Ext(config.OpenAPIFormatYAML)).To(Equal("yaml"))
	})

	It("returns json for the json format", func() {
		Expect(openapigen.Ext(config.OpenAPIFormatJSON)).To(Equal("json"))
	})
})
