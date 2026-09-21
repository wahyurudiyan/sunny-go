package commands_test

// End-to-end coverage for `sgo run --debug` (PLAN.md/ARCHITECTURE.md
// §15), driving the real compiled `sgo` binary as a subprocess — same
// spirit as e2e_test.go. It runs against a minimal fixture project
// (a cmd/<name>/main.go that just echoes an env var on a loop, like
// internal/run's own supervisor fixture) rather than a full
// `sgo init`-scaffolded service, so the child's own stdout is direct,
// unambiguous proof of what environment it actually started with —
// no /proc-reading or guessing at the generated app's internals
// needed to confirm a dashboard edit really reached the child process.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// safeBuffer is a bytes.Buffer safe for the subprocess's stdout pipe
// writer goroutine and a spec's polling reads at once.
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

// freePort asks the OS for an ephemeral port, then releases it — a
// small, accepted TOCTOU race, same tradeoff any test picking a port
// for a real listener makes.
func freePort() int {
	GinkgoHelper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	Expect(err).NotTo(HaveOccurred())
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// writeEchoFixture scaffolds the minimal thing `sgo run` needs — a
// single cmd/<name>/main.go — as a program that loops printing its
// GREETING env var, so a spec can observe exactly what environment it
// was (re)started with via its own stdout.
func writeEchoFixture(dir string) {
	GinkgoHelper()
	mainDir := filepath.Join(dir, "cmd", "echoapp")
	Expect(os.MkdirAll(mainDir, 0755)).To(Succeed())

	// dir lives outside the sunny-go module tree (a fresh os.MkdirTemp),
	// so `go run ./cmd/echoapp` needs its own go.mod to find a module
	// root at all.
	Expect(os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module echofixture\n\ngo 1.25.0\n"), 0644)).To(Succeed())

	src := `package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	greeting := os.Getenv("GREETING")
	for {
		fmt.Println("GREETING=" + greeting)
		time.Sleep(20 * time.Millisecond)
	}
}
`
	Expect(os.WriteFile(filepath.Join(mainDir, "main.go"), []byte(src), 0644)).To(Succeed())
}

var _ = Describe("sgo run --debug, driven as a real subprocess", func() {
	It("serves the config dashboard, and a PUT through it writes .env and restarts the child with the new value", func() {
		projectDir, err := os.MkdirTemp("", "sgo-run-e2e-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(projectDir)).To(Succeed()) })

		writeEchoFixture(projectDir)
		envPath := filepath.Join(projectDir, ".env")
		Expect(os.WriteFile(envPath, []byte("GREETING=hello\n"), 0644)).To(Succeed())

		port := freePort()

		out := &safeBuffer{}
		cmd := exec.Command(sgoBinary, "run", "--debug", "--debug-port", strconv.Itoa(port))
		cmd.Dir = projectDir
		cmd.Stdin = strings.NewReader("")
		cmd.Stdout = out
		cmd.Stderr = out
		Expect(cmd.Start()).To(Succeed())

		exited := make(chan error, 1)
		go func() { exited <- cmd.Wait() }()

		DeferCleanup(func() {
			_ = cmd.Process.Signal(syscall.SIGTERM)
			Eventually(exited, 10*time.Second).Should(Receive())
		})

		base := fmt.Sprintf("http://127.0.0.1:%d", port)

		Eventually(out.String, 10*time.Second, 50*time.Millisecond).Should(ContainSubstring("GREETING=hello"))
		Eventually(func() error {
			resp, err := http.Get(base + "/api/config")
			if err != nil {
				return err
			}
			return resp.Body.Close()
		}, 10*time.Second, 100*time.Millisecond).Should(Succeed())

		resp, err := http.Get(base + "/api/config")
		Expect(err).NotTo(HaveOccurred())
		var items []map[string]any
		Expect(json.NewDecoder(resp.Body).Decode(&items)).To(Succeed())
		Expect(resp.Body.Close()).To(Succeed())
		Expect(items).To(ContainElement(map[string]any{"key": "GREETING", "source": "dotenv", "writable": true}))

		putReq, err := http.NewRequest(http.MethodPut, base+"/api/config/GREETING",
			bytes.NewBufferString(`{"value":"world"}`))
		Expect(err).NotTo(HaveOccurred())
		putReq.Header.Set("Content-Type", "application/json")
		putResp, err := http.DefaultClient.Do(putReq)
		Expect(err).NotTo(HaveOccurred())
		Expect(putResp.StatusCode).To(Equal(http.StatusOK))
		Expect(putResp.Body.Close()).To(Succeed())

		content, err := os.ReadFile(envPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(Equal("GREETING=world\n"))

		// The direct proof the dashboard-driven restart actually reached
		// the child: its own stdout now echoes the new value.
		Eventually(out.String, 10*time.Second, 50*time.Millisecond).Should(ContainSubstring("GREETING=world"))
	})

	It("exits cleanly on SIGTERM, leaving no child process behind, in non-debug mode too", func() {
		projectDir, err := os.MkdirTemp("", "sgo-run-e2e-sigterm-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(projectDir)).To(Succeed()) })

		writeEchoFixture(projectDir)

		out := &safeBuffer{}
		cmd := exec.Command(sgoBinary, "run")
		cmd.Dir = projectDir
		cmd.Stdin = strings.NewReader("")
		cmd.Stdout = out
		cmd.Stderr = out
		Expect(cmd.Start()).To(Succeed())

		exited := make(chan error, 1)
		go func() { exited <- cmd.Wait() }()

		Eventually(out.String, 10*time.Second, 50*time.Millisecond).Should(ContainSubstring("GREETING="))

		Expect(cmd.Process.Signal(syscall.SIGTERM)).To(Succeed())
		Eventually(exited, 10*time.Second).Should(Receive(BeNil()))

		Consistently(func() (string, error) {
			// Matches both the `go run ./cmd/echoapp` toolchain process and
			// its spawned compiled binary (named "echoapp" by go's usual
			// last-path-component convention) — proof Stop's process-group
			// kill reached both, not just the direct child. A killed
			// process's own entry can briefly (or, in a container without
			// a reaping init, indefinitely) survive as a zombie/<defunct>
			// row purely awaiting its new parent's wait() — already dead,
			// not a real leak — so those are filtered out rather than
			// treated as evidence Stop failed.
			check := exec.Command("ps", "-eo", "stat,args")
			checkOut, err := check.Output()
			if err != nil {
				return "", err
			}
			var alive []string
			for _, line := range strings.Split(string(checkOut), "\n") {
				if !strings.Contains(line, "echoapp") {
					continue
				}
				if strings.HasPrefix(strings.TrimSpace(line), "Z") || strings.Contains(line, "defunct") {
					continue
				}
				alive = append(alive, line)
			}
			return strings.Join(alive, "\n"), nil
		}, 300*time.Millisecond, 50*time.Millisecond).Should(BeEmpty())
	})
})
