# `sgo` command reference

Target command surface for the architecture in `ARCHITECTURE.md`. Not all
commands exist yet — see `PLAN.md` for phasing. Commands below marked
**(planned)** don't exist on the current branch.

## `sgo init` ✅

```
sgo init [project-name] [flags]
```

Scaffolds a new project (ARCHITECTURE §3) and writes `sgo.yaml`. Fails if
`<project-name>` already exists in the current directory.

- **No selection flags, stdin is a TTY** → launches the interactive
  wizard (ARCHITECTURE §10, `internal/wizard`, built on `huh`): project
  name (pre-filled if given as an argument), module path, HTTP framework,
  persistence mode, datastores, Redis cache, Elasticsearch search.
- **Any selection flag given, or stdin isn't a TTY** → scaffolds directly
  from flags/defaults, no prompts. This is also the automatic fallback in
  CI/scripts/pipes, with no separate flag needed to request it.

Flags (any one of these, if set, skips the wizard):

| Flag | Values | Default | Notes |
|---|---|---|---|
| `--module` | Go module path | `<project-name>` | |
| `--http-framework` | `gin`, `echo`, `chi` | `gin` | |
| `--persistence-mode` | `orm`, `self-managed` | `orm` | |
| `--db` | comma list of `postgres`, `mysql`, `mongo` | none | |
| `--cache` | comma list, currently only `redis` | none | |
| `--search` | comma list, currently only `elasticsearch` | none | |

There's no `--yes` flag — a non-TTY stdin already skips the wizard
automatically, and any single selection flag signals "I want direct
control" just as well.

Replaces the old `sunny create`/`sunny new`.

## `sgo generate proto <name>` ✅

```
sgo generate proto user
```

Creates `contract/pb/user.proto` from a starter service+CRUD template
(`CreateUser`/`GetUser`/`ListUsers`/`UpdateUser`/`DeleteUser` RPCs, no
imports). Fails if the file already exists (edit it directly, don't
regenerate a proto over hand-written changes). Must be run inside an sgo
project (a directory with `sgo.yaml`) — the proto's `go_package` option
is derived from the project's module path.

## `sgo generate code <name>` ✅

```
sgo generate code user
```

Compiles `contract/pb/user.proto` (via a pure-Go compiler — no `buf`/
`protoc` needed, see ARCHITECTURE §4) and regenerates:

- `contract/gen/user/*.pb.go`, `*_grpc.pb.go` — via the real
  `protoc-gen-go`/`protoc-gen-go-grpc` plugins, run with `go run
  <module>@<version>`
- **Domain layer** (`internal/domain/user/`): `user_gen.go` — the
  Aggregate Root (the message matching the entity name, or explicitly
  marked `option (sgo.aggregate_root) = true;`), its Value Objects
  (`option (sgo.value_object) = true;`) and Domain Events (`option
  (sgo.domain_event) = true;`) — plus `repository.go` (the fixed
  `Create/Get/List/Update/Delete` port, typed directly against the
  aggregate) and `errors.go`. `internal/domain/event/event.go` — the
  shared `DomainEvent` interface every entity's events implement, so one
  `EventPublisher` can accept events from any of them
- **Application layer** (`internal/application/user/`):
  `command_gen.go`/`query_gen.go` — DTOs derived from each RPC's request
  message, classified by naming convention (`Create/Update/Delete` →
  command, `Get/List` → query, overridable with `(sgo.command)`/
  `(sgo.query)`). `internal/application/ports/event_publisher.go` — the
  shared `EventPublisher` interface and its no-op default
- `internal/infrastructure/transport/user_mapper_gen.go` — wire↔domain
  conversions, validated against any `(buf.validate.field)` constraints
  the proto declares (real `buf.build/go/protovalidate`, not sgo
  hand-rolling CEL evaluation) — plus wire↔application conversions for
  the gRPC adapter's own request-side mapping
