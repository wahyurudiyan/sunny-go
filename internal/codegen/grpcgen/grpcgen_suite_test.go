package grpcgen_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGRPCGen(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GRPCGen Suite")
}
