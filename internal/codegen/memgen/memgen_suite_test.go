package memgen_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMemGen(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MemGen Suite")
}
