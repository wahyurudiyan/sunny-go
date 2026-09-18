# sunny-go

`sgo` is an open-source CLI that bootstraps and evolves a Go service from
a `.proto` contract, serving both HTTP and gRPC from a hexagonal
(ports & adapters) core.

> **Status:** in active redesign. The command surface described here
> (`sgo`) is the target; the current branch still exposes the older
> `sunny` command set while the rework in `PLAN.md` lands. See
> `docs/CLI.md` for exactly what's implemented today vs. planned.

- **`ARCHITECTURE.md`** — the target architecture: hexagonal layout,
  generated-vs-owned file strategy, pluggable HTTP frameworks and
  datastores, `sgo init`/`sgo ui` design.
- **`PLAN.md`** — phased delivery plan for getting there.
- **`docs/CLI.md`** — command reference (implemented and planned).

## Quick start (target)

```
sgo init myservice --http-framework gin --persistence-mode orm --db postgres
cd myservice
sgo generate proto user
# edit contract/pb/user.proto
sgo generate code user
# implement business logic in internal/core/service/user_service.go
```

## What `sgo` generates

- A proto-first contract under `contract/pb/`, compiled via `buf`/`protoc`
  into `contract/gen/`.
- A hexagonal core (`internal/core`) with domain entities, ports, and
  use-case services — framework- and datastore-agnostic.
- Pluggable HTTP adapters (Gin, Echo, or Chi) and a gRPC adapter, both
  backed by the same service implementation.
- Pluggable persistence (self-managed or ORM) across Postgres, MySQL, and
  MongoDB, plus Redis caching and Elasticsearch search adapters.
- Regeneration that never deletes hand-written business logic — see
  `ARCHITECTURE.md` §6.

## Contributing / development

See `PLAN.md` for the current phase and what's in scope for it.
