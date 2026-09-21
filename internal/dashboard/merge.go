package dashboard

import (
	"context"
	"os"
	"sort"
	"strings"

	"github.com/wahyurudiyan/sunny-go/internal/run/envsource"
)

// configValue is one entry in the dashboard's merged view: a key
// declared by at least one configured envsource.Source, together with
// its currently effective value and whether the dashboard can write it.
type configValue struct {
	Key      string
	Value    string
	Source   string
	Writable bool
}

// mergeSources fetches every configured source and returns:
//   - env: the full environment to run the child with — source values
//     first, then the real process environment last, so a real
//     OS/CI-set variable always wins over a leftover .env value
//     (exec.Cmd keeps only the last occurrence of a duplicate key).
//   - values: one entry per key any source declared — sgo run's own
//     configuration surface, not the whole OS environment. A key also
//     set in the real environment shows as effective source "env" and
//     isn't writable, since editing its source wouldn't change what
//     the child actually sees.
//   - owner: which Source instance a given key's next Write should
//     target — whichever source's Fetch reported it.
func mergeSources(ctx context.Context, sources []envsource.Source) (env []string, values []configValue, owner map[string]envsource.Source, err error) {
	owner = make(map[string]envsource.Source)
	declared := make(map[string]envsource.Value)
	var order []string

	for _, src := range sources {
		vs, ferr := src.Fetch(ctx)
		if ferr != nil {
			return nil, nil, nil, ferr
		}
		for _, v := range vs {
			if _, exists := declared[v.Key]; !exists {
				order = append(order, v.Key)
			}
			declared[v.Key] = v
			owner[v.Key] = src
		}
	}

	// The env the child should actually run with — same precedence
	// rule sgo run itself uses on startup.
	env, err = envsource.Merge(ctx, sources)
	if err != nil {
		return nil, nil, nil, err
	}

	osEnv := os.Environ()
	osValues := make(map[string]string, len(osEnv))
	for _, kv := range osEnv {
		key, val, _ := strings.Cut(kv, "=")
		osValues[key] = val
	}

	sort.Strings(order)
	for _, key := range order {
		if v, ok := osValues[key]; ok {
			values = append(values, configValue{Key: key, Value: v, Source: "env", Writable: false})
			continue
		}
		d := declared[key]
		values = append(values, configValue{Key: key, Value: d.Value, Source: d.Source, Writable: true})
	}

	return env, values, owner, nil
}
