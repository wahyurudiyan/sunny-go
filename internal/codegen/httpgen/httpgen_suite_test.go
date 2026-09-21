package httpgen_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHTTPGen(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HTTPGen Suite")
}
