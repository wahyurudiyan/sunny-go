package run_test

import (
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/run"
)

var _ = Describe("FindCmdDir", func() {
	var dir string

	BeforeEach(func() {
		var err error
		dir, err = os.MkdirTemp("", "sgo-run-findcmddir-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })
	})

	It("errors when there's no cmd directory at all", func() {
		_, err := run.FindCmdDir(dir)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no cmd/<name> directory"))
	})

	It("returns the single cmd/<name> directory, relative to projectDir", func() {
		Expect(os.MkdirAll(filepath.Join(dir, "cmd", "shop"), 0755)).To(Succeed())

		got, err := run.FindCmdDir(dir)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(filepath.Join("cmd", "shop")))
	})

	It("ignores files under cmd/, only counting directories", func() {
		Expect(os.MkdirAll(filepath.Join(dir, "cmd", "shop"), 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(dir, "cmd", "README.md"), []byte("hi"), 0644)).To(Succeed())

		got, err := run.FindCmdDir(dir)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(filepath.Join("cmd", "shop")))
	})

	It("errors when there's more than one cmd/<name> directory", func() {
		Expect(os.MkdirAll(filepath.Join(dir, "cmd", "shop"), 0755)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(dir, "cmd", "worker"), 0755)).To(Succeed())

		_, err := run.FindCmdDir(dir)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("more than one"))
	})
})

// safeBuffer is a bytes.Buffer safe for one writer goroutine (the
// supervised child's stdout pipe) and one reader goroutine (the spec
// polling it via Eventually) at once.
type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

var _ = Describe("Supervisor", func() {
	var (
		out *safeBuffer
		sup *run.Supervisor
	)

	BeforeEach(func() {
		out = &safeBuffer{}
		sup = &run.Supervisor{
			ProjectDir: "testdata/fixture",
			CmdDir:     "cmd/testapp",
			Stdout:     out,
			Stderr:     out,
		}
		DeferCleanup(func() { Expect(sup.Stop()).To(Succeed()) })
	})

	It("runs the child with the given environment", func() {
		Expect(sup.Start(append(os.Environ(), "GREETING=hello"))).To(Succeed())

		Eventually(out.String, 10*time.Second, 50*time.Millisecond).Should(ContainSubstring("GREETING=hello"))
	})

	It("refuses to Start twice without a Stop in between", func() {
		Expect(sup.Start(append(os.Environ(), "GREETING=hello"))).To(Succeed())
		Eventually(out.String, 10*time.Second, 50*time.Millisecond).Should(ContainSubstring("GREETING=hello"))

		err := sup.Start(append(os.Environ(), "GREETING=again"))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("already running"))
	})

	It("Stop kills the child (including go run's spawned binary), and is idempotent", func() {
		Expect(sup.Start(append(os.Environ(), "GREETING=hello"))).To(Succeed())
		Eventually(out.String, 10*time.Second, 50*time.Millisecond).Should(ContainSubstring("GREETING=hello"))

		Expect(sup.Stop()).To(Succeed())

		before := len(out.String())
		Consistently(func() int { return len(out.String()) }, 300*time.Millisecond, 50*time.Millisecond).Should(Equal(before))

		// nothing running: a second Stop is a no-op, not an error
		Expect(sup.Stop()).To(Succeed())
	})

	It("Restart stops the current child and starts a new one with a new environment", func() {
		Expect(sup.Start(append(os.Environ(), "GREETING=hello"))).To(Succeed())
		Eventually(out.String, 10*time.Second, 50*time.Millisecond).Should(ContainSubstring("GREETING=hello"))

		Expect(sup.Restart(append(os.Environ(), "GREETING=world"))).To(Succeed())

		Eventually(out.String, 10*time.Second, 50*time.Millisecond).Should(ContainSubstring("GREETING=world"))
	})

	It("Wait returns nil immediately when nothing is running", func() {
		Expect(sup.Wait()).To(Succeed())
	})

	It("Wait blocks while the child is running and unblocks once it exits", func() {
		Expect(sup.Start(append(os.Environ(), "GREETING=hello"))).To(Succeed())
		Eventually(out.String, 10*time.Second, 50*time.Millisecond).Should(ContainSubstring("GREETING=hello"))

		waitReturned := make(chan struct{})
		go func() {
			defer close(waitReturned)
			_ = sup.Wait()
		}()

		isClosed := func() bool {
			select {
			case <-waitReturned:
				return true
			default:
				return false
			}
		}

		Consistently(isClosed, 200*time.Millisecond, 20*time.Millisecond).Should(BeFalse())

		Expect(sup.Stop()).To(Succeed())

		Eventually(isClosed, 2*time.Second, 20*time.Millisecond).Should(BeTrue())
	})
})
