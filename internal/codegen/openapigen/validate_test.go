package openapigen_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/openapigen"
)

var _ = Describe("Validate", func() {
	DescribeTable("accepts a well-formed document",
		func(doc string) {
			Expect(openapigen.Validate([]byte(doc))).To(Succeed())
		},
		Entry("3.0 JSON", `{"openapi":"3.0.3","info":{"title":"t","version":"1"},"paths":{}}`),
		Entry("3.1 JSON", `{"openapi":"3.1.0","info":{"title":"t","version":"1"},"paths":{}}`),
		Entry("3.0 YAML", "openapi: 3.0.3\ninfo:\n  title: t\n  version: \"1\"\npaths: {}\n"),
		Entry("3.1 YAML", "openapi: 3.1.0\ninfo:\n  title: t\n  version: \"1\"\npaths: {}\n"),
	)

	It("rejects a document that's missing a required field, with a real schema-validation error", func() {
		// info.version is required by the meta-schema.
		err := openapigen.Validate([]byte(`{"openapi":"3.0.3","info":{"title":"t"},"paths":{}}`))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("version"))
	})

	It("rejects a document with no openapi field", func() {
		err := openapigen.Validate([]byte(`{"info":{"title":"t","version":"1"},"paths":{}}`))
		Expect(err).To(MatchError(ContainSubstring(`no top-level "openapi" field`)))
	})

	It("rejects a document declaring an unsupported openapi version", func() {
		err := openapigen.Validate([]byte(`{"openapi":"2.0","info":{"title":"t","version":"1"},"paths":{}}`))
		Expect(err).To(MatchError(ContainSubstring("unsupported")))
	})

	It("rejects a document whose root isn't an object", func() {
		err := openapigen.Validate([]byte(`["not an object"]`))
		Expect(err).To(MatchError(ContainSubstring("not an object")))
	})

	It("rejects input that's neither valid JSON nor valid YAML", func() {
		err := openapigen.Validate([]byte("{not: valid: anything:"))
		Expect(err).To(HaveOccurred())
	})

	It("rejects an empty document", func() {
		err := openapigen.Validate([]byte(""))
		Expect(err).To(MatchError(ContainSubstring("empty")))
	})
})
