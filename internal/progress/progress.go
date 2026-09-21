// Package progress shows a terminal spinner around a blocking call —
// the loading indicator `sgo init`/`sgo generate {proto,code,openapi}`
// show while they work (ARCHITECTURE.md §14). No new dependency:
// `charmbracelet/huh` already pulls in `bubbletea`/`bubbles` as
// indirect ones, but a full TUI Program loop is a heavier fit for "spin
// around one blocking function" than this package's ~70 lines.
package progress

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
)

var frames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

const interval = 80 * time.Millisecond

// Run executes fn while showing message as an animated spinner, if
// stdout is a real terminal — the same isatty check `sgo init`'s
// wizard-vs-flags branch already uses (ARCHITECTURE.md §10). Otherwise
// (piped output, CI, scripts) it prints a single "<message>..." line
// instead, so log/script output never fills up with "\r" control
// characters. Either way, the line is fully cleared before Run returns
// — callers print their own success/error message afterward, the same
// as every command already does today.
func Run(message string, fn func() error) error {
	return run(os.Stdout, isatty.IsTerminal(os.Stdout.Fd()), message, fn)
}

// run is Run's implementation with the output stream and TTY-ness
// injected, so tests can assert on exact output without redirecting
// the real os.Stdout.
func run(w io.Writer, isTTY bool, message string, fn func() error) error {
	if !isTTY {
		fmt.Fprintln(w, message+"...")
		return fn()
	}

	done := make(chan struct{})
	stopped := make(chan struct{})

	go func() {
		defer close(stopped)

		fmt.Fprintf(w, "\r%s %s", frames[0], message)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		i := 1
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				fmt.Fprintf(w, "\r%s %s", frames[i%len(frames)], message)
				i++
			}
		}
	}()

	err := fn()

	close(done)
	<-stopped

	fmt.Fprintf(w, "\r%s\r", strings.Repeat(" ", len([]rune(message))+2))

	return err
}
