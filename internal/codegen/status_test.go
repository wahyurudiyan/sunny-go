package codegen_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/codegen"
)

// The one implementation behind both `sgo list services`' printout and
// the web UI's service dashboard (Phase 6) — see internal/commands/list.go
// and internal/webui.
var _ = Describe("Status", func() {
	var dir string

	BeforeEach(func() {
		var err error
		dir, err = os.MkdirTemp("", "sgo-status-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })
	})

	It("reports every stage missing for a name with no generated output", func() {
		s := codegen.Status(dir, "user")
		Expect(s).To(Equal(codegen.ServiceStatus{Name: "user"}))
	})

	It("reports each stage true once its file/directory exists", func() {
		Expect(os.MkdirAll(filepath.Join(dir, "contract", "pb"), 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(dir, "contract", "pb", "user.proto"), []byte("x"), 0644)).To(Succeed())

		Expect(os.MkdirAll(filepath.Join(dir, "contract", "gen", "user"), 0755)).To(Succeed())

		Expect(os.MkdirAll(filepath.Join(dir, "internal", "domain", "user"), 0755)).To(Succeed())

		Expect(os.MkdirAll(filepath.Join(dir, "internal", "application", "user"), 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(dir, "internal", "application", "user", "service.go"), []byte("x"), 0644)).To(Succeed())

		s := codegen.Status(dir, "user")
		Expect(s).To(Equal(codegen.ServiceStatus{
			Name:         "user",
			Proto:        true,
			ContractGen:  true,
			DomainEntity: true,
			ServiceImpl:  true,
		}))
	})
})
