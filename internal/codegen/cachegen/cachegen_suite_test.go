package cachegen_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCacheGen(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CacheGen Suite")
}