- HTTP route registration for the project's `sgo.yaml`-selected framework
  (`internal/infrastructure/transport/http/<framework>/user_routes_gen.go`
  + `server_gen.go`), with routes derived from the RPC naming convention
  (`Create*`→`POST`, `Get*`→`GET .../{id}`, `List*`→`GET`,
  `Update*`→`PUT .../{id}`, `Delete*`→`DELETE .../{id}` — no
  `google.api.http` support yet, see ARCHITECTURE §8.1/§12)
- A gRPC server adapter
  (`internal/infrastructure/transport/grpc/user_grpc_server_gen.go`)
  implementing the real protoc-gen-go-grpc server interface
- A default in-memory repository
  (`internal/infrastructure/persistence/memory/user_repository_gen.go`)
  — always generated, so the service is runnable even with no
  persistence engine selected
- If `sgo.yaml` selects a persistence engine (`--db` at `sgo init`): the
  real repository adapter for it —
  `internal/infrastructure/persistence/<engine>/user_repository_gen.go`
  + `conn_gen.go`. Postgres/MySQL support both `orm` (GORM) and
  `self-managed` (hand-written SQL) modes, per `--persistence-mode`;
  MongoDB uses the official driver (no mode split). IDs are
  `google/uuid`-generated on every engine. `AutoMigrate` runs at startup
  so a fresh, empty database works without a separate migration step.
- If `sgo.yaml` selects `redis`/`elasticsearch` (`--cache`/`--search` at
  `sgo init`): the `Cache`/`Search` ports
  (`internal/core/port/out/cache.go`/`search.go` — intentionally still
  here, out of the DDD layers' scope) and their adapters
  (`internal/adapter/out/cache/redis/`,
  `internal/adapter/out/search/elasticsearch/`). Connected in
  `wire_gen.go` if selected, but **not** auto-wired into any service —
  add one as a parameter to `New<Entity>Service` yourself
  (`internal/application/user/service.go`) if you want to use it.
- `internal/infrastructure/bootstrap/wire_gen.go` — regenerated to wire
  *every* service on record (not just this one) to whichever repository
  is active (the real adapter if one is selected, otherwise in-memory)
  and a shared no-op `EventPublisher`, and starts both servers: HTTP on
  `:8080`, gRPC on `:9090`

...and **creates, but never overwrites**, the owned files:
`internal/domain/user/aggregate.go`,
`internal/application/user/service.go`. If the application service
gained methods since the last run (a new RPC), stubs are appended to the
owned service file instead of the whole file being rewritten; if it lost
one that was already implemented, that implementation is left in place
with a warning comment, never deleted — see ARCHITECTURE §6/§17. Safe to
run repeatedly: covered by an end-to-end Ginkgo suite that edits the
proto and asserts a hand-written method body survives, twice, with a
real `go build` after each run, plus a separate suite that builds and
runs the actual compiled binary and drives a full HTTP CRUD cycle
against it over real sockets.

Finishes by running `go mod tidy` in the project, so the
`google.golang.org/protobuf`/`google.golang.org/grpc` dependencies
`contract/gen` and the adapters now need are picked up automatically, and
records `user` in `sgo.yaml`'s `services` list.

Supersedes the old `sunny generate contract proto` / `sunny generate
api` / `sunny generate service` three-step flow — `generate code` is one
step that does entity + ports + service + mapper + HTTP + gRPC together,
because they all derive from the same proto walk. Unlike the old `sunny
generate api`, this one actually reads the proto's contents.

## `sgo list services` ✅

```
sgo list services
```

Lists services tracked in `sgo.yaml` (populated by `sgo generate code`),
and for each: proto present? `contract/gen` present? domain entity
present? service implementation present?

## `sgo validate <proto-file>` — not yet implemented

Not currently a command. Proto errors currently surface as part of
`sgo generate code`'s compile step (clear, position-annotated messages —
see `internal/codegen/proto.Compile`); a standalone `validate` that does
only that step without generating anything is straightforward to add
later if wanted, just not built yet.

