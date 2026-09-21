# Changelog

All notable changes to `sgo` are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project uses [Semantic Versioning](https://semver.org/).

## [0.2.0] - 2026-09-21

Loading indicators, and `sgo run [--debug]` with a live config
dashboard, both described in full in `ARCHITECTURE.md` §14/§15 and
`PLAN.md` Phases 9/10.

### Added

- **`sgo run [--debug] [--debug-port 4748]`** — run the current
  project's service the way `go run ./cmd/<name>` would, loading
  configuration from `.env` (a real OS/CI environment variable already
  set always wins over a leftover `.env` value). Stops the service
  cleanly on Ctrl-C.
- **`--debug`** additionally serves a `127.0.0.1`-only config
  dashboard: lists every config key in effect, its source, and
  whether it's writable; editing a writable value writes it back to
  `.env` and restarts the service with the new value in effect. Live
  updates via Server-Sent Events. Values are masked by default — never
  sent to the browser until you explicitly reveal one.
- `internal/run/envsource` — the `Source` interface configuration
  providers implement (`.env` today; a remote config repo and
  KMS/Vault-style secret managers are a defined interface with no
  concrete adapter yet).
- Terminal loading spinners for `sgo init`, `sgo generate
  {proto,code,openapi}` — animates on a real terminal, prints a plain
  status line when piped (CI/scripts unaffected).

### Fixed

- `sgo --version` reported `0.0.1` on what was actually already
  published as the `0.1.0` release; this changelog's own `0.0.1` entry
  below is renamed to `0.1.0` to match the tag actually published for
  it.

## [0.1.0] - 2026-09-21

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

[0.2.0]: https://github.com/wahyurudiyan/sunny-go/releases/tag/0.2.0
[0.1.0]: https://github.com/wahyurudiyan/sunny-go/releases/tag/0.1.0
