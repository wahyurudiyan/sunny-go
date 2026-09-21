package progress

import (
	"bytes"
	"errors"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("run", func() {
	Context("when stdout isn't a terminal (piped output, CI, scripts)", func() {
		It("prints a single plain line instead of animating", func() {
			var buf bytes.Buffer
			called := false

			err := run(&buf, false, "Generating code", func() error {
				called = true
				return nil
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(called).To(BeTrue())
			Expect(buf.String()).To(Equal("Generating code...\n"))
		})

		It("still calls fn and returns its error", func() {
			var buf bytes.Buffer
			want := errors.New("boom")

			err := run(&buf, false, "Generating code", func() error {
				return want
			})

			Expect(err).To(MatchError(want))
		})

		It("never writes a carriage return, so log output stays readable", func() {
			var buf bytes.Buffer
			Expect(run(&buf, false, "Working", func() error { return nil })).To(Succeed())
			Expect(buf.String()).NotTo(ContainSubstring("\r"))
		})
	})

	Context("when stdout is a terminal", func() {
		It("animates while fn runs and fully clears the line before returning", func() {
			var buf bytes.Buffer

			err := run(&buf, true, "Generating code", func() error {
				time.Sleep(3 * interval)
				return nil
			})

			Expect(err).NotTo(HaveOccurred())

			out := buf.String()
			Expect(out).To(ContainSubstring("Generating code"))
			// At least the first paint plus a couple of ticks happened.
			Expect(strings.Count(out, "\r")).To(BeNumerically(">=", 3))
			// The line is left fully cleared: spaces, then a final \r
			// with nothing after it.
			lastCR := strings.LastIndex(out, "\r")
			Expect(out[:lastCR]).To(HaveSuffix("\r" + strings.Repeat(" ", len("Generating code")+2)))
			Expect(out[lastCR:]).To(Equal("\r"))
		})

		It("propagates fn's error after clearing the line", func() {
			var buf bytes.Buffer
			want := errors.New("boom")

			err := run(&buf, true, "Working", func() error {
				return want
			})

			Expect(err).To(MatchError(want))
			Expect(buf.String()).To(HaveSuffix("\r"))
		})
	})
})
