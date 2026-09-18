package mongogen_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMongoGen(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MongoGen Suite")
}
