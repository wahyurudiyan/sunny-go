# sunny-go

`sgo` is an open-source CLI that bootstraps and evolves a Go service from
a `.proto` contract, serving both HTTP and gRPC from a hexagonal
(ports & adapters) core.

> **Status:** in active development. `sgo init`, `sgo generate
> {proto,code,openapi}`, `sgo openapi validate`, `sgo list services`, and
> `sgo ui` are all real and working: a generated project serves HTTP and
> gRPC once you implement its service, backed by a real Postgres, MySQL,
> or MongoDB repository if you selected one (in-memory otherwise), plus
> Redis/Elasticsearch clients if selected — the quick start is runnable
> today, not aspirational. `sgo ui` covers the same ground from a browser
> instead of the terminal, calling the identical
> `internal/codegen`/`internal/config` functions the CLI does. See
> `docs/CLI.md` for exactly what's implemented, and `PLAN.md` for what's
> left (just polish at this point).

- **`ARCHITECTURE.md`** — the target architecture: hexagonal layout,
  generated-vs-owned file strategy, pluggable HTTP frameworks and
  datastores, `sgo init`/`sgo ui` design.
- **`PLAN.md`** — phased delivery plan for getting there.
- **`docs/CLI.md`** — command reference (implemented and planned).
- **`CHANGELOG.md`** — release history (`sgo --version`).

## Quick start

```
sgo init myservice --http-framework gin --persistence-mode orm --db postgres
cd myservice
sgo generate proto user
# edit contract/pb/user.proto
sgo generate code user
# implement business logic in internal/core/service/user_service.go
```

Running `sgo generate code user` again after editing the proto further
regenerates everything derived from it (types, ports, the mapper, HTTP
routes, the gRPC server) without touching what you wrote into
`user_service.go`. `go run ./cmd/myservice` then serves HTTP on `:8080`
and gRPC on `:9090`, both backed by the same service instance.

## What `sgo` generates

- A proto-first contract under `contract/pb/`, compiled with a pure-Go
  compiler (no `protoc`/`buf` install required) into `contract/gen/` via
  the real `protoc-gen-go`/`protoc-gen-go-grpc` plugins.
- A hexagonal core (`internal/core`) with domain entities, ports, and
  use-case services — framework- and datastore-agnostic.
- Pluggable HTTP adapters (Gin, Echo, or Chi) and a gRPC adapter, both
  backed by the same service implementation. Routes are derived from RPC
  naming (`Create`/`Get`/`List`/`Update`/`Delete`), not `google.api.http`
  annotations yet.
- A generated in-memory repository so a service is always runnable, even
  with nothing selected. If `sgo.yaml` selects one, the real adapter
  instead: self-managed (hand-written SQL) or GORM for Postgres/MySQL,
  the official driver for MongoDB — verified to actually persist data
  across a server restart, not just compile. Redis caching and
  Elasticsearch search clients are generated and connected if selected,
  available for a service to use, though not auto-wired into one (that's
  a constructor edit you make yourself, on purpose — see `ARCHITECTURE.md`
  §8.2).
- Regeneration that never deletes hand-written business logic — see
  `ARCHITECTURE.md` §6. Implemented and covered by an end-to-end test
  suite that edits a proto and asserts hand-written code survives
  regeneration, with a real `go build` after each run, plus a suite that
  builds and runs the compiled binary and drives a full HTTP CRUD cycle
  against it, plus a suite that drives the actual `sgo` binary itself as
  a subprocess through the whole quick-start flow above.
- OpenAPI documentation (`sgo generate openapi`, ARCHITECTURE.md §13):
  a project-wide `docs/openapi.yaml`/`.json`, 3.0 or 3.1, built from the
  same routes the HTTP adapter actually serves — never a hand-maintained
  second copy. `sgo openapi validate` checks any document (that one, or
  any other file, `sgo`-generated or not) against the real OpenAPI JSON
  Schema meta-schema.

## Contributing / development

See `PLAN.md` for the current phase and what's in scope for it.

```
make build   # bin/sgo
make test    # go test ./... (Ginkgo specs)
make dev     # go run ./cmd, no build step
```

Most specs are self-contained (they generate into a temp dir and either
pattern-match the output or build/run a throwaway module). A handful hold
generated Postgres/Redis adapters to a higher standard by running them
against a real local instance: they dial `127.0.0.1:5432`/`:6379` first
and skip gracefully (`Skip(...)`, not a failure) if nothing answers, so
`make test` is safe with no services running at all. To actually exercise
that path, start a local Postgres and Redis reachable with sgo's own
generated-adapter defaults — user `postgres`, password `postgres`,
database `postgres`, and no auth on Redis. MySQL, MongoDB, and
Elasticsearch adapters are compile-verified instead (see
`ARCHITECTURE.md` §12) since this project doesn't assume those are
installed locally.
