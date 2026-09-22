# Contributing to sgo

Thanks for your interest in improving `sgo`. This document covers
building from source, running the test suite, and the workflow this
project actually uses for proposing and landing changes.

## Building from source

```sh
git clone https://github.com/wahyurudiyan/sunny-go.git
cd sunny-go
make build   # bin/sgo
make dev     # go run ./cmd/sgo, no build step
```

Requires Go 1.26+ (matches `go.mod`) — no `protoc`/`buf` install needed
for `sgo` itself or anything it generates; see `ARCHITECTURE.md` §4 for
why.

## Running the test suite

```sh
make test    # go test ./... (Ginkgo v2 + Gomega specs)
```

Every package with behavior worth testing has a Ginkgo suite —
`Describe`/`Context`/`It` blocks, not plain `testing.T` table tests. Most
specs are fully self-contained: they generate into a temp directory and
either pattern-match the output or assemble a throwaway Go module and
build/run it for real.

A handful hold generated adapters to a higher standard by running them
against a real local datastore:

- **Postgres/Redis**: these specs dial `127.0.0.1:5432`/`:6379` first and
  skip gracefully (`Skip(...)`, not a failure) if nothing answers, so
  `make test` is safe with no services running at all. To actually
  exercise that path, start a local Postgres and Redis reachable with
  `sgo`'s own generated-adapter defaults — user `postgres`, password
  `postgres`, database `postgres`, no auth on Redis.
- **MySQL, MongoDB, and Elasticsearch** adapters are compile-verified
  instead (a real throwaway module is assembled and `go build`'d against
  the generated code) rather than run against a live instance, since
  this project doesn't assume those are installed locally — see
  `ARCHITECTURE.md` §12.

Before pushing, also run:

```sh
gofmt -l .    # should print nothing
go vet ./...
```

## How this project plans and ships changes

`sgo` develops in phases, each documented before it's built, not after:

1. **`ARCHITECTURE.md`** describes the target design for a phase — the
   "why," not just the "what." A new phase gets a new numbered section
   (or a **(planned)** marker on an existing one) before any code is
   written.
2. **`PLAN.md`** breaks that design into a checklist with explicit exit
   criteria — what proves the phase is actually done, not just that code
   was written.
3. Where a design has more than one reasonable answer (a genuinely open
   question, not a style preference), it gets resolved explicitly and the
   decision — and why — is recorded in `ARCHITECTURE.md`'s decision
   table, not left implicit in a diff.
4. Implementation follows the plan, verified continuously: real `go
   build`/`go run` against generated output, not just pattern-matching
   generated source as text. A phase whose exit criteria depend on
   generated code actually working gets an end-to-end Ginkgo spec that
   proves it, e.g. editing a proto and asserting a hand-written method
   survives two regenerations with a real build after each.
5. Docs are updated alongside the code they describe, honestly — a
   partially-done phase gets marked as such (with what's actually
   deferred and why), not rounded up to "done."
6. A pull request is opened for review. **PRs are not merged by their
   own author** — someone else reviews and merges.

If you're proposing a change bigger than a bug fix, start by reading the
relevant `ARCHITECTURE.md` section and the current `PLAN.md` phase, so
your proposal fits the existing design rather than reinventing a decision
that's already been made (and reasoned about) elsewhere in those docs.
For something small (a typo, a clear bug fix, a missing test), a direct
PR is fine — the phase process above is for anything that changes what
`sgo` generates or how.

## Code style

- Ginkgo/Gomega for tests, not `testing.T` table tests — see any
  existing `*_test.go` file for the pattern (`Describe` names the unit
  under test, `Context` names the scenario, `It` states the expected
  behavior in one sentence).
- Generated-vs-owned file split (`ARCHITECTURE.md` §6) is a hard
  invariant, not a convention to bend: a `_gen.go` file is always fully
  rewritten; an owned file is created once and only ever appended to,
  never rewritten or reformatted, when it needs a new stub.
- No `protoc`/`buf` dependency, ever, for `sgo` itself or anything it
  generates — a new feature that would require one is the wrong design,
  not a tradeoff to accept (see `ARCHITECTURE.md` Decision #5).
- Prefer verifying generated code with a real `go build`/`go run` over
  trusting that a generated string "looks right."

## Reporting issues

Open a GitHub issue with what you ran, what you expected, and what
happened instead. For a bug in generated code, include the proto (or the
minimal part of it that reproduces the issue) and the `sgo init` flags
you used — both are usually needed to reproduce anything proto-generator
related.
