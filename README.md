# sunny-go (`sgo`)

`sgo` is an open-source CLI that bootstraps and evolves a Go service from
a single `.proto` contract. One `sgo generate code` run turns a proto
file into a full DDD-layered service — domain aggregates with real
invariant enforcement, a CQRS application layer, pluggable HTTP/gRPC
adapters, and a persistence adapter for the datastore you picked —
without ever touching the business logic you've already written.

> **Status:** in active development, but everything documented below is
> real and working, not aspirational. `sgo init`, `sgo generate
> {proto,code,openapi}`, `sgo openapi {validate,ui}`, `sgo list
> {services,endpoints}`, and `sgo ui` all run today: a generated project
> serves HTTP and gRPC as soon as you implement its application service,
> backed by a real Postgres, MySQL, or MongoDB repository if you selected
> one (in-memory otherwise), with Redis/Elasticsearch clients wired up if
> selected too. See [`docs/CLI.md`](docs/CLI.md) for the exact command
> reference and [`PLAN.md`](PLAN.md) for what's still ahead.

## Table of contents

- [Why sgo](#why-sgo)
- [Install](#install)
- [Quick start](#quick-start)
- [How it works](#how-it-works)
- [Project layout](#project-layout)
- [Generated vs. owned files](#generated-vs-owned-files)
- [The domain model: proto custom options](#the-domain-model-proto-custom-options)
- [HTTP routing](#http-routing)
- [Repository queries beyond CRUD](#repository-queries-beyond-crud)
- [Persistence, cache, and search](#persistence-cache-and-search)
- [OpenAPI documentation](#openapi-documentation)
- [The web UI](#the-web-ui)
- [Command reference](#command-reference)
- [Configuration (`sgo.yaml`)](#configuration-sgoyaml)
- [FAQ / troubleshooting](#faq--troubleshooting)
- [Project docs](#project-docs)
- [Contributing](#contributing)

## Why sgo

Most Go service generators stop at scaffolding a folder structure once.
`sgo` treats the proto file as the durable source of truth and the
generated code as something you keep regenerating as the contract
grows — safely, because it never rewrites a line of business logic you
wrote yourself:

- **Proto-first, always.** Every layer — wire types, domain structs,
  HTTP routes, the OpenAPI document — is derived from `contract/pb/*.proto`
  by walking the real compiled descriptor, not a hand-maintained parallel
  model.
- **No `protoc`/`buf` install required.** Proto compilation goes through
  a pure-Go compiler ([`bufbuild/protocompile`](https://github.com/bufbuild/protocompile)); wire types come from the
  real `protoc-gen-go`/`protoc-gen-go-grpc` plugins, run via `go run
  <module>@<version>` so no binary needs to be on `PATH`.
- **A real domain model, not flat structs.** Aggregates, value objects,
  and domain events are inferred from the proto (by naming convention or
  explicit `sgo.*` options) and generated with actual invariant
  enforcement via [`buf.build/go/protovalidate`](https://github.com/bufbuild/protovalidate-go) — not a second,
  hand-rolled validation layer.
- **Hand-written code survives regeneration.** Every generated layer
  splits into a `_gen.go` file (always overwritten) and, where there's
  business logic to write, an owned file `sgo` creates once and never
  overwrites again — see [Generated vs. owned files](#generated-vs-owned-files).
- **You choose the stack, `sgo` wires it up.** Gin, Echo, or Chi for
  HTTP; Postgres, MySQL, or MongoDB for persistence (ORM or hand-written
  SQL); Redis and Elasticsearch as optional clients. Everything is
  selected once at `sgo init` and persisted to `sgo.yaml`.

## Install

Requires Go 1.26+.

```sh
go install github.com/wahyurudiyan/sunny-go/cmd/sgo@latest
```

This installs the `sgo` binary to `$(go env GOPATH)/bin` — make sure
that's on your `PATH`. Pin a specific release with `@v0.0.1` instead of
`@latest`; see [`CHANGELOG.md`](CHANGELOG.md) and the
[releases page](https://github.com/wahyurudiyan/sunny-go/releases) for
what each version contains.

Building from a local clone works the same way any Go module does —
see [Contributing](#contributing).

## Quick start

```sh
sgo init myservice --http-framework gin --persistence-mode orm --db postgres
cd myservice
sgo generate proto user
# edit contract/pb/user.proto — add fields, RPCs, or custom sgo options
sgo generate code user
# implement business logic in internal/application/user/service.go
go run ./cmd/myservice
```

`myservice` now serves HTTP on `:8080` and gRPC on `:9090`, both backed
by the same application service instance and (since `--db postgres` was
selected) a real Postgres repository — `AutoMigrate` runs at startup, so
an empty database works with no separate migration step.

Running `sgo generate code user` again after editing the proto
regenerates everything derived from it — the aggregate, the repository
port, the DTOs, the mapper, the HTTP routes, the gRPC server — without
touching anything you wrote into `service.go` or `aggregate.go`. Add a
field, add an RPC, mark a message as a value object: re-run the command,
and only what actually needs to change does.

No `sgo.yaml` yet and want to explore visually instead? `sgo ui` opens
the same flow in a browser — see [The web UI](#the-web-ui).

## How it works

```
contract/pb/*.proto
       │  sgo generate code <name>
       ▼
┌─────────────────────────────────────────────────────────────┐
│ contract/gen/<name>/       real protoc-gen-go(-grpc) output   │
│ internal/domain/<name>/    aggregate, value objects, events,  │
│                             repository port, sentinel errors  │
│ internal/application/<name>/  CQRS command/query DTOs +        │
│                                 application service (owned)    │
│ internal/infrastructure/…  HTTP + gRPC adapters, persistence,  │
│                              wire_gen.go composition root       │
└─────────────────────────────────────────────────────────────┘
       │  go run ./cmd/<project>
       ▼
   HTTP :8080  +  gRPC :9090
```

Everything above the `implement business logic` step is mechanical and
safe to re-run on every proto change. The only thing `sgo` ever asks a
human to write is the body of the application service (what a use case
actually does) and, optionally, aggregate business methods.

## Project layout

A freshly generated project (`--http-framework gin --db postgres --cache
redis --search elasticsearch`, one `user` service) looks like this:

```
myservice/
├── sgo.yaml                              # project manifest — see Configuration below
├── contract/
│   ├── pb/user.proto                     # hand-authored — the only thing you edit by hand here
│   └── gen/user/                         # real protoc-gen-go(-grpc) output
│       ├── user.pb.go
│       └── user_grpc.pb.go
├── internal/
│   ├── domain/
│   │   ├── event/event.go                # shared DomainEvent interface (all entities)
│   │   └── user/
│   │       ├── user_gen.go               # aggregate root, value objects, domain events
│   │       ├── aggregate.go              # OWNED — business methods, event recording
│   │       ├── repository.go             # fixed CRUD port (+ repository_query methods)
│   │       └── errors.go
│   ├── application/
│   │   ├── ports/event_publisher.go      # shared EventPublisher interface + no-op default
│   │   └── user/
│   │       ├── command_gen.go            # CQRS command DTOs
│   │       ├── query_gen.go              # CQRS query DTOs
│   │       └── service.go                # OWNED — implement your use cases here
│   ├── infrastructure/
│   │   ├── transport/
│   │   │   ├── user_mapper_gen.go        # wire ↔ domain/application conversions
│   │   │   ├── http/gin/                 # server_gen.go + user_routes_gen.go
│   │   │   └── grpc/user_grpc_server_gen.go
│   │   ├── persistence/
│   │   │   ├── memory/user_repository_gen.go     # always generated, always runnable
│   │   │   └── postgres/                          # real adapter, since --db postgres
│   │   │       ├── conn_gen.go
│   │   │       └── user_repository_gen.go
│   │   └── bootstrap/wire_gen.go         # composition root — wires every service
│   ├── adapter/out/cache/redis/          # since --cache redis
│   └── core/port/out/cache.go            # cache/search ports (project-scoped, not per-entity)
├── cmd/myservice/main.go
├── docker/{Dockerfile,docker-compose.yml}
└── go.mod
```

Full design rationale for this layout — why domain/application/
infrastructure, why a shared `event` package, why cache/search stay
outside the DDD layers — is in [`ARCHITECTURE.md`](ARCHITECTURE.md) §17.

## Generated vs. owned files

Every layer that could contain hand-written logic splits in two:

| | Generated (`*_gen.go`) | Owned |
|---|---|---|
| Rewritten every run? | Always, fully | Created once, never overwritten |
| Contains | Structs, interfaces, DTOs, routes, mappers — anything fully derivable from the proto | Business logic: aggregate methods, application service bodies |
| Example | `internal/domain/user/user_gen.go` | `internal/domain/user/aggregate.go` |
| | `internal/application/user/command_gen.go` | `internal/application/user/service.go` |

When an owned file needs a **new** method (you added an RPC, or a proto
change added a required `repository_query` counterpart), `sgo` appends
a `panic("sgo: TODO implement ...")` stub for just that method — your
existing methods are never touched, reformatted, or reordered. If a
method that used to exist is no longer needed, `sgo` leaves it in place
with a one-line warning comment instead of silently deleting code you
might still be using elsewhere.

This is covered by an end-to-end test suite that edits a proto and
asserts a hand-written method body survives two consecutive `generate
code` runs, with a real `go build` after each — see `ARCHITECTURE.md`
§6.

## The domain model: proto custom options

`sgo` infers your DDD model from naming convention by default, with
explicit proto options (`import "sgo/options.proto";`, vendored — no
extra dependency to add) to override it when the convention doesn't fit:

| Option | Applies to | Effect | Default (unmarked) |
|---|---|---|---|
| `(sgo.aggregate_root) = true` | message | Marks the Aggregate Root explicitly | The message matching the entity name |
| `(sgo.value_object) = true` | message | Generated with no setters + constructor-time validation | — |
| `(sgo.domain_event) = true` | message | Generated as a domain event struct implementing `DomainEvent` | — |
| `(sgo.command) = true` | RPC | Classified as a CQRS command | `Create`/`Update`/`Delete` prefix |
| `(sgo.query) = true` | RPC | Classified as a CQRS query | `Get`/`List` prefix |
| `(sgo.repository_query) = true` | RPC | Adds a matching method to the repository port | not added |
| `(sgo.hide_route) = true` | RPC | Skips HTTP route registration for this RPC | route registered |
| `(sgo.base_path) = "/v2"` | service | Overrides the default `/api/v1` route prefix | `/api/v1` |

Example:

```proto
syntax = "proto3";
package user.v1;

import "sgo/options.proto";

option go_package = "myservice/contract/gen/user";

service UserService {
  option (sgo.base_path) = "/v2";

  rpc CreateUser(CreateUserRequest) returns (UserResponse);
  rpc GetUser(GetUserRequest) returns (UserResponse);

  rpc FindUserByEmail(FindUserByEmailRequest) returns (UserResponse) {
    option (sgo.repository_query) = true;
  }
}

message User {
  option (sgo.aggregate_root) = true;
  string id = 1;
  string name = 2;
  string email = 3;
}

message Money {
  option (sgo.value_object) = true;
  int64 amount = 1;
  string currency = 2;
}
```

Field-level invariants use the real, widely-adopted
[`buf.build/go/protovalidate`](https://github.com/bufbuild/protovalidate-go)
(`(buf.validate.field) = {...}`), enforced at the wire↔domain boundary —
not a second, sgo-specific validation DSL.

## HTTP routing

An RPC with no annotation gets a route derived from its name:

| RPC prefix | Verb | Path |
|---|---|---|
| `Create*` | `POST` | `/api/v1/<entity>s` |
| `Get*` | `GET` | `/api/v1/<entity>s/{id}` |
| `List*` | `GET` | `/api/v1/<entity>s` |
| `Update*` | `PUT` | `/api/v1/<entity>s/{id}` |
| `Delete*` | `DELETE` | `/api/v1/<entity>s/{id}` |
| anything else | `POST` | `/api/v1/<entity>s/{id}` if the request has an `id` field, else no path param |

An RPC carrying a real `(google.api.http)` annotation (the same
extension [`google/api/http.proto`](https://github.com/googleapis/googleapis/blob/master/google/api/http.proto)
uses, vendored so no `googleapis` dependency is needed) uses its
declared method, path, and body instead — multiple named path
parameters, not just `id`:

```proto
rpc ArchiveUser(ArchiveUserRequest) returns (UserResponse) {
  option (google.api.http) = {
    post: "/accounts/{account_id}/archive/{reason_code}"
  };
}
```

`sgo list endpoints` prints exactly what the generated adapter serves —
convention-derived and annotation-derived routes alike — since it
reuses the same route-derivation function the adapter and `sgo generate
openapi` both call.

## Repository queries beyond CRUD

The repository port is a fixed `Create`/`Get`/`List`/`Update`/`Delete`
shape by design (so it stays fully decoupled from RPC naming) — but a
real service usually needs more, like `FindByEmail`. Mark the RPC:

```proto
rpc FindUserByEmail(FindUserByEmailRequest) returns (UserResponse) {
  option (sgo.repository_query) = true;
}
```

`sgo generate code` then:

- Adds `FindByEmail(ctx context.Context, email string) (*User, error)`
  to `internal/domain/user/repository.go` — the method name strips the
  entity's own name from the RPC name (`FindUserByEmail` → `FindByEmail`
  on entity `user`); one that doesn't mention the entity keeps its own
  name.
- **In-memory adapter:** auto-implements it as a linear scan, with zero
  hand-editing, whenever the method takes exactly one scalar parameter
  whose name matches a domain field (`email` → the `Email` field).
- **Real engines (Postgres/MySQL/MongoDB):** can't have real query logic
  auto-generated — sgo has no way to know what SQL a `FindByEmail` needs
  — so each engine's adapter package gets an *owned* companion file
  (`postgres/user_repository.go`, next to the generated one) with a
  `panic("sgo: TODO implement FindByEmail")` stub. Implement it once; it
  survives every later regeneration, the same as `service.go` does.

Add `option (sgo.hide_route) = true;` on the same RPC if it shouldn't
also become a public HTTP endpoint — its gRPC method and repository
counterpart are unaffected.

Constraints, stated rather than silently mishandled: every request field
must be scalar (no nested messages, no repeated fields), and the
response is always the single-entity shape `Get`/`Create`/`Update`
already use — no pagination yet. Both fail generation with a clear
error instead of guessing.

## Persistence, cache, and search

Selected once at `sgo init` (`--db`, `--persistence-mode`, `--cache`,
`--search`), persisted to `sgo.yaml`:

- **A default in-memory repository is always generated**, so a service
  is runnable and demoable with zero configuration, even before you pick
  a real datastore.
- **Postgres/MySQL**, `orm` (GORM) or `self-managed` (hand-written SQL)
  per `--persistence-mode`; **MongoDB** via the official driver (no mode
  split). IDs are `google/uuid`-generated on every engine; `AutoMigrate`
  runs at startup.
- **Redis** (cache) and **Elasticsearch** (search) clients are generated
  and connected in `wire_gen.go` when selected, but intentionally **not**
  auto-wired into any service constructor — add one as a parameter to
  `New<Entity>Service` yourself when you actually want to use it. Nothing
  guesses at how you'll use a cache.
- Only the **first** engine in `sgo.yaml`'s `persistence.engines` list is
  wired into `wire_gen.go` — one active persistence engine per project,
  not per entity, today.

## OpenAPI documentation

```sh
sgo generate openapi              # writes docs/openapi.yaml (or .json)
sgo openapi validate              # validates it against the real OpenAPI meta-schema
sgo openapi ui                    # serves it through an offline Redoc viewer at :4749
```

The document is built from the exact same routes the HTTP adapter
serves — never a hand-maintained second copy — and self-validated
against the real, vendored OpenAPI JSON Schema meta-schema before being
written. `sgo openapi ui` embeds [Redoc](https://github.com/Redocly/redoc)
(`go:embed`, no CDN, works offline) for a real interactive API
reference instead of raw YAML in a text editor.

## The web UI

```sh
sgo ui   # http://127.0.0.1:4747
```

Covers the same ground as the CLI from a browser: a create-project form
outside a project, and inside one, a dashboard with an `sgo.yaml`
viewer/editor, a services list with per-stage generation status, a
new-service form, per-service "Generate code", and a proto editor.
Every action calls the exact same `internal/codegen`/`internal/config`
functions the CLI commands do — verified by a parity test that drives
both surfaces through the same sequence and asserts byte-identical
output. Binds to `127.0.0.1` only; no auth, since it's never exposed
beyond loopback.

## Command reference

| Command | What it does |
|---|---|
| `sgo init [name]` | Scaffold a new project; interactive wizard or flag-driven |
| `sgo generate proto <name>` | Scaffold a starter CRUD proto |
| `sgo generate code <name>` | Compile the proto and regenerate every derived layer |
| `sgo generate openapi` | Write `docs/openapi.<ext>` from every registered service's routes |
| `sgo openapi validate [path]` | Validate an OpenAPI document against the real meta-schema |
| `sgo openapi ui` | Serve the generated OpenAPI doc through an offline Redoc viewer |
| `sgo list services` | Show each service's generation status |
| `sgo list endpoints [service]` | Print every HTTP route the project currently serves |
| `sgo ui` | Start the localhost web UI |

Full flags, defaults, and behavior for each command: [`docs/CLI.md`](docs/CLI.md).

## Configuration (`sgo.yaml`)

Written once by `sgo init`, read by every later `sgo generate`/`sgo
list`/`sgo openapi` command — this is what makes those commands
non-interactive after the first setup:

```yaml
module: myservice
httpFramework: gin
persistence:
    mode: orm
    engines:
        - postgres
cache:
    - redis
search:
    - elasticsearch
openapi:
    version: "3.0"
    format: yaml
services:
    - user
```

`services` is populated automatically by `sgo generate code` — you don't
edit it by hand. Everything else can be hand-edited (or changed via `sgo
ui`'s `sgo.yaml` editor) before the next `sgo generate code` run; note
that the HTTP framework and DDD layout are chosen once at `sgo init`
time and aren't currently safe to change on an already-generated
project (see [`ARCHITECTURE.md`](ARCHITECTURE.md) §12).

## FAQ / troubleshooting

**Do I need `protoc` or `buf` installed?**
No. Proto compilation uses a pure-Go compiler
([`bufbuild/protocompile`](https://github.com/bufbuild/protocompile)); code generation
runs the real `protoc-gen-go`/`protoc-gen-go-grpc` plugins via `go run
<module>@<version>`, which only needs a Go toolchain.

**I hand-edited a generated `_gen.go` file and my changes disappeared.**
Expected — anything in a `_gen.go` file is fully rewritten on every
`sgo generate code` run. Put hand-written logic in the paired owned file
instead (`aggregate.go`, `service.go`, or a persistence adapter's
`<entity>_repository.go` companion file).

**Can I change the HTTP framework or persistence engine after `sgo init`?**
Not today — both are chosen once at `sgo init` time. Safe regeneration
relies on finding owned files at their current expected path and shape;
retargeting either would need a real migration path, which isn't built
yet (tracked as a known limitation, not silently unsupported).

**How do I add a field or RPC to an existing service?**
Edit `contract/pb/<name>.proto` directly, then re-run `sgo generate code
<name>`. New fields/RPCs get generated support automatically; a new RPC
on the owned application service gets a fresh stub appended, and your
existing method bodies are left untouched.

**Where do I put logic that uses the Redis/Elasticsearch client I selected?**
Neither is auto-wired into any service constructor on purpose — add it
as a parameter to `New<Entity>Service` in `internal/application/<entity>/service.go`
yourself, since only you know how you actually want to use it.

## Project docs

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — the target architecture in full:
  the DDD domain/application/infrastructure layout, generated-vs-owned
  file strategy, pluggable HTTP frameworks and datastores, every design
  decision and why it was made.
- [`PLAN.md`](PLAN.md) — the phased delivery plan, what's done and what's
  still ahead.
- [`docs/CLI.md`](docs/CLI.md) — full command reference.
- [`CHANGELOG.md`](CHANGELOG.md) — release history.
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — development setup and the
  contribution workflow this project uses.

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for building from source,
running the test suite, and the phase-based workflow this project uses
for proposing changes.
