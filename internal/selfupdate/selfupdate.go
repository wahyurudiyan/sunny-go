// Package selfupdate resolves and installs new versions of sgo itself,
// via `go list -m -versions` and `go install` — the same module-proxy
// resolution and install mechanism a user would invoke by hand
// (README's own install line), not a second, hand-rolled HTTP client
// against the module proxy or a self-replacing binary downloader.
// ARCHITECTURE.md §23 has the full design and its rationale.
package selfupdate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/mod/semver"
)

// Module is sgo's own module path — what `sgo update` resolves
// available versions for.
const Module = "github.com/wahyurudiyan/sunny-go"

// Package is the installable command path under Module that `go
// install` actually builds — matches README's own install line and
// cmd/sgo's location in the module.
const Package = Module + "/cmd/sgo"

// listOutput mirrors the fields `go list -m -versions -json` prints
// that we actually use. The command always emits at least "Path"; the
// main module itself (or one with zero published tags) simply comes
// back without a "Versions" field, which is not an error.
type listOutput struct {
	Versions []string
}

// Versions runs `go list -m -versions -json module` and returns every
// published semver tag, sorted ascending. A module with no tags (or
// sgo's own module when run from inside its own checkout, where `go
// list` describes the main module instead of querying the proxy)
// returns a nil slice with no error — that's a real, expected outcome,
// not a failure, and callers should check for it explicitly rather
// than treating an empty result as "something went wrong".
func Versions(module string) ([]string, error) {
	cmd := exec.Command("go", "list", "-m", "-versions", "-json", module)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("go list -m -versions %s: %w\n%s", module, err, exitErr.Stderr)
		}
		return nil, fmt.Errorf("go list -m -versions %s: %w", module, err)
	}

	var parsed listOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("parsing go list output for %s: %w", module, err)
	}

	versions := make([]string, 0, len(parsed.Versions))
	for _, v := range parsed.Versions {
		if semver.IsValid(v) {
			versions = append(versions, v)
		}
	}
	semver.Sort(versions)

	return versions, nil
}

// Latest returns the highest semver version Versions(module) finds, and
// false if there are none.
func Latest(module string) (version string, ok bool, err error) {
	versions, err := Versions(module)
	if err != nil {
		return "", false, err
	}
	if len(versions) == 0 {
		return "", false, nil
	}
	return versions[len(versions)-1], true, nil
}

// NormalizeVersion prefixes a bare "0.2.0"-style version with "v" (Go
// module versions are always "v"-prefixed) and reports whether the
// result is a syntactically valid semver — it does not check that the
// version is actually published.
func NormalizeVersion(v string) (string, bool) {
	if v == "" {
		return "", false
	}
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return v, semver.IsValid(v)
}

// Install runs `go install pkg@version`, streaming its combined output
// to w as it runs so a build failure is visible immediately rather than
// swallowed. `go install` itself already writes to a temp file and
// renames into place, so no separate atomic-replace logic is needed
// here.
func Install(w io.Writer, pkg, version string) error {
	cmd := exec.Command("go", "install", fmt.Sprintf("%s@%s", pkg, version))
	cmd.Stdout = w
	cmd.Stderr = w

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go install %s@%s: %w", pkg, version, err)
	}
	return nil
}

// BinDir returns the directory `go install` places a built binary in:
// `go env GOBIN` if set, otherwise `go env GOPATH`'s "bin"
// subdirectory — the same fallback `go install` itself applies, so
// this always names the real destination rather than guessing.
func BinDir() (string, error) {
	if gobin, err := goEnv("GOBIN"); err != nil {
		return "", err
	} else if gobin != "" {
		return gobin, nil
	}

	gopath, err := goEnv("GOPATH")
	if err != nil {
		return "", err
	}
	if gopath == "" {
		return "", fmt.Errorf("neither GOBIN nor GOPATH is set")
	}

	return gopath + "/bin", nil
}

// BinMismatch reports whether the directory `go install` places a
// binary in differs from the directory the given executable path lives
// in — a copy, a symlink, or a second Go environment putting the
// currently running sgo somewhere other than GOBIN. Pass "" for exe to
// use the currently running process's own executable (os.Executable()).
// Exposed as its own function, rather than folded into the command
// layer, so it's testable without needing a real `go install` to
// succeed first.
func BinMismatch(exe string) (installDir, runningFrom string, mismatched bool, err error) {
	installDir, err = BinDir()
	if err != nil {
		return "", "", false, err
	}

	if exe == "" {
		exe, err = os.Executable()
		if err != nil {
			return "", "", false, err
		}
	}
	if resolved, resolveErr := filepath.EvalSymlinks(exe); resolveErr == nil {
		exe = resolved
	}

	return installDir, exe, filepath.Dir(exe) != installDir, nil
}

func goEnv(key string) (string, error) {
	cmd := exec.Command("go", "env", key)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("go env %s: %w", key, err)
	}
	return strings.TrimSpace(out.String()), nil
}
