package wizard

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// yesColumn returns the rendered column "Yes" starts at — diagnosed by
// capturing the real wizard under a pty: huh's Confirm centers its
// buttons under a box sized to that field's own title text by default,
// so two Confirm fields with different-length titles in the same group
// land their buttons at different columns. This regression guard
// exercises huh's real View() rendering rather than re-deriving the
// bug from the library's source a second time.
func yesColumn(field huh.Field) int {
	GinkgoHelper()
	rendered := ansiEscape.ReplaceAllString(field.View(), "")
	for _, line := range strings.Split(rendered, "\n") {
		if idx := strings.Index(line, "Yes"); idx >= 0 {
			return idx
		}
	}
	Fail("no rendered line contained \"Yes\"")
	return -1
}

var _ = Describe("wizard Confirm fields", func() {
	It("renders every Confirm's buttons at the same column within a group, regardless of title length", func() {
		short := huh.NewConfirm().
			Title("Enable Redis cache?").
			WithButtonAlignment(lipgloss.Left).
			WithWidth(80)
		long := huh.NewConfirm().
			Title("Enable Elasticsearch search?").
			WithButtonAlignment(lipgloss.Left).
			WithWidth(80)

		Expect(yesColumn(short)).To(Equal(yesColumn(long)))
	})
})
