package envsource

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
)

// dotEnvSourceName is Name()'s (and every Value's Source field's) value
// for DotEnvSource.
const dotEnvSourceName = "dotenv"

// assignmentPattern matches a KEY=VALUE line, with an optional leading
// "export " (a common shell-sourceable .env convention) and optional
// single/double-quoting around the value. Multi-line values and
// ${VAR}-style interpolation aren't supported — out of scope for a
// debug-time convenience loader (ARCHITECTURE.md §15).
var assignmentPattern = regexp.MustCompile(`^(?:export\s+)?([A-Za-z_][A-Za-z0-9_]*)=(.*)$`)

// DotEnvSource reads and writes a .env file, the one real Source
// implementation in v1 (ARCHITECTURE.md §15). Write rewrites the file
// in place, preserving the order of every existing line — including
// comments and blanks — and appends new keys at the end.
type DotEnvSource struct {
	// Path is the .env file's path. It's fine for it not to exist yet:
	// Fetch reports no values, and Write creates it.
	Path string

	mu sync.Mutex
}

func (d *DotEnvSource) Name() string { return dotEnvSourceName }

// Fetch parses Path. A missing file is reported as zero values, not an
// error — most generated projects won't have a .env until someone adds
// one.
func (d *DotEnvSource) Fetch(_ context.Context) ([]Value, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	lines, err := d.readLines()
	if err != nil {
		return nil, err
	}

	values := make([]Value, 0, len(lines))
	for _, l := range lines {
		if !l.isAssignment {
			continue
		}
		values = append(values, Value{Key: l.key, Value: l.value, Source: dotEnvSourceName})
	}

	return values, nil
}

// Write sets key to value, updating it in place if the key already has
// a line, or appending a new `KEY=value` line otherwise. Rewriting an
// existing key's line drops any inline quoting/comment that line had —
// documented, deliberate: preserving *file* order and every *other*
// line verbatim is the guarantee; round-tripping one edited line's
// exact original formatting isn't.
func (d *DotEnvSource) Write(_ context.Context, key, value string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	lines, err := d.readLines()
	if err != nil {
		return err
	}

	found := false
	for i, l := range lines {
		if l.isAssignment && l.key == key {
			lines[i] = dotEnvLine{isAssignment: true, key: key, value: value}
			found = true
			break
		}
	}
	if !found {
		lines = append(lines, dotEnvLine{isAssignment: true, key: key, value: value})
	}

	return d.writeLines(lines)
}

type dotEnvLine struct {
	raw          string // verbatim for comments/blanks — printed as-is
	isAssignment bool
	key, value   string
}

func (l dotEnvLine) String() string {
	if !l.isAssignment {
		return l.raw
	}
	return fmt.Sprintf("%s=%s", l.key, l.value)
}

func (d *DotEnvSource) readLines() ([]dotEnvLine, error) {
	data, err := os.ReadFile(d.Path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("envsource: reading %s: %w", d.Path, err)
	}

	var lines []dotEnvLine
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		raw := scanner.Text()
		trimmed := strings.TrimSpace(raw)

		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			lines = append(lines, dotEnvLine{raw: raw})
			continue
		}

		m := assignmentPattern.FindStringSubmatch(trimmed)
		if m == nil {
			lines = append(lines, dotEnvLine{raw: raw})
			continue
		}

		lines = append(lines, dotEnvLine{
			isAssignment: true,
			key:          m[1],
			value:        unquote(m[2]),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("envsource: reading %s: %w", d.Path, err)
	}

	return lines, nil
}

func (d *DotEnvSource) writeLines(lines []dotEnvLine) error {
	var buf bytes.Buffer
	for _, l := range lines {
		buf.WriteString(l.String())
		buf.WriteByte('\n')
	}

	if err := os.WriteFile(d.Path, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("envsource: writing %s: %w", d.Path, err)
	}
	return nil
}

// unquote strips one layer of matching single or double quotes, the
// same convention most .env tooling follows for values containing
// spaces (e.g. KEY="hello world").
func unquote(v string) string {
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			return v[1 : len(v)-1]
		}
	}
	return v
}
