package selfupdate_test

// Exercised against real tools, not mocked, matching the rest of this
// repo's testing standard: Versions/Latest run the actual `go list -m
// -versions -json` against the real Go module proxy (this environment's
// network egress already has to reach it for `go build`/`go mod tidy`
// to work at all), and Install runs a real `go install` of a small,
// already-cached, genuinely tagged module (golang.org/x/tools/cmd/stringer
// — already a dependency of this repo, so its module zip is already in
// the local module cache and the build is fast). sgo's own module
// (selfupdate.Module) has no published tags yet, so the "no versions"
// path is exercised for free against the real thing, not simulated.

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/mod/semver"

	"github.com/wahyurudiyan/sunny-go/internal/selfupdate"
)

var _ = Describe("Versions", func() {
	It("returns every published semver tag, ascending, for a real tagged module", func() {
		versions, err := selfupdate.Versions("github.com/spf13/cobra")
		Expect(err).NotTo(HaveOccurred())
		Expect(versions).NotTo(BeEmpty())
		Expect(versions).To(ContainElement("v1.9.1"))

		for i := 1; i < len(versions); i++ {
			Expect(semver.Compare(versions[i-1], versions[i])).To(BeNumerically("<", 0),
				"expected %s before %s", versions[i-1], versions[i])
		}
	})

	It("returns no versions and no error for sgo's own module, which has no tags yet", func() {
		versions, err := selfupdate.Versions(selfupdate.Module)
		Expect(err).NotTo(HaveOccurred())
		Expect(versions).To(BeEmpty())
	})

	It("errors for a module path that can't be resolved at all", func() {
		_, err := selfupdate.Versions("github.com/this-owner-does-not-exist-anywhere/also-does-not-exist")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("Latest", func() {
	It("returns the highest version Versions() finds", func() {
		versions, err := selfupdate.Versions("github.com/spf13/cobra")
		Expect(err).NotTo(HaveOccurred())

		latest, ok, err := selfupdate.Latest("github.com/spf13/cobra")
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
		Expect(latest).To(Equal(versions[len(versions)-1]))
	})

	It("reports ok=false, no error, for an untagged module", func() {
		_, ok, err := selfupdate.Latest(selfupdate.Module)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeFalse())
	})
})

var _ = Describe("NormalizeVersion", func() {
	It("prefixes a bare version with v", func() {
		v, ok := selfupdate.NormalizeVersion("0.2.0")
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal("v0.2.0"))
	})

	It("leaves an already-prefixed version alone", func() {
		v, ok := selfupdate.NormalizeVersion("v0.2.0")
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal("v0.2.0"))
	})

	It("rejects an empty string", func() {
		_, ok := selfupdate.NormalizeVersion("")
		Expect(ok).To(BeFalse())
	})

	It("rejects something that isn't valid semver", func() {
		_, ok := selfupdate.NormalizeVersion("not-a-version")
		Expect(ok).To(BeFalse())
	})
})

var _ = Describe("Install and BinDir", func() {
	var gobin string

	BeforeEach(func() {
		var err error
		gobin, err = os.MkdirTemp("", "sgo-selfupdate-gobin-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(gobin)).To(Succeed()) })

		prevGOBIN, hadGOBIN := os.LookupEnv("GOBIN")
		Expect(os.Setenv("GOBIN", gobin)).To(Succeed())
		DeferCleanup(func() {
			if hadGOBIN {
				Expect(os.Setenv("GOBIN", prevGOBIN)).To(Succeed())
			} else {
				Expect(os.Unsetenv("GOBIN")).To(Succeed())
			}
		})
	})

	It("BinDir reports GOBIN when it's set", func() {
		dir, err := selfupdate.BinDir()
		Expect(err).NotTo(HaveOccurred())
		Expect(dir).To(Equal(gobin))
	})

	It("installs a real, already-cached module into GOBIN", func() {
		var out bytes.Buffer
		err := selfupdate.Install(&out, "golang.org/x/tools/cmd/stringer", "v0.47.0")
		Expect(err).NotTo(HaveOccurred(), out.String())

		binName := "stringer"
		if runtime.GOOS == "windows" {
			binName += ".exe"
		}
		Expect(filepath.Join(gobin, binName)).To(BeAnExistingFile())
	})

	It("surfaces a real install failure instead of swallowing it", func() {
		var out bytes.Buffer
		err := selfupdate.Install(&out, selfupdate.Package, "v99.99.99")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("BinMismatch", func() {
	var gobin string

	BeforeEach(func() {
		var err error
		gobin, err = os.MkdirTemp("", "sgo-selfupdate-gobin-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(gobin)).To(Succeed()) })

		Expect(os.Setenv("GOBIN", gobin)).To(Succeed())
		DeferCleanup(func() { Expect(os.Unsetenv("GOBIN")).To(Succeed()) })
	})

	It("reports no mismatch when the executable is inside GOBIN", func() {
		installDir, running, mismatched, err := selfupdate.BinMismatch(filepath.Join(gobin, "sgo"))
		Expect(err).NotTo(HaveOccurred())
		Expect(installDir).To(Equal(gobin))
		Expect(running).To(Equal(filepath.Join(gobin, "sgo")))
		Expect(mismatched).To(BeFalse())
	})

	It("reports a mismatch when the executable lives somewhere else", func() {
		elsewhere, err := os.MkdirTemp("", "sgo-selfupdate-elsewhere-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(elsewhere)).To(Succeed()) })

		installDir, running, mismatched, err := selfupdate.BinMismatch(filepath.Join(elsewhere, "sgo"))
		Expect(err).NotTo(HaveOccurred())
		Expect(installDir).To(Equal(gobin))
		Expect(running).To(Equal(filepath.Join(elsewhere, "sgo")))
		Expect(mismatched).To(BeTrue())
	})

	It("doesn't error when resolving its own running executable", func() {
		_, _, _, err := selfupdate.BinMismatch("")
		Expect(err).NotTo(HaveOccurred())
	})
})
