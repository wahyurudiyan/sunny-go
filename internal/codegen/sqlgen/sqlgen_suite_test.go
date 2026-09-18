package sqlgen_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSQLGen(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SQLGen Suite")
}
