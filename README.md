# sgo

[![Go Reference](https://pkg.go.dev/badge/github.com/wahyurudiyan/sunny-go.svg)](https://pkg.go.dev/github.com/wahyurudiyan/sunny-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/wahyurudiyan/sunny-go)](https://goreportcard.com/report/github.com/wahyurudiyan/sunny-go)
[![Go version](https://img.shields.io/github/go-mod/go-version/wahyurudiyan/sunny-go)](go.mod)
[![Latest release](https://img.shields.io/github/v/release/wahyurudiyan/sunny-go?include_prereleases)](https://github.com/wahyurudiyan/sunny-go/releases)

**Write one `.proto` file. Get a real, DDD-layered Go service — HTTP and
gRPC, domain invariants, CQRS, and a persistence adapter — without ever
babysitting the boilerplate again.**

Most generators scaffold once and abandon you. `sgo` keeps working with
you: edit the proto, run `sgo generate code`, and every generated layer
regenerates around the business logic you already wrote — never through
it. Add a field, add an RPC, mark a message as a value object — the
generator adapts, your code doesn't move.

```sh
go install github.com/wahyurudiyan/sunny-go/cmd/sgo@latest

sgo init myservice --http-framework gin --db postgres
cd myservice && sgo generate proto user && sgo generate code user
go run ./cmd/myservice   # HTTP :8080, gRPC :9090, both real, both now
```

> **Status:** every command below is real and running today, not a
> roadmap. See [`docs/CLI.md`](docs/CLI.md) for the full reference and
> [`PLAN.md`](PLAN.md) for what's next.

## Contents

- [Why sgo](#-why-sgo)
- [Install](#-install)
- [Quick start](#-quick-start)
- [How it works](#-how-it-works)
- [Project layout](#-project-layout)
- [Generated vs. owned files](#-generated-vs-owned-files)
- [The domain model: proto custom options](#-the-domain-model-proto-custom-options)
- [HTTP routing](#-http-routing)
- [Repository queries beyond CRUD](#-repository-queries-beyond-crud)
- [Persistence, cache, and search](#-persistence-cache-and-search)
- [OpenAPI documentation](#-openapi-documentation)
- [Run it, debug it live](#-run-it-debug-it-live)
- [The web UI](#-the-web-ui)
- [Command reference](#-command-reference)
- [Configuration (`sgo.yaml`)](#-configuration-sgoyaml)
- [FAQ / troubleshooting](#-faq--troubleshooting)
- [Project docs](#-project-docs)
- [Contributing](#-contributing)

## 🚀 Why sgo

- **Proto-first, for real.** Wire types, domain structs, HTTP routes,
  the OpenAPI doc — every layer is derived by walking the actual
  compiled descriptor of `contract/pb/*.proto`. Not a hand-maintained
  model that quietly drifts from the contract.
- **Zero `protoc`/`buf` install.** Compilation runs through a pure-Go
  compiler ([`bufbuild/protocompile`](https://github.com/bufbuild/protocompile)); wire generation calls the real
  `protoc-gen-go`/`protoc-gen-go-grpc` plugins via `go run
  <module>@<version>`. Clone the repo, run one command — no binary has
  to already be on your `PATH`.
- **A real domain model, not flat structs.** Aggregates, value objects,
  and domain events are inferred from your proto and generated with
  actual invariant enforcement via
  [`buf.build/go/protovalidate`](https://github.com/bufbuild/protovalidate-go)
  — not a second, hand-rolled validation layer bolted on after.
- **Your code is never collateral damage.** Every generated layer
  splits into a `_gen.go` file (rewritten every run) and, wherever
  there's business logic to write, an owned file `sgo` creates once and
  never touches again.
- **Bring your own stack.** Gin, Echo, or Chi for HTTP; Postgres,
  MySQL, or MongoDB for persistence (ORM or hand-written SQL); Redis
  and Elasticsearch on the side. Pick once at `sgo init`, `sgo` wires
  it all together.
- **Live loop, not edit-and-pray.** `sgo run --debug` runs your service
  and gives you a browser dashboard to watch and edit its config in
  real time, restarting it for you when something changes.

## 📦 Install

Requires Go 1.26+.

```sh
go install github.com/wahyurudiyan/sunny-go/cmd/sgo@latest
```

Installs the `sgo` binary to `$(go env GOPATH)/bin` — make sure that's
on your `PATH`. Pin a release with `@v0.2.0` instead of `@latest`; see
[`CHANGELOG.md`](CHANGELOG.md) and the
[releases page](https://github.com/wahyurudiyan/sunny-go/releases) for
what each version ships.

Building from a local clone works the same as any Go module — see
[Contributing](#-contributing).

## ⚡ Quick start

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
selected) a real Postgres repository — `AutoMigrate` runs at startup,
so an empty database works with no separate migration step.

Run `sgo generate code user` again after editing the proto and
everything derived from it regenerates — the aggregate, the repository
port, the DTOs, the mapper, the HTTP routes, the gRPC server — without
touching a line you wrote into `service.go` or `aggregate.go`.

No `sgo.yaml` yet and want to see it before you commit to a terminal?
`sgo ui` opens the same flow in a browser — see [The web UI](#-the-web-ui).

## 🔧 How it works

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

Everything above "implement business logic" is mechanical and safe to
re-run on every proto change. The only thing `sgo` ever hands you a pen
for is what a use case actually *does* — and, optionally, your
aggregate's own business methods.

## 🗂️ Project layout

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

Full design rationale — why domain/application/infrastructure, why a
shared `event` package, why cache/search live outside the DDD layers —
is in [`ARCHITECTURE.md`](ARCHITECTURE.md) §17.

## 🔒 Generated vs. owned files

Every layer that could hold hand-written logic splits in two, and `sgo`
never confuses the two:

| | Generated (`*_gen.go`) | Owned |
|---|---|---|
| Rewritten every run? | Always, fully | Created once, never overwritten |
| Contains | Structs, interfaces, DTOs, routes, mappers — anything fully derivable from the proto | Business logic: aggregate methods, application service bodies |
| Example | `internal/domain/user/user_gen.go` | `internal/domain/user/aggregate.go` |
| | `internal/application/user/command_gen.go` | `internal/application/user/service.go` |

When an owned file needs a **new** method (a new RPC, or a proto change
that adds a required `repository_query` counterpart), `sgo` appends a
`panic("sgo: TODO implement ...")` stub for just that method — your
existing methods are never touched, reformatted, or reordered. If a
method you had is no longer needed, `sgo` leaves it in place with a
one-line warning comment instead of quietly deleting code you might
still rely on.

Backed by an end-to-end test suite that edits a proto and asserts a
hand-written method body survives two consecutive `generate code` runs,
with a real `go build` after each — see `ARCHITECTURE.md` §6.

## 🧬 The domain model: proto custom options

`sgo` infers your DDD model from naming convention by default, with
explicit proto options (`import "sgo/options.proto";`, vendored — no
extra dependency to add) to override it when convention doesn't fit:

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
| `(sgo.obfuscate_visible) = N` | field (`string`) | Masks the value everywhere sgo serializes/logs it: first `N` chars visible, the rest replaced by a fixed-length mask | field shown in full |
| `(sgo.pii) = true` | field | Flags the field `x-sensitive` in generated OpenAPI docs — documentation only, no masking by itself | not flagged |

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
  // Masked in HTTP/gRPC responses and structured logs as "123*****" —
  // the real value is untouched in the repository and never mutated,
  // even though HTTP serializes this exact struct instance.
  string id_number = 4 [(sgo.obfuscate_visible) = 3, (sgo.pii) = true];
}

message Money {
  option (sgo.value_object) = true;
  int64 amount = 1;
  string currency = 2;
}
```

Field-level invariants use the real, widely-adopted
[`buf.build/go/protovalidate`](https://github.com/bufbuild/protovalidate-go)
(`(buf.validate.field) = {...}`), enforced at the wire↔domain boundary
— not a second, sgo-specific validation DSL.

`(sgo.obfuscate_visible)` masks non-destructively — the real value in
the repository is never mutated, even for the HTTP adapter, which
serializes the exact domain struct instance a repository may still hold
a pointer to (a generated `MarshalJSON` shadow-struct handles that; the
gRPC adapter always builds a fresh wire struct per call, so it masks
directly). The same field also gets a generated `slog.LogValuer`, so it
stays redacted in structured logs too. A plain `(sgo.pii)` marker, with
no `obfuscate_visible`, changes nothing about serialization or logging —
it's classification for the generated OpenAPI doc only. A field's real
protobuf `[json_name = "..."]` (not an sgo option — standard proto
syntax) also now overrides its generated JSON key, in the domain struct,
CQRS DTOs, and the OpenAPI doc alike; unset, a field's JSON key is still
its raw proto name, unchanged from every prior release. Full design:
[`ARCHITECTURE.md` §22](ARCHITECTURE.md).

## 🌐 HTTP routing

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

`sgo list endpoints` prints exactly what the generated adapter serves
— convention-derived and annotation-derived routes alike — since it
reuses the same route-derivation function the adapter and `sgo generate
openapi` both call.

## 🔍 Repository queries beyond CRUD

The repository port is a fixed `Create`/`Get`/`List`/`Update`/`Delete`
shape by design (fully decoupled from RPC naming) — but real services
need more, like `FindByEmail`. Mark the RPC:

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
- **In-memory adapter:** auto-implements it as a linear scan, zero
  hand-editing, whenever the method takes exactly one scalar parameter
  whose name matches a domain field (`email` → the `Email` field).
- **Real engines (Postgres/MySQL/MongoDB):** can't have real query
  logic auto-generated — sgo has no way to know what SQL a `FindByEmail`
  needs — so each engine's adapter package gets an *owned* companion
  file (`postgres/user_repository.go`, next to the generated one) with
  a `panic("sgo: TODO implement FindByEmail")` stub. Implement it once;
  it survives every later regeneration, same as `service.go`.

Add `option (sgo.hide_route) = true;` on the same RPC if it shouldn't
also become a public HTTP endpoint — its gRPC method and repository
counterpart are unaffected.

Constraints, stated rather than silently mishandled: every request
field must be scalar (no nested messages, no repeated fields), and the
response is always the single-entity shape `Get`/`Create`/`Update`
already use — no pagination yet. Both fail generation with a clear
error instead of guessing.

## 🗄️ Persistence, cache, and search

Selected once at `sgo init` (`--db`, `--persistence-mode`, `--cache`,
`--search`), persisted to `sgo.yaml`:

- **A default in-memory repository is always generated**, so a service
  is runnable and demoable with zero configuration, even before you
  pick a real datastore.
- **Postgres/MySQL**, `orm` (GORM) or `self-managed` (hand-written SQL)
  per `--persistence-mode`; **MongoDB** via the official driver (no
  mode split). IDs are `google/uuid`-generated on every engine;
  `AutoMigrate` runs at startup.
- **Redis** (cache) and **Elasticsearch** (search) clients are
  generated and connected in `wire_gen.go` when selected, but
  intentionally **not** auto-wired into any service constructor — add
  one as a parameter to `New<Entity>Service` yourself when you actually
  want to use it. Nothing guesses at how you'll use a cache.
- Only the **first** engine in `sgo.yaml`'s `persistence.engines` list
  is wired into `wire_gen.go` — one active persistence engine per
  project, not per entity, today.

## 📖 OpenAPI documentation

```sh
sgo generate openapi              # writes docs/openapi.yaml (or .json)
sgo openapi validate              # validates it against the real OpenAPI meta-schema
sgo openapi ui                    # serves it through an offline Redoc viewer at :4749
```

The document is built from the exact same routes the HTTP adapter
serves — never a hand-maintained second copy — and self-validated
against the real, vendored OpenAPI JSON Schema meta-schema before
being written. `sgo openapi ui` embeds [Redoc](https://github.com/Redocly/redoc)
(`go:embed`, no CDN, works offline) for a real interactive API
reference instead of raw YAML in a text editor.

## 🏃 Run it, debug it live

```sh
sgo run [--debug] [--debug-port 4748]
```

Runs the current project's service the way `go run ./cmd/<name>` would
— `.env` values loaded, real OS/CI environment variables always win
over a leftover `.env` value. Stops cleanly on Ctrl-C.

`--debug` additionally serves a `127.0.0.1`-only dashboard: every
config key currently in effect, its source, and whether it's writable.
Edit a writable value and the service restarts with the new value in
effect, live, over Server-Sent Events — no manual Ctrl-C/rerun. Values
are masked by default; nothing crosses the wire until you click Reveal
for that one key.

## 🖥️ The web UI

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

## 📋 Command reference

| Command | What it does |
|---|---|
| `sgo init [name]` | Scaffold a new project; interactive wizard or flag-driven |
| `sgo generate proto <name>` | Scaffold a starter CRUD proto |
| `sgo generate code <name>` | Compile the proto and regenerate every derived layer |
| `sgo generate openapi` | Write `docs/openapi.<ext>` from every registered service's routes |
| `sgo openapi validate [path]` | Validate an OpenAPI document against the real meta-schema |
| `sgo openapi ui` | Serve the generated OpenAPI doc through an offline Redoc viewer |
| `sgo run [--debug]` | Run the service like `go run` would, optionally with a live config dashboard |
| `sgo list services` | Show each service's generation status |
| `sgo list endpoints [service]` | Print every HTTP route the project currently serves |
| `sgo ui` | Start the localhost web UI |

Full flags, defaults, and behavior for each command:
[`docs/CLI.md`](docs/CLI.md).

## ⚙️ Configuration (`sgo.yaml`)

Written once by `sgo init`, read by every later `sgo generate`/`sgo
list`/`sgo openapi`/`sgo run` command — this is what makes those
commands non-interactive after the first setup:

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

`services` is populated automatically by `sgo generate code` — you
don't edit it by hand. Everything else can be hand-edited (or changed
via `sgo ui`'s `sgo.yaml` editor) before the next `sgo generate code`
run; note that the HTTP framework and DDD layout are chosen once at
`sgo init` time and aren't currently safe to change on an
already-generated project (see [`ARCHITECTURE.md`](ARCHITECTURE.md)
§12).

## ❓ FAQ / troubleshooting

**Do I need `protoc` or `buf` installed?**
No. Proto compilation uses a pure-Go compiler
([`bufbuild/protocompile`](https://github.com/bufbuild/protocompile));
code generation runs the real `protoc-gen-go`/`protoc-gen-go-grpc`
plugins via `go run <module>@<version>`, which only needs a Go
toolchain.

**I hand-edited a generated `_gen.go` file and my changes disappeared.**
Expected — anything in a `_gen.go` file is fully rewritten on every
`sgo generate code` run. Put hand-written logic in the paired owned
file instead (`aggregate.go`, `service.go`, or a persistence adapter's
`<entity>_repository.go` companion file).

**Can I change the HTTP framework or persistence engine after `sgo init`?**
Not today — both are chosen once at `sgo init` time. Safe regeneration
relies on finding owned files at their current expected path and
shape; retargeting either would need a real migration path, which
isn't built yet (tracked as a known limitation, not silently
unsupported).

**How do I add a field or RPC to an existing service?**
Edit `contract/pb/<name>.proto` directly, then re-run `sgo generate
code <name>`. New fields/RPCs get generated support automatically; a
new RPC on the owned application service gets a fresh stub appended,
and your existing method bodies are left untouched.

**Where do I put logic that uses the Redis/Elasticsearch client I selected?**
Neither is auto-wired into any service constructor on purpose — add it
as a parameter to `New<Entity>Service` in
`internal/application/<entity>/service.go` yourself, since only you
know how you actually want to use it.

## 📚 Project docs

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — the target architecture in
  full: the DDD domain/application/infrastructure layout,
  generated-vs-owned file strategy, pluggable HTTP frameworks and
  datastores, every design decision and why it was made.
- [`PLAN.md`](PLAN.md) — the phased delivery plan, what's done and
  what's still ahead.
- [`docs/CLI.md`](docs/CLI.md) — full command reference.
- [`CHANGELOG.md`](CHANGELOG.md) — release history.
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — development setup and the
  contribution workflow this project uses.

## 🤝 Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for building from source,
running the test suite, and the phase-based workflow this project uses
for proposing changes. Issues and PRs welcome.
