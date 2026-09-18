package wiregen_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestWiregen(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Wiregen Suite")
}
