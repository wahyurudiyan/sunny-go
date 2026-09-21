package wizard

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestWizard(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Wizard Suite")
}
