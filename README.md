# sunny-go

`sgo` is an open-source CLI that bootstraps and evolves a Go service from
a `.proto` contract, serving both HTTP and gRPC from a hexagonal
(ports & adapters) core.

> **Status:** in active development. `sgo init`, `sgo generate proto`,
> and `sgo generate code` (below) are real and working — the quick start
> is runnable today, not aspirational. HTTP/gRPC serving, pluggable
> persistence adapters, and the web UI are still ahead. See
> `docs/CLI.md` for exactly what's implemented today vs. planned, and
> `PLAN.md` for the phase currently in progress.

- **`ARCHITECTURE.md`** — the target architecture: hexagonal layout,
  generated-vs-owned file strategy, pluggable HTTP frameworks and
  datastores, `sgo init`/`sgo ui` design.
- **`PLAN.md`** — phased delivery plan for getting there.
- **`docs/CLI.md`** — command reference (implemented and planned).

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
regenerates everything derived from it (types, ports, the mapper) without
touching what you wrote into `user_service.go`.

## What `sgo` generates

- A proto-first contract under `contract/pb/`, compiled with a pure-Go
  compiler (no `protoc`/`buf` install required) into `contract/gen/` via
  the real `protoc-gen-go`/`protoc-gen-go-grpc` plugins.
- A hexagonal core (`internal/core`) with domain entities, ports, and
  use-case services — framework- and datastore-agnostic.
- Pluggable HTTP adapters (Gin, Echo, or Chi) and a gRPC adapter, both
  backed by the same service implementation. *(planned — Phase 3)*
- Pluggable persistence (self-managed or ORM) across Postgres, MySQL, and
  MongoDB, plus Redis caching and Elasticsearch search adapters.
  *(planned — Phase 4)*
- Regeneration that never deletes hand-written business logic — see
  `ARCHITECTURE.md` §6. Implemented and covered by an end-to-end test
  suite that edits a proto and asserts hand-written code survives
  regeneration, with a real `go build` after each run.

## Contributing / development

See `PLAN.md` for the current phase and what's in scope for it.
