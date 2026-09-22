package openapiui_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOpenAPIUI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OpenAPIUI Suite")
}
