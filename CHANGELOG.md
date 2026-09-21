# Changelog

All notable changes to `sgo` are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project uses [Semantic Versioning](https://semver.org/).

## [0.0.1] - 2026-09-21

First real release: a working proto-first Go service generator, both a
CLI and a web UI, described in full in `ARCHITECTURE.md` and delivered
phase by phase per `PLAN.md`.

### Added

- **`sgo init`** — scaffold a new hexagonal (ports & adapters) Go
  service. Interactive terminal wizard (`charmbracelet/huh`) when no
  selection flags are given and stdin is a TTY; flag-driven otherwise.
  Selects the HTTP framework (Gin, Echo, or Chi), persistence mode (ORM
  or self-managed) and datastore(s) (Postgres, MySQL, MongoDB), optional
  Redis cache and Elasticsearch search, and OpenAPI documentation
  version/format — all persisted to `sgo.yaml`.
- **`sgo generate proto <name>`** — scaffold a starter CRUD proto
  (`Create`/`Get`/`List`/`Update`/`Delete` RPCs and their messages).
- **`sgo generate code <name>`** — compile the proto with a pure-Go
  compiler (no `protoc`/`buf` install required) and generate the
  hexagonal core (domain entities, usecase/repository ports, service
  skeletons), the wire↔domain mapper, HTTP routes for the selected
  framework, a gRPC server adapter, the selected persistence/cache/
  search adapters (or an in-memory fallback), and the composition root.
  Regeneration never deletes hand-written business logic.
- **`sgo generate openapi`** — generate a project-wide OpenAPI 3.0.x or
  3.1.x document (yaml or json) from every registered service's already-
  derived HTTP routes, self-validated against the real OpenAPI JSON
  Schema meta-schema before being written.
- **`sgo openapi validate [path]`** — validate any OpenAPI document
  (json or yaml, 3.0.x or 3.1.x) against the real OpenAPI meta-schema;
  defaults to the current project's generated doc.
- **`sgo list services`** — show each tracked service's generation
  status (proto / contract/gen / domain entity / service implementation).
- **`sgo ui [--port 4747]`** — a localhost-only web UI covering the same
  ground as the CLI: a create-project form, and (inside an existing
  project) a dashboard with an `sgo.yaml` viewer/editor, a services list
  with generated-vs-owned status, a new-service form, per-service
  "Generate code" and proto-editing, calling the exact same
  `internal/codegen`/`internal/config` functions the CLI does.
- Real, verified persistence adapters: Postgres and MySQL (self-managed
  SQL or GORM), MongoDB, plus Redis caching and Elasticsearch search
  clients — generated and connected when selected, alongside a default
  in-memory repository so a service is always runnable.
- `--version` / `-v`.

### Notes

- No `google.api.http` support yet — HTTP routes are derived from the
  `Create`/`Get`/`List`/`Update`/`Delete` RPC naming convention.
- One active persistence engine per project (not per entity).
- MySQL, MongoDB, and Elasticsearch adapters are compile-verified in CI;
  Postgres and Redis are exercised against real local instances when
  available.

[0.0.1]: https://github.com/wahyurudiyan/sunny-go/releases/tag/v0.0.1
