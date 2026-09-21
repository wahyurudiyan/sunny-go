// Package run implements `sgo run` (ARCHITECTURE.md §15): finding a
// generated project's entrypoint, running it like `go run` would, and
// — in --debug mode — restarting it when the config dashboard writes a
// change.
package run

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
)

// FindCmdDir returns the project's single cmd/<name> directory,
// relative to projectDir (e.g. "cmd/shop") — sgo only ever scaffolds
// exactly one per project (ARCHITECTURE.md §3), so there's nothing to
// disambiguate given the project directory alone.
func FindCmdDir(projectDir string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(projectDir, "cmd", "*"))
	if err != nil {
		return "", fmt.Errorf("run: finding cmd/<name>: %w", err)
	}

	var dirs []string
	for _, m := range matches {
		info, err := os.Stat(m)
		if err == nil && info.IsDir() {
			dirs = append(dirs, m)
		}
	}

	switch len(dirs) {
	case 0:
		return "", fmt.Errorf("run: no cmd/<name> directory found under %s — is this an sgo project?", projectDir)
	case 1:
		return filepath.Rel(projectDir, dirs[0])
	default:
		return "", fmt.Errorf("run: found more than one cmd/<name> directory under %s (%v) — sgo-generated projects only ever have one", projectDir, dirs)
	}
}

// Supervisor owns one `go run ./<CmdDir>` child process at a time,
// restartable with a new environment. Process-group kill semantics
// (Stop) are POSIX-only (Linux/macOS) — the same platform assumption
// the rest of sgo's own tooling already makes.
type Supervisor struct {
	// ProjectDir is the generated project's root — the child runs with
	// this as its working directory.
	ProjectDir string
	// CmdDir is the entrypoint package, relative to ProjectDir (e.g.
	// "cmd/shop"), as returned by FindCmdDir.
	CmdDir string
	// Stdout/Stderr default to os.Stdout/os.Stderr when nil — the
	// child's own output, streamed live, the same as a plain `go run`.
	Stdout, Stderr io.Writer

	mu      sync.Mutex
	cmd     *exec.Cmd
	exited  chan struct{} // closed when the running cmd.Wait() returns
	waitErr error
}

// Start launches the child with env as its complete environment (see
// package run's MergeEnv for how `sgo run` itself decides what that
// is). Returns an error if a child is already running — call Stop (or
// Restart) first.
func (s *Supervisor) Start(env []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cmd != nil {
		return fmt.Errorf("run: already running")
	}

	cmd := exec.Command("go", "run", "./"+s.CmdDir)
	cmd.Dir = s.ProjectDir
	cmd.Env = env
	cmd.Stdout = s.stdout()
	cmd.Stderr = s.stderr()
	// go run spawns the actual compiled binary as its own child
	// process; putting the whole tree in its own process group lets
	// Stop kill both together instead of only the `go` toolchain
	// process and orphaning the binary it started.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("run: starting %s: %w", s.CmdDir, err)
	}

	exited := make(chan struct{})
	s.cmd = cmd
	s.exited = exited

	go func() {
		err := cmd.Wait()
		s.mu.Lock()
		s.waitErr = err
		s.mu.Unlock()
		close(exited)
	}()

	return nil
}

// Stop kills the running child's whole process group and waits for it
// to actually exit. A no-op if nothing is running.
func (s *Supervisor) Stop() error {
	s.mu.Lock()
	cmd, exited := s.cmd, s.exited
	s.mu.Unlock()

	if cmd == nil {
		return nil
	}

	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	<-exited // wait for the Wait() goroutine to actually reap it

	s.mu.Lock()
	s.cmd, s.exited = nil, nil
	s.mu.Unlock()

	return nil
}

// Restart stops the current child (if any) and starts a new one with
// env — the effect of a dashboard-driven config write.
func (s *Supervisor) Restart(env []string) error {
	if err := s.Stop(); err != nil {
		return err
	}
	return s.Start(env)
}

// Wait blocks until the currently running child exits on its own
// (crashed, or the program returned) and reports its error, or returns
// nil immediately if nothing is running. Not meant to be called
// concurrently with Restart from a different goroutine — `sgo run`'s
// non-debug mode uses Wait as its main blocking loop and never calls
// Restart; --debug mode calls Restart from dashboard handlers and never
// calls Wait (ARCHITECTURE.md §15).
func (s *Supervisor) Wait() error {
	s.mu.Lock()
	exited := s.exited
	s.mu.Unlock()

	if exited == nil {
		return nil
	}

	<-exited

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.waitErr
}

func (s *Supervisor) stdout() io.Writer {
	if s.Stdout != nil {
		return s.Stdout
	}
	return os.Stdout
}

func (s *Supervisor) stderr() io.Writer {
	if s.Stderr != nil {
		return s.Stderr
	}
	return os.Stderr
}
