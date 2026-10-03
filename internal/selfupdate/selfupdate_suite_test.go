package selfupdate_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSelfupdate(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Selfupdate Suite")
}
