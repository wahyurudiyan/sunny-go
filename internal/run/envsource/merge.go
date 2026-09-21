package envsource

import (
	"context"
	"os"
)

// Merge fetches every source and combines the result with the real
// process environment into one slice suitable for exec.Cmd.Env:
// source values first, then the real environment last, so a real
// OS/CI environment variable already set always wins over a leftover
// config-source value — the convention most .env tooling already
// follows (exec.Cmd keeps only the last occurrence of a duplicate
// key), and the merge precedence `sgo run` itself uses on startup.
func Merge(ctx context.Context, sources []Source) ([]string, error) {
	var env []string
	for _, src := range sources {
		values, err := src.Fetch(ctx)
		if err != nil {
			return nil, err
		}
		for _, v := range values {
			env = append(env, v.Key+"="+v.Value)
		}
	}
	env = append(env, os.Environ()...)
	return env, nil
}
