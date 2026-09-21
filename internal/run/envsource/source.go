// Package envsource defines where `sgo run` reads and writes
// configuration (ARCHITECTURE.md §15): a small Source interface, and
// today exactly one real implementation, DotEnvSource. A remote config
// repository and KMS/Vault-style secret managers get the same interface
// later — this package is what makes that possible without redesigning
// sgo run or the --debug dashboard when one is added.
package envsource

import "context"

// Value is one configuration key/value as reported by a Source.
type Value struct {
	Key    string
	Value  string
	Source string // the reporting Source's Name(), e.g. "dotenv"
}

// Source is a configuration source `sgo run` can read from, and —
// where supported — write back to.
type Source interface {
	// Name identifies this source; stamped onto every Value it returns
	// from Fetch.
	Name() string

	// Fetch returns every key/value currently held by this source. A
	// source with nothing to report (e.g. no .env file present) returns
	// a nil slice and no error — that's a normal, expected state, not a
	// failure.
	Fetch(ctx context.Context) ([]Value, error)

	// Write persists key's new value back to this source, creating the
	// key if it doesn't already exist. A source that cannot support
	// writes (e.g. a read-only KMS grant) returns a clear error here —
	// never a silent no-op.
	Write(ctx context.Context, key, value string) error
}
