package openapigen_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOpenAPIGen(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OpenAPIGen Suite")
}
