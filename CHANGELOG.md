# Changelog

All notable changes to `sgo` are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project uses [Semantic Versioning](https://semver.org/).

## [0.1.1] - 2026-09-22

Replaces the generated architecture's hexagonal core with real DDD
tactical patterns (aggregates, value objects, domain events, a CQRS
application layer), then builds five more phases on top of it:
discoverability (`sgo list endpoints`, an OpenAPI viewer UI),
proto-defined HTTP routing, repository-port methods beyond fixed CRUD,
and a rewritten README/CONTRIBUTING.md. Full detail in `ARCHITECTURE.md`
and `PLAN.md`.

### Added

- **DDD domain/application/infrastructure layers**, replacing the
  previous hexagonal core/port/adapter layout: `internal/domain/<entity>`
  (Aggregate Root, Value Objects, Domain Events, repository port,
  sentinel errors), `internal/application/<entity>` (CQRS command/query
  DTOs and an application service), `internal/infrastructure` (HTTP/gRPC
  transport, persistence, the `wire_gen.go` composition root). Role
  inference (aggregate root, value object, domain event, command, query)
  is driven by real vendored proto extensions
  (`import "sgo/options.proto";`), not a second hand-authored modeling
  file. Field-level invariants are enforced for real via
  `buf.build/go/protovalidate`, not hand-rolled CEL evaluation.
- **`sgo list endpoints [service]`** — print every HTTP route currently
  derived for one or all registered services (method, path, source RPC),
  reusing the exact route-derivation function the generated adapter and
  `sgo generate openapi` already call.
- **`sgo openapi ui [--port 4749]`** — serve a project's generated
  OpenAPI document through an embedded, offline
  [Redoc](https://github.com/Redocly/redoc) viewer — a real interactive
  API reference, not raw YAML in a text editor. `sgo generate code` now
  also prints a reminder to run `sgo generate openapi` once a service
  exists.
- **Proto-defined HTTP paths (`google.api.http`)** — an RPC carrying a
  real `(google.api.http)` annotation (vendored
  `google/api/http.proto`/`annotations.proto`, still no `protoc`/`buf`
  or network access required) now drives its generated route's method,
  path, and body directly, with multiple independently-named path
  parameters supported, not just `id`. An RPC without one keeps today's
  `Create`/`Get`/`List`/`Update`/`Delete` naming-convention routing,
  unchanged. `option (sgo.base_path) = "/v2";` on a service overrides the
  default `/api/v1` prefix for every route it derives.
- **Repository-port methods beyond fixed CRUD** — `option
  (sgo.repository_query) = true;` on an RPC adds a matching method to
  the domain repository port (e.g. `FindByEmail(ctx, email string)`),
  alongside its usual application-service method, HTTP route, and gRPC
  method. The in-memory adapter auto-implements it as a linear scan when
  it takes one scalar parameter matching a domain field; Postgres/MySQL/
  MongoDB adapters get an owned companion file with a
  `panic("sgo: TODO implement ...")` stub instead, since real query
  logic can't be auto-generated — a hand-written implementation survives
  every later `sgo generate code` run, the same as `service.go` does.
  `option (sgo.hide_route) = true;` independently skips HTTP route
  registration for an RPC without affecting its gRPC method or
  repository counterpart.
- **README.md rewritten as a full getting-started guide**: table of
  contents, install, an expanded quick start, a project-layout tree
  captured from a real generated project, the generated-vs-owned file
  split, a full `sgo.*`/`google.api.http` proto option reference with a
  worked example, HTTP routing, repository queries beyond CRUD,
  persistence/cache/search, OpenAPI, the web UI, a command reference
  table, an `sgo.yaml` reference, and an FAQ/troubleshooting section.
- **`CONTRIBUTING.md`** — build/test instructions moved out of README.md
  into their own file, expanded with the phase-based
  plan → `AskUserQuestion` → `ARCHITECTURE.md`/`PLAN.md` → implement → PR
  workflow this project actually uses.

### Changed

- HTTP routes are no longer exclusively derived from RPC naming — see
  "Proto-defined HTTP paths" above. Every proto that doesn't opt into a
  `(google.api.http)` annotation keeps generating byte-identical routes.
- The repository port is still a fixed `Create`/`Get`/`List`/`Update`/
  `Delete` shape by default, now extensible per RPC — see
  "Repository-port methods beyond fixed CRUD" above.
- Requires Go 1.26+ (up from 1.25+), picked up automatically via the Go
  toolchain mechanism when adding the real
  `google.golang.org/genproto/googleapis/api/annotations` dependency for
  `google.api.http` support.

### Known gaps

- **Shared kernel** (a value object declared once and reused by multiple
  entities' protos via `import`) is deferred — each entity's value
  objects currently regenerate per file that declares them.
- **GORM value-object field embedding** isn't implemented yet — the ORM
  persistence mode doesn't yet map a value object onto embedded GORM
  columns.
- A `repository_query` method's request must be entirely scalar fields
  (no nested messages, no repeated fields), and its response is always
  the existing single-entity shape — no list-shaped (paginated) custom
  queries yet. Both fail generation with a clear error rather than
  guessing.

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

[0.1.1]: https://github.com/wahyurudiyan/sunny-go/releases/tag/v0.1.1
[0.0.1]: https://github.com/wahyurudiyan/sunny-go/releases/tag/v0.0.1
