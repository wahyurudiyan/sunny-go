package envsource_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEnvSource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "EnvSource Suite")
}