## `sgo ui` ✅

```
sgo ui [--port 4747]
```

Starts a localhost-only web UI (ARCHITECTURE §11) at
`http://127.0.0.1:<port>` (default `4747`). Run it from a directory with
no `sgo.yaml` yet and it shows a create-project form covering the same
selections as `sgo init`'s wizard; run it from inside an existing
project and it shows a dashboard: the config summary, a raw `sgo.yaml`
viewer/editor, a new-service form (`sgo generate proto` equivalent), a
services list with generated-vs-owned status per service (proto /
contract/gen / domain entity / service implementation), a
"Generate code" button per service (`sgo generate code` equivalent), and
a raw proto viewer/editor per service.

Creating a project moves the server onto it, so the dashboard for the
project you just created shows up immediately — no restart needed,
equivalent to `sgo init myservice && cd myservice`.

Every action calls the same `internal/codegen`/`internal/config`
functions the CLI commands above call, never the `sgo` binary itself —
verified by a test that drives both surfaces through the same sequence
and asserts byte-identical generated output
(`internal/commands/webui_parity_test.go`). No auth, since it never
listens on anything but loopback.

## `sgo generate openapi` ✅

```
sgo generate openapi [--version 3.0|3.1] [--format yaml|json]
```

Recompiles every service in `sgo.yaml`'s `services` list and writes a
project-wide `docs/openapi.yaml` (or `.json`) from their already-derived
HTTP routes (ARCHITECTURE §13) — a documentation output of the
proto-first pipeline, not a second source of truth; `contract/pb` is
still the only thing you hand-edit. Always fully overwritten, like
`contract/gen`.

`--version`/`--format` override `sgo.yaml`'s `openapi.version`/
`openapi.format` for this one run only, without changing the persisted
selection (set that at `sgo init` with `--openapi-version`/
`--openapi-format`, or later through `sgo ui`'s `sgo.yaml` editor).

Validates the file it just wrote against the real OpenAPI meta-schema
before reporting success — the same check `sgo openapi validate` runs on
demand. If that fails, it's a bug in `sgo` itself.

Fails clearly instead of writing a wrong-but-valid document if two
services would collide: a same-named message defined differently by two
proto files, or two RPCs whose derived route is the same HTTP verb and
path (rename one to fit the Create/Get/List/Update/Delete convention).

## `sgo openapi validate` ✅

```
sgo openapi validate [path]
```

Validates an OpenAPI document — json or yaml, 3.0.x or 3.1.x
(auto-detected from its own `openapi:` field) — against the real,
vendored OpenAPI JSON Schema meta-schema. This checks the document's own
well-formedness, not whether it matches any particular project's
generated routes.

With no path, validates the current project's own generated
`docs/openapi.<ext>` (run `sgo generate openapi` first if it doesn't
exist yet). With a path, validates that file instead — `sgo`-generated
or hand-authored/imported, and doesn't need to be run from inside an
`sgo` project at all.

## Removed/renamed from the current CLI

| Current (`sunny`) | New (`sgo`) | Why |
|---|---|---|
| `sunny create` / `sunny new` | `sgo init` | Matches the spec's wording and common CLI convention (`npm init`, `git init`). |
| `sunny generate contract proto <name>` | `sgo generate proto <name>` | Same behavior, shorter path. |
| `sunny generate api <name>` + `sunny generate service <name>` | `sgo generate code <name>` | Collapsed into one command since both derive from the same proto descriptor walk; the old two-step flow also didn't actually read the proto contents (see prior analysis) — the new one does. |
| `--http-framework fiber\|gin\|echo` | `--http-framework gin\|echo\|chi` (in `sgo init`) | Fiber dropped, Chi added, per spec #4. Framework choice moves from a per-`create` flag with no follow-through to a `sgo.yaml`-persisted, adapter-backed choice. |
