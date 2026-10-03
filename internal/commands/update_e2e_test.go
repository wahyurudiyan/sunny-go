package commands_test

// End-to-end coverage for `sgo update` (PLAN.md Phase 19), driving the
// real compiled binary the same way e2e_test.go does for the rest of
// the command surface. sgo's own module has no published tags yet
// (ARCHITECTURE.md §23's stated limitation), so the "no tagged
// releases" path is exercised for real rather than simulated; the
// success-path install itself is covered at the internal/selfupdate
// package level against a real, already-tagged module instead, since
// there's no real sgo tag to install here yet.

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("sgo update", func() {
	It("reports no tagged releases for the bare command, against the real (untagged) module", func() {
		out, err := runSgo(".", "update")
		Expect(err).NotTo(HaveOccurred(), out)
		Expect(out).To(ContainSubstring("No tagged releases found"))
	})

	It("--check reports the same, without attempting an install", func() {
		out, err := runSgo(".", "update", "--check")
		Expect(err).NotTo(HaveOccurred(), out)
		Expect(out).To(ContainSubstring("No tagged releases found"))
	})

	It("--check --version reports the requested version against the current one, without installing", func() {
		out, err := runSgo(".", "update", "--check", "--version", "0.2.0")
		Expect(err).NotTo(HaveOccurred(), out)
		Expect(out).To(ContainSubstring("Requested version: v0.2.0"))
		Expect(out).To(ContainSubstring("currently running"))
	})

	It("rejects a syntactically invalid --version before attempting anything", func() {
		out, err := runSgo(".", "update", "--version", "not-a-version")
		Expect(err).To(HaveOccurred())
		Expect(out).To(ContainSubstring("is not a valid version"))
	})

	It("--version against the real module surfaces the real go install failure rather than hanging or swallowing it", func() {
		out, err := runSgo(".", "update", "--version", "v99.99.99")
		Expect(err).To(HaveOccurred())
		Expect(out).To(ContainSubstring("go install"))
	})
})

var _ = Describe("sgo --version", func() {
	It("still reports the build-time Version const, unaffected by the new command", func() {
		out, err := runSgo(".", "--version")
		Expect(err).NotTo(HaveOccurred(), out)
		Expect(out).To(ContainSubstring("sgo version"))
	})
})
