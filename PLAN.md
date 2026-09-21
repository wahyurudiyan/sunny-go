# sgo Development Plan

Phased roadmap implementing `ARCHITECTURE.md`. Each phase has a concrete
exit criterion so we know when to move on. Phases are ordered by
dependency, not necessarily by priority — see notes per phase.

## Phase 0 — Stabilize & clear the ground

This phase clears dead/broken code and lands the test framework; the
real command surface (`init`/`generate`) is intentionally thin-to-absent
until Phases 1–2 rebuild it properly.

- [x] Restore/rewrite root `README.md` (had accidentally been overwritten
      with a generated child project's README).
- [x] Rename the binary/`Use` field from `sunny` to `sgo`; update
      `Makefile` build target accordingly. Module path stays
      `github.com/wahyurudiyan/sunny-go` (see ARCHITECTURE §Decisions #1).
      Also fixed the root `Makefile` itself, which had the same
      accidental-overwrite problem as the README (referenced a
      nonexistent `proto/` dir and `bin/test-setup`).
- [x] Delete `internal/templates/`, `internal/generate/`,
      `internal/create/`, and `internal/plugins/protoc-gen-go-http/`
      (confirmed dead/disconnected in the prior analysis) — superseded by
      `internal/codegen/` in later phases. Along with them, delete
      `internal/commands/create.go`, `generate.go`, and `list.go`, which
      depended on those packages (this also removes the duplicate
      `init()` that was double-registering `generateCmd` — moot once the
      file is gone). `sgo init`, `sgo generate proto/code`, and `sgo list`
      return in Phases 1–2 built against the new `internal/codegen`.
      Kept `internal/banner/` and the root/help command skeleton.
- [x] Adopt Ginkgo v2 + Gomega as the test framework (ARCHITECTURE
      "Testing strategy" section); added as `go.mod` dependencies.
- [x] Add `internal/config` package with the `sgo.yaml` schema (read,
      write, validate) from ARCHITECTURE §7, with a Ginkgo BDD suite
      (`config_suite_test.go` + `config_test.go`) — the reference example
      for how every later package is tested.

**Exit criteria:** `go build ./...` and `go vet ./...` clean, no dead
generator code left, `sgo.yaml` can be round-tripped (write → read →
equal) under a passing Ginkgo suite, root README and Makefile accurate.
All met on this branch.

## Phase 1 — Project scaffolding (`sgo init`, non-interactive) ✅

- [x] `sgo init <project> --http-framework <gin|echo|chi>
      --persistence-mode <orm|self-managed> --db <postgres,mysql,...>
      --cache redis --search elasticsearch` (flags only, no wizard yet).
- [x] Directory scaffolding per ARCHITECTURE §3, conditioned on what was
      selected (no `mongo/` adapter dir if Mongo wasn't picked, etc.).
- [x] `go:embed`-based template engine in `internal/template`, replacing
      the old string-constant approach. Used for the static files
      (`go.mod`, `Makefile`, `Dockerfile`, `main.go`); `docker-compose.yml`
      is instead built as a Go struct and marshaled with `yaml.Marshal` —
      conditional service blocks in text/template got unreadable fast, and
      a struct guarantees valid YAML for any combination of selections,
      including zero datastores.
- [x] `docker/docker-compose.yml` generated with only the datastore
      services actually selected.
- [x] `sgo.yaml` written with the resolved selections.
- [x] Ginkgo specs for `internal/template` and
      `internal/codegen/project`, including one that actually shells out
      to `go build ./...` on a scaffolded project (the exit criterion
      below, as a regression test, not a one-off manual check).

**Exit criteria:** `sgo init demo --http-framework gin --persistence-mode
orm --db postgres` produces a project that `go build`s (even though the
core/domain/service is still empty at this point) and whose
`docker-compose.yml` brings up exactly one Postgres service. Verified
both manually and by `internal/codegen/project`'s Ginkgo suite.

## Phase 2 — Proto → entities, ports, service skeletons ✅

This is the heart of requirements #1–#3.

- [x] `sgo generate proto <name>` → `contract/pb/<name>.proto` scaffold
      (`internal/codegen/proto.GenerateStub`).
- [x] `sgo generate code <name>` (`internal/codegen.GenerateCode`
      orchestrates all of the below):
  - [x] Compile with `bufbuild/protocompile` (pure Go — **not** `buf`/
        `protoc`; see ARCHITECTURE §4 and Decision #5) → descriptor.
  - [x] Walk the descriptor into an IR (`internal/codegen/proto`:
        `File`/`Message`/`Field`/`Service`/`Method`) rather than working
        with `protoreflect` directly everywhere else.
  - [x] Run the real `protoc-gen-go`/`protoc-gen-go-grpc` plugins
        (`internal/codegen/wiregen`, via `go run <module>@<pinned
        version>`) into `contract/gen/<name>/`.
  - [x] Generate `core/domain/<entity>/<entity>_gen.go` (every message in
        the file, not just the entity itself — see ARCHITECTURE §5) +
        `<entity>.go` (owned, created once).
  - [x] Generate `core/port/in/<entity>_usecase.go` (mirrors the proto
        service's RPCs) and `core/port/out/<entity>_repository.go` (fixed
        CRUD shape, see ARCHITECTURE §8.2 and Decision #11).
  - [x] Generate `core/service/<entity>_service.go` (owned, created once
        with panic-stub method bodies for each usecase method).
  - [x] Generate the wire↔domain mapper
        (`internal/adapter/mapper/<entity>_mapper_gen.go`, always
        overwritten, including recursive mapping for nested/repeated
        message fields).
  - [x] Run `go mod tidy` in the project afterward so the
        protobuf/grpc dependencies `contract/gen` now needs are picked up
        automatically — the generated project builds without the
        developer having to know to do this themselves.
- [x] **Safe-regeneration**: implemented as described in ARCHITECTURE §6
      (`internal/codegen/core.ensureMethods`) — `go/parser` to find
      existing methods by receiver+name, line-based insertion for the
      orphan-method warning comment, append for missing-method stubs, one
      `gofmt` pass over the result. Covered by two layers of Ginkgo specs:
      `internal/codegen/core`'s `Describe("safe regeneration")` (12
      specs: first generation, no-op regen, new RPC appended, RPC
      removed → orphan comment, comment not duplicated on a third run,
      constructor/struct customization untouched) and
      `internal/codegen`'s end-to-end suite, which does the same thing
      through the real CLI-facing orchestrator and asserts `go build`
      still succeeds on the assembled project — twice, once per
      generation. This is the regression test for requirement #2, not a
      manual check.
- [x] `sgo list services` rewired: reads `sgo.yaml`'s `services` list
      (populated by `sgo generate code`) and checks proto/contract-gen/
      domain/service-file presence per service.

**Exit criteria:** Editing a proto, running `generate code` twice, with a
hand-written method body added after the first run, and having that body
still present — verified in a test, not just by inspection. Met, and then
some: also verified with a proto edit that both adds and removes an RPC
in between the two runs, and with a real `go build` after each run, both
at the unit level (`internal/codegen/core`) and end to end through the
CLI-facing orchestrator (`internal/codegen`). Also verified manually
through the actual `sgo` binary (scaffold a project, generate a service,
hand-write `GetUser`, edit the proto to add a field and an RPC, `sgo
generate code` again, confirm the hand-written body and the new field/RPC
both show up and the project still builds).

**Deviations from the original plan, and why:**
- `buf`/`protoc` turned out to be avoidable entirely (Decision #5) rather
  than just picking one as primary — a strictly better outcome than the
  plan asked for, so worth calling out explicitly.
- `google.api.http` HTTP-route annotations aren't supported yet; the
  starter proto template avoids them so the pure-Go compiler doesn't need
  the `googleapis` proto files vendored in. This becomes Phase 3's
  problem — see ARCHITECTURE §12's open question on it.
- `contract/gen` ended up per-entity (`contract/gen/<name>/`) rather than
  flat, to avoid every entity's wire types colliding in one Go package
  (Decision #9).

## Phase 3 — HTTP framework adapters ✅

**Resolved the `google.api.http` question by going with the CRUD
RPC-naming convention**, not adding the `googleapis` proto dependency —
keeps Decision #5's "no external proto dependency" property intact. See
ARCHITECTURE §8.1/§12/Decision #14.

- [x] ~~`Router` interface~~ — dropped (ARCHITECTURE Decision #13); each
      framework's generated `Server` type exposes its native engine field
      directly instead, since `wire_gen.go` is already regenerated per
      current framework selection and has no need for an abstraction over
      it.
- [x] Gin adapter.
- [x] Echo adapter.
- [x] Chi adapter.
- [x] Generated `<entity>_routes_gen.go` per framework
      (`internal/codegen/httpgen`), driven by `httpgen.BuildRoutes`'
      CRUD-naming-convention route derivation (shared across all three
      frameworks; only the emitted Go code differs per framework) —
      `/api/v1/<entity>s[/:id]` paths, not a fixed guess independent of
      the proto's actual RPCs like the pre-Phase-0 implementation.
- [x] `internal/adapter/in/grpc/<entity>_grpc_server_gen.go`
      (`internal/codegen/grpcgen`) — a real gRPC server implementing
      protoc-gen-go-grpc's generated `<Entity>ServiceServer` interface,
      converting wire↔domain via Phase 2's mapper and delegating to the
      usecase port. Not originally itemized in this phase's checklist,
      but required to actually meet the exit criterion below.
- [x] A generated in-memory repository per entity
      (`internal/codegen/memgen`, ARCHITECTURE Decision #15) so a
      generated project is runnable before Phase 4's real adapters exist
      — also not originally planned, added because there was otherwise no
      way to meet the exit criterion without waiting for Phase 4.
- [x] `internal/bootstrap/wire_gen.go` (`internal/codegen/bootstrap`) —
      the composition root, regenerated on every `sgo generate code` run
      to wire *every* service on record (not just the one just
      generated): constructs each entity's in-memory repo + service,
      registers its HTTP routes and gRPC server, and starts both servers.
      Also generated (with zero entities) by `sgo init`, so a freshly
      scaffolded project already builds and runs.
- [x] `cmd/<project>/main.go` now just calls `bootstrap.Run(ctx)` — since
      that call never needs to change as services are added (wire_gen.go
      is what changes), main.go stays a one-time-generated file with
      nothing that would ever need hand-editing at this stage, rather
      than needing its own regeneration story.

**Exit criteria:** A generated project with any of the three frameworks
selected starts an HTTP server whose routes match the proto's
naming-convention-derived routes, and a gRPC server on a separate port,
both backed by the same service implementation. Verified for real, not
just by inspection: `internal/codegen`'s Ginkgo suite scaffolds a
project, generates a service, hand-implements it against the generated
in-memory repository, builds the actual binary, runs it as a subprocess,
and drives a full HTTP create/list/get/update/delete cycle against it
over real sockets — including a regression assertion that `List` returns
records, not an empty slice (a real bug in the in-memory repository's
pagination default, found and fixed while doing this manually before
automating it). Also verified manually: the same HTTP cycle via `curl`,
plus a real gRPC client hitting `CreateUser`/`GetUser`/`ListUsers` and
observing it share state with the HTTP-created records (continuing the
same id sequence) — confirming both protocols are backed by the same
service/repository instance, not two independent stacks. Gin was the
primary manual target; Echo and Chi were verified to build cleanly for a
generated project (Ginkgo covers their generated content;
`internal/codegen`'s runtime HTTP test currently exercises Gin only).

## Phase 4 — Persistence, cache, search ✅

**Framing changed slightly since Phase 3**: every entity already has a
generated in-memory repository wired as the default (ARCHITECTURE
Decision #15). This phase was about adding the *real* adapters below and
making `wire_gen.go` select one of them over the in-memory default when
`sgo.yaml` has a persistence engine chosen — not about making a project
runnable for the first time, which Phase 3 already covers.

- [x] `internal/adapter/out/persistence/<engine>/<entity>_repository_gen.go`
      self-managed adapter (`database/sql`, hand-written SQL, `pgx`/
      `go-sql-driver/mysql` drivers) for Postgres and MySQL
      (`internal/codegen/sqlgen`).
- [x] Same port, GORM-backed adapter, selectable via `sgo.yaml`
      `persistence.mode` — one shared pair of templates for both engines
      (`internal/codegen/sqlgen`), since GORM's API is dialect-agnostic;
      only the dialector import/DSN differ.
- [x] MongoDB adapter (`go.mongodb.org/mongo-driver`), same repository
      port (`internal/codegen/mongogen`).
- [x] Redis cache adapter behind `core/port/out/cache.go`
      (`internal/codegen/cachegen`).
- [x] Elasticsearch adapter behind `core/port/out/search.go`
      (`internal/codegen/searchgen`, using `esapi`).
- [x] `internal/bootstrap/wire_gen.go` selects the real adapter for
      whatever `sgo.yaml`'s **first** selected persistence engine is
      (falling back to the Phase 3 in-memory default when none was
      selected) instead of always using the in-memory one; also connects
      a cache/search client when selected, though — see Decision #19 —
      neither is auto-wired into a service's constructor.
- [x] Connection config via env vars (`POSTGRES_HOST`/`_PORT`/`_USER`/
      `_PASSWORD`/`_DB`, `MYSQL_*`, `MONGO_HOST`/`_PORT`/`_DB`,
      `REDIS_HOST`/`_PORT`, `ELASTICSEARCH_ADDR`), with generic
      placeholder defaults — not yet cross-wired to Phase 1's
      `docker-compose.yml` env values (see Open Questions).

**Not originally itemized, added because they were necessary:**
- **UUID-based string ids** (`github.com/google/uuid`), generated in the
  adapter before insert, instead of relying on each engine's own
  auto-increment/ObjectID mechanism — the one thing that let `Create`
  have an identical shape across Postgres, MySQL, and Mongo despite their
  very different native id conventions, given the domain `Id` field is
  already a plain `string` (Decision #17).
- **`AutoMigrate`** per SQL adapter (`CREATE TABLE IF NOT EXISTS` for
  self-managed, `db.AutoMigrate` for GORM), called once from
  `wire_gen.go`, so a freshly generated project works against an empty
  database without a separate migration step. Mongo doesn't need one
  (schemaless).

**Exit criteria:** A project generated with Postgres+ORM+Redis boots,
connects to both, and the generated repository adapter satisfies the
port interface (compiles against Phase 2's generated port). **Exceeded**,
not just met: verified against a real, locally running Postgres and
Redis (this development environment has both installed, no Docker
daemon available) — full CRUD through `internal/codegen/sqlgen`'s Ginkgo
suite for *both* Postgres modes, a get/set/delete round trip through
`internal/codegen/cachegen`'s suite for Redis, and — the strongest
check — an `internal/codegen` end-to-end spec that builds the actual
binary, creates a record over real HTTP, **kills and restarts the
process**, and confirms the record is still there: real persistence, not
the Phase 3 in-memory adapter's. MySQL, MongoDB, and Elasticsearch have
no local server or Docker daemon available in this environment, so
they're verified differently: MySQL/Mongo by generating a throwaway
module and running `go build`/`go run` against the real client libraries
(catches wrong signatures, wrong imports — the same class of bug the
Mongo template's `bson.D{{"_id", 1}}` templating collision turned out to
be, caught and fixed this way) rather than a live round trip;
Elasticsearch by the same compile-check alone, no live round trip
possible without a query target. All four persistence/cache combinations
(mysql+orm, mysql+self-managed, mongo+orm, with redis+elasticsearch
alongside each) were also verified to `go build` cleanly end to end
through the actual `sgo` CLI.

**Deviations from the original plan, and why:**
- `docker-compose.yml`'s env values (project-name-based
  `POSTGRES_USER`/`_PASSWORD`/`_DB` from Phase 1) aren't cross-wired to
  the adapters' own generic defaults (`postgres`/`postgres`/`postgres`).
  Running via `docker-compose` today needs the developer to either align
  the two by hand or set the adapter's env vars to match. Not resolved
  here — see Open Questions.
- `sgo.yaml` only supports **one active persistence engine per project**
  (the first in `persistence.engines`), even though the schema allows a
  list and `--db` accepts a comma-separated one. Per-entity persistence
  engine choice isn't supported — see ARCHITECTURE §12.

## Phase 5 — `sgo init` interactive wizard ✅

**Pulled forward, done immediately after Phase 1.** Originally sequenced
after Phases 2–4, but on inspection the wizard only depends on Phase 1's
`project.Scaffold`/`project.Options` — it collects exactly the fields
Phase 1's flags already collect and hands them to the same function. It
doesn't touch proto/HTTP/persistence codegen at all, so there was no real
reason to wait.

- [x] Terminal wizard (`huh`, see ARCHITECTURE §10) covering project
      name/module, HTTP framework, persistence mode, datastores, in
      `internal/wizard`. Uses `huh.ThemeCharm()` for the animated,
      focus-driven look.
- [x] Wizard calls the same scaffolding function as the Phase 1 flag-only
      path (`project.Scaffold`) — no duplicated logic. The answer-assembly
      step (`internal/wizard/answers.go`) is factored out of the form
      itself specifically so it has a Ginkgo suite without needing to
      drive a real terminal.
- [x] Non-interactive flag path keeps working for CI/scripts: passing any
      selection flag (`--module`, `--http-framework`, `--persistence-mode`,
      `--db`, `--cache`, `--search`) skips the wizard outright, as does a
      non-TTY stdin. There's no separate `--yes` flag — any one selection
      flag already signals "I want direct control," which covers the same
      case with one less flag to document.

**Exit criteria:** `sgo init` with no flags and a TTY launches the
wizard; `sgo init demo --http-framework gin ...` still works
non-interactively (also the automatic fallback when stdin isn't a TTY);
both go through `project.Scaffold`, so they produce an identical
`sgo.yaml` shape for the same choices. Verified manually (flag path) and
via `internal/wizard`'s Ginkgo suite (answer-assembly logic); the form
interaction itself isn't automated — driving a real bubbletea/huh form in
CI needs a pty harness (e.g. `x/exp/teatest`), which isn't set up yet and
wasn't worth blocking this phase on.

## Phase 6 — Web UI mode ✅

- [x] `sgo ui [--port 4747]`: localhost HTTP server (`internal/webui`),
      `go:embed` static frontend (`internal/webui/static/`: plain HTML/
      CSS/JS, no build step or npm dependency).
- [x] `/api/*` reusing `internal/codegen` + `internal/codegen/project` +
      `internal/config` directly (never shelling out to the `sgo`
      binary) — init wizard equivalent (`POST /api/init`), `generate`
      trigger (`POST /api/services`, `POST /api/services/{name}/generate`),
      `sgo.yaml` viewer/editor (`GET`/`PUT /api/config`),
      generated-vs-owned file status per service (`GET /api/state`, via
      the new shared `codegen.Status`), plus a raw proto viewer/editor
      (`GET`/`PUT /api/services/{name}/proto`) not originally itemized
      but a natural extension of the same pattern.
- [x] No auth (localhost-only binding, `127.0.0.1:<port>`); flagged as a
      later item if remote access is ever requested.

**Exit criteria — met:** `sgo ui` scaffolds a new project and runs
`generate code` on an existing one entirely from the browser (verified
live: server started, driven through a real headless-Chromium browser
via Playwright — form submission, service creation, code generation,
the sgo.yaml and proto editors all screenshotted working end to end),
calling the exact same code paths as the CLI. The "one shared
integration test" requirement is `internal/commands/webui_parity_test.go`:
it drives the real `sgo` binary through init → generate proto → generate
code on one project and `internal/webui`'s handlers through the JSON
equivalent on another (same project name, so directly comparable), then
asserts the two produce **byte-identical** generated file trees — not
just "both happened to work".

Not originally itemized, added because they were necessary: extracting
`project.BuildOptions` (shared name/config validation, previously
private to `commands/init.go`) and `codegen.Status` (shared per-service
status, previously inlined in `commands/list.go`) so the CLI and web UI
could call literally the same functions instead of two implementations
kept in sync by hand — this is also what made the parity test possible
to write honestly.

## Phase 7 — Polish

- [x] Ginkgo test coverage for the rest of `internal/codegen/*` (Phase 0
      set the pattern via `internal/config`; everything since should
      already have specs — this item is about closing gaps, not starting
      from zero). Also fixed a real flaky-test bug found along the way:
      `sqlgen_test.go`'s live-Postgres CRUD spec left a row behind
      between `DescribeTable` entries, and its cleanup step silently
      swallowed failures instead of surfacing them.
- [x] End-to-end test: `sgo init` → `sgo generate proto` → edit proto →
      `sgo generate code` → `go build` the generated project in CI, as a
      Ginkgo spec. `internal/commands/e2e_test.go` — the one spec in the
      repo that drives the actual compiled `sgo` binary as a subprocess,
      not internal/codegen's Go API directly, so flag parsing and each
      command's output are genuinely exercised.
- [x] Example project committed under `examples/` or generated in CI and
      thrown away — pick one, don't do both. Went with generated-in-CI:
      `internal/commands/e2e_test.go` (added above) already generates a
      full project from scratch and builds it on every run, so a
      committed `examples/` copy would just be a second copy that goes
      stale the moment nobody remembers to regenerate it — see
      ARCHITECTURE.md Decision #21.
- [x] Update root `README.md`, `ARCHITECTURE.md`, `docs/CLI.md` for drift
      accumulated during Phases 1–6. `docs/CLI.md` held up, no changes
      needed. `ARCHITECTURE.md` §9 (`sgo` tool-internal layout) was the
      real drift: it was still the pre-Phase-0 design sketch
      (`internal/cli`, `internal/registry`, a nested
      `codegen/persistence/{postgres,mysql,mongo}` tree) rather than the
      layout that actually landed — rewritten to match
      `internal/commands`, `internal/config`, and one flat package per
      generator (`sqlgen`, `mongogen`, `cachegen`, `searchgen`, ...).
      README.md gained a short Development section (the live-Postgres/
      Redis test assumptions weren't documented anywhere) and a mention
      of the new CLI-subprocess e2e suite.

## Phase 8 — OpenAPI documentation ✅

All Phases 0–7 are done; this is new scope, not part of the original 8
requirements. Full design in ARCHITECTURE.md §13. Decided via
`AskUserQuestion` before any code: OpenAPI stays a generated **output**
of the existing proto-first pipeline (proto is still the only source of
truth, exactly like `contract/gen`), not a second, competing input —
and real spec-compliance checking uses an actual JSON Schema validator
against the official vendored meta-schemas, not a hand-rolled subset
validator.

- [x] **Definition** — `internal/codegen/openapigen`'s `Document`,
      `Info`, `PathItem`, `Operation`, `Parameter`, `RequestBody`,
      `Response`, `Schema`, `Components` types: one shared Go model for
      both OpenAPI 3.0.x and 3.1.x. Turned out simpler than planned: a
      version-aware `nullable`/`type:[X,"null"]` split was never needed
      because proto3's `Kind` (`internal/codegen/proto`) has no
      "optional"/nullable concept at all yet — every field sgo can
      generate is required, so the *same* `Schema` value validates
      under either version's meta-schema, and `Encode` only stamps the
      top-level `openapi:` version string. Documented in `types.go`'s
      doc comment as the reason, not silently simplified.
- [x] **Generator** — walks every service in `sgo.yaml`'s `services`
      list, recompiling each one's proto and calling
      `httpgen.BuildRoutes` — confirmed load-bearing by
      `openapigen/generate_test.go`, which cross-checks a generated
      `docs/openapi.yaml`'s paths against the *actual* generated Gin
      route file's registrations (stronger evidence than comparing
      against `BuildRoutes`'s own output a second time, which the
      generator already calls internally). Also fails loudly on two
      real hazards aggregating multiple services into one flat
      `components.schemas`/`paths` namespace creates: a same-named
      message defined differently by two services, and two RPCs
      deriving the same verb+path (which `gin.Engine` would panic on
      at registration time anyway) — neither was originally itemized,
      both found while implementing and covered by their own specs.
      `sgo generate openapi [--version 3.0|3.1] [--format yaml|json]`.
- [x] **Checker** — real JSON Schema validation via
      `github.com/santhosh-tekuri/jsonschema/v5` against the vendored
      meta-schemas, auto-detected version. Runs automatically after
      `generate openapi` writes a file, and standalone via `sgo openapi
      validate [path]`.
- [x] New direct dependency in sgo's own `go.mod`:
      `github.com/santhosh-tekuri/jsonschema/v5`.
- [x] `sgo.yaml` gained `openapi: {version, format}`, persisted at
      `sgo init` via `--openapi-version`/`--openapi-format` (defaults
      `3.0`/`yaml`), also added to the interactive wizard (a new
      select group) and the web UI's `POST /api/init` — not just the
      flag path, so all three project-creation surfaces agree.
- [x] Ginkgo specs: 29 in `openapigen` (~89% coverage) covering content
      per version × format, a deliberately-broken document failing the
      checker with a real schema-validation error, the route-parity
      check described above, the two collision-detection error paths,
      and every scalar `Kind` mapping to a valid schema. Plus 7 CLI
      subprocess specs (`internal/commands/openapi_e2e_test.go`)
      driving the real binary, matching Phase 7's e2e-test standard.
- [ ] `sgo ui` follow-up — still not done, still not required for this
      phase; left for whenever it's actually asked for.

**Exit criteria — met:** `sgo generate openapi` on a project with a
generated service produces a `docs/openapi.yaml`/`.json` that passes
`sgo openapi validate`, for both 3.0 and 3.1, both formats — verified
directly (see above) and cross-checked with a genuinely independent
third party never touching sgo's own vendored schemas or validation
code: Python's `openapi-spec-validator` accepted the generated output
for both version/format combinations by hand, in addition to the
automated suite.

## Phase 11 — Wizard TUI polish **(planned)**

Small, self-contained, ships independently of Phases 12–16. Full design
in ARCHITECTURE.md §14.

Diagnosed by capturing `sgo init`'s wizard under a real pty and
inspecting the rendered frames rather than guessing: `huh`'s `Confirm`
field centers its Yes/No buttons under that field's own title text.
"Enable Redis cache?" (19 chars) and "Enable Elasticsearch search?" (29
chars) sit in the same group, so their buttons land at different
horizontal offsets — the buttons visibly jump left-right between the
two fields instead of lining up, which reads as "not tidy"/"position
not proper".

- [ ] Give every `huh.NewConfirm()` in the wizard the same fixed
      `WithWidth` (or the group's own configured width) so all buttons
      in a group anchor to the same column regardless of each
      question's title length, instead of each one centering
      independently.
- [ ] Capture the wizard under a real pty again after the fix (same
      method used to diagnose it) and confirm the button columns now
      line up — a repeatable check, not just "looks right once".
- [ ] A `wizard` Ginkgo spec asserting the rendered frame's button
      column position is identical across every `Confirm` field in the
      same group (regression guard, since this is exactly the kind of
      thing that's easy to silently reintroduce by adding a new field
      with a different-length title later).

**Exit criteria:** every `Confirm` field's Yes/No buttons in the same
wizard group render at the same horizontal position, verified against a
real pty capture, with a spec catching a regression.

## Phase 12 — Configurable, simplified project layout **(planned)**

Full design in ARCHITECTURE.md §15. Decided via `AskUserQuestion` before
any code: layout customization is a `layout:` section in `sgo.yaml`
itself (a small DSL, not a separate config file), and the *default*
layout changes too, not just gains an escape hatch — the current
default (`internal/core/{domain,port/{in,out},service}`,
`internal/adapter/{in/{http,grpc},out/{persistence,cache,search},mapper}`)
nests up to 6 directories deep for a single generated file, which is
the concrete thing "hard to understand" names.

- [ ] **Simplified default layout** — drop the `core`/`adapter` wrapper
      directories and the `in`/`out` sub-grouping entirely; everything
      that's currently under `internal/core/*` or `internal/adapter/*`
      moves one level up, directly under `internal/`:
      ```
      internal/
      ├── domain/<entity>/
      ├── port/                  # usecase + repository + cache + search, all here
      ├── service/
      ├── http/<framework>/
      ├── grpc/
      ├── persistence/<engine>/
      ├── cache/<engine>/
      ├── search/<engine>/
      ├── mapper/
      └── bootstrap/
      ```
      The driving/driven (`in`/`out`) distinction was mostly serving the
      directory tree, not the reader — `<Entity>Usecase` vs.
      `<Entity>Repository` already say which is which by name once
      they're just files in `internal/port/`. Cuts the deepest path
      from 6 directories to 4 (`internal/persistence/postgres/...`).
- [ ] **`internal/codegen/layout`** — one small package every other
      `internal/codegen/*gen` package asks "where does this file go?"
      instead of hardcoding `filepath.Join(...)` itself. A `layout.Slot`
      enum (`Domain`, `Port`, `Service`, `HTTP`, `GRPC`, `Persistence`,
      `Cache`, `Search`, `Mapper`, `Bootstrap`, `Cmd`) resolves to a path
      template (`{entity}`, `{framework}`, `{engine}`, `{project}`
      placeholders), defaulting to the simplified layout above; a
      `layout:` section in `sgo.yaml` overrides any subset of slots,
      the rest keep their default.
      ```yaml
      layout:
        persistence: internal/store/{engine}   # only this one customized
      ```
- [ ] Prototype the resolver against **one** generator package first
      (`internal/codegen/core`, the smallest) before rolling it out to
      the other ~9 — same risk-reduction Phase 2 used for safe
      regeneration ("a small throwaway spike... in case it's messier
      than expected"). Confirm it doesn't break §6's safe-regeneration
      file-finding before touching `httpgen`, `sqlgen`, `mongogen`,
      `cachegen`, `searchgen`, `grpcgen`, `wiregen`, the mapper
      generator, or `project.Scaffold`.
- [ ] `layout:` is set at `sgo init` time, same as HTTP framework or
      persistence engine already are. **Known limitation, documented
      rather than silently unsupported:** changing `layout:` after a
      project has already been generated isn't supported in v1 — safe
      regeneration (§6) finds owned files by their *current* expected
      path, so moving that path out from under already-generated files
      needs a real migration step this phase doesn't build. Logged as a
      Non-goal below.
- [ ] The interactive wizard and web UI aren't extended with a
      layout-editing UI in this phase — `sgo.yaml` is still a plain text
      file either surface can point someone at; a dedicated UI for it
      is a follow-up, not required to ship the underlying capability.
- [ ] Update every path mentioned in ARCHITECTURE.md §3/§8, docs/CLI.md,
      and the e2e specs' own path assertions to the new default.
- [ ] Ginkgo specs: `internal/codegen/layout`'s resolver (default
      resolution per slot, override-one-keep-rest, unknown slot key
      rejected with a clear error), plus one full generate-code run
      against a project with a custom `layout:` override, asserting the
      overridden slot's files land at the custom path and every other
      slot still lands at its default.

**Exit criteria:** a freshly generated project's default layout is
`internal/{domain,port,service,http,grpc,persistence,cache,search,
mapper,bootstrap}` (no `core`/`adapter` wrapper, no `in`/`out`); a
project with a `layout:` override in `sgo.yaml` gets exactly the
customized slots at their custom paths, everything else unchanged, and
still passes the full existing e2e suite (init → generate proto → hand
edit → generate code → `go build`) at the new default paths.

## Phase 13 — `sgo list endpoints` **(planned)**

Small, independent of Phases 12/14/15 — ships fast, gets more useful
once Phase 15 lands proto-defined paths but doesn't depend on it. Full
design in ARCHITECTURE.md §16.

- [ ] `sgo list endpoints [<service>]` — prints every HTTP route
      currently derived for the project's registered services (all of
      them with no argument, one with it): method, path, and which
      RPC/service it comes from. Reuses
      `internal/codegen/httpgen.BuildRoutes` directly — the exact same
      route list `sgo generate openapi` already turns into
      `docs/openapi.<ext>` and the generated `*_routes_gen.go` actually
      registers, not a second derivation.
- [ ] Ginkgo specs (content per framework selection) plus a CLI e2e spec
      asserting the printed table matches the generated route file's own
      registrations, same cross-check `openapigen/generate_test.go`
      already does for the OpenAPI document.

**Exit criteria:** `sgo list endpoints` on a project with generated
services prints exactly the routes the generated HTTP adapter actually
registers — verified against the generated route file, not just
`BuildRoutes`'s own output a second time.

## Phase 14 — OpenAPI discoverability + a live Swagger/Redoc UI **(planned)**

Full design in ARCHITECTURE.md §17. `sgo generate openapi` and `sgo
openapi validate` already exist (Phase 8) — this phase covers both
halves of what was actually missing: nothing in the terminal points you
at them, and there's no way to *browse* the generated document short of
opening the raw YAML/JSON.

- [ ] **Discoverability** — `sgo generate code` prints a "Next steps"
      hint mentioning `sgo generate openapi` once a service exists, the
      same way `sgo init` already prints one for `sgo generate proto`;
      README's status line and quick start call it out explicitly
      (partly done already, finish the rest).
- [ ] **`sgo openapi ui [--port 4749]`** — serves the project's
      generated `docs/openapi.<ext>` through an embedded interactive API
      doc viewer at `http://127.0.0.1:<port>` (same `127.0.0.1`-only, no
      auth stance as `sgo ui`/`sgo run --debug`, §11/§15). Reads
      whatever's already on disk — same "run `sgo generate openapi`
      first if it doesn't exist yet" UX `sgo openapi validate` already
      has, not auto-regenerating on every request.
- [ ] Vendor a specific pinned version of a standalone doc-viewer bundle
      (evaluating Redoc's single-file standalone build against
      `swagger-ui-dist`'s multi-file one — whichever is smaller wins,
      same self-contained-by-default stance as the vendored OpenAPI
      meta-schemas in Phase 8) via `go:embed`, not a CDN `<script>` tag —
      works offline, consistent with every other "no external dependency
      at runtime" choice this project has made (pure-Go proto compiler,
      pre-installed browser in dev, vendored meta-schemas).
- [ ] Ginkgo specs (`httptest`, same pattern as `internal/webui`/
      `internal/dashboard`) plus a real-browser Playwright pass
      confirming the page actually renders the project's real paths —
      same standard `sgo ui` and the Phase 10 dashboard were held to.

**Exit criteria:** `sgo generate code` visibly points at `sgo generate
openapi`; `sgo openapi ui` on a project with a generated doc serves a
real interactive viewer showing that project's actual endpoints,
verified in a real browser, no internet access required.

## Phase 15 — Proto-defined HTTP paths (`google.api.http`) + root path **(planned)**

The riskiest and most novel piece of this batch — resolves the
`google.api.http` open question ARCHITECTURE.md §12/Decision #14 already
flagged as "a possible future upgrade if the convention-based routing
proves too rigid," rather than reversing that decision outright. Full
design in ARCHITECTURE.md §18. Decided via `AskUserQuestion` before any
code: **annotations are an optional override, not a replacement** — an
RPC with no `google.api.http` option keeps today's naming-convention
routing (`Create*`→`POST`, etc., §8.1); one with the option uses it
instead. Every existing generated project keeps working unchanged.

- [ ] **Vendor `google/api/http.proto` + `google/api/annotations.proto`**
      (small, Apache-2.0, just the extension/option definitions — not
      the whole `googleapis` tree) so `sgo generate proto`-authored files
      can `import "google/api/annotations.proto";` and the pure-Go
      compiler (`bufbuild/protocompile`, Decision #5) can resolve that
      import from the vendored copy without a `protoc`/`buf` binary or
      network access — preserves the property Decision #5 already
      established, doesn't reopen it.
- [ ] **`sgo`'s own `base_path` service option** — `google.api.http` has
      no concept of a service-level path prefix, only per-method rules,
      so a root path needs sgo's own small option:
      `option (sgo.base_path) = "/v1";` on the service (a proto
      extension in the 50000–99999 organization-reserved range). Applies
      to every route on that service, annotation-derived or
      convention-derived — replacing today's hardcoded `/api/v1` prefix
      (§8.1) with this as the default when unset, so nothing changes for
      a proto that doesn't opt in.
- [ ] **`internal/codegen/httpgen.BuildRoutes`** — for each RPC, check
      for a `google.api.http` option first (method/path from
      `get`/`post`/`put`/`delete`/`patch`, path parameters from `{name}`
      bindings, request body from the `body` field); fall back to
      today's naming-convention derivation when absent. Path parameters
      translate to each framework's own syntax (Gin/Echo `:name`, Chi
      `{name}`) same as the existing `{id}` handling already does — not
      limited to a field literally named `id` any more, resolving the
      other open question §12 already flagged alongside this one.
- [ ] `sgo generate openapi` and `sgo list endpoints` (Phase 13) need no
      changes at all — both already just consume `BuildRoutes`'s output,
      so proto-defined paths show up in generated OpenAPI docs and
      `sgo list endpoints` for free once `BuildRoutes` itself understands
      them.
- [ ] Ginkgo specs: a proto with an explicit `google.api.http` option
      generates the annotated route, not the convention-derived one; a
      proto without one is byte-identical to today's output (regression
      guard that the override really is optional); path-parameter
      translation per framework; the `base_path` override; plus a CLI
      e2e spec building a real project from an annotated proto and
      hitting the actual generated route over a real socket, same
      standard the HTTP CRUD e2e suite (Phase 3/4) already holds
      generated adapters to.

**Exit criteria:** an RPC with a `google.api.http` option gets exactly
that route; one without keeps today's convention-derived route,
unchanged; a service with `(sgo.base_path)` set gets that prefix instead
of the default `/api/v1`; every existing generated-project e2e spec
still passes with no proto changes.

## Phase 16 — Docs: README as a getting-started guide, CONTRIBUTING.md **(planned)**

Deliberately last in this batch — it should describe the layout,
commands, and proto conventions Phases 11–15 actually land with, not
what's true today. Mechanical relative to the rest of this batch.

- [ ] **README.md** restructured into a fuller guide: overview, install,
      getting started/quick start, full command reference pointer
      (`docs/CLI.md`), architecture pointer (`ARCHITECTURE.md`), FAQ/
      troubleshooting section if anything recurring surfaced by then.
      Still the repo's actual `README.md` (GitHub renders it on the repo
      homepage) — "as a wiki" means comprehensive and navigable, not a
      literal GitHub Wiki, which is a separate, harder-to-review,
      harder-to-PR surface than a file already in the repo.
- [ ] **`CONTRIBUTING.md`** — the existing "Contributing / development"
      section moved out of README.md into its own file (standard
      GitHub convention: `CONTRIBUTING.md`, not `CONTRIBUTE.md` — GitHub
      links to it automatically from the "Contributing" prompt on a new
      issue/PR when it's named this way), expanded with the phase-based
      workflow this project actually uses (plan → `AskUserQuestion` →
      PLAN.md/ARCHITECTURE.md → implement → PR, not merged by the author)
      so an outside contributor understands the process before opening
      one.
- [ ] Cross-check every command/path mentioned in both files against
      `docs/CLI.md` and the current generated layout — this phase is the
      one place drift between "what the docs say" and "what `sgo`
      actually does" gets caught for this whole batch.

**Exit criteria:** README.md covers overview → install → getting started
end to end without needing `ARCHITECTURE.md`/`docs/CLI.md` open
side-by-side for a first-time user; `CONTRIBUTING.md` exists and is
what GitHub links to from a new issue/PR; nothing in either file
contradicts `docs/CLI.md` or the actual current layout.

## Non-goals (for now)

- Multi-service monorepo orchestration beyond one `sgo.yaml` per repo.
- Auth/remote access for `sgo ui`.
- Auto-migrating hand-written code when a proto field/method is removed
  (the compiler is the safety net, per ARCHITECTURE §6.4).
- Bundling/vendoring `buf` itself — documented prerequisite for now.
- OpenAPI as an input (scaffolding routes/code from a hand-written
  spec) — considered for Phase 8 and explicitly declined; proto stays
  the only source of truth. Revisit only if requested.
- Diffing a generated project's actual routes against an externally
  supplied OpenAPI spec (contract testing) — the Phase 8 checker
  validates a document's own well-formedness, not code-vs-spec
  consistency. Revisit only if requested.
- Changing `layout:` in `sgo.yaml` after a project has already been
  generated — Phase 12 picks the layout at `sgo init` time only; safe
  regeneration (§6) finds owned files at their *current* expected path,
  so retargeting that path for an already-generated project needs a real
  migration step (move files, update package paths/imports) this phase
  doesn't build. Revisit if it turns out people want to change layout
  mid-project rather than only choosing it up front.
- A layout-editing UI in the wizard or `sgo ui` — `sgo.yaml`'s `layout:`
  section is hand-edited YAML in v1, same as any other manifest field
  before it got wizard/web-UI treatment. Revisit once the underlying
  capability (Phase 12) has seen real use.
- A generic templating/plugin system for arbitrary custom generators —
  Phase 12's `layout:` only relocates *where* sgo's own fixed set of
  generated files land, it doesn't let someone add a wholly new kind of
  generated file. Nobody's asked for that yet.

## Sequencing notes

- Phase 2 (safe regeneration) is the riskiest and most novel piece of the
  whole plan — it's the direct answer to requirement #2. Worth doing a
  small throwaway spike before committing to the `go/parser` approach in
  ARCHITECTURE §6, in case it's messier than expected against real Go
  source.
- Phases 3 and 4 are independent of each other and could be reordered or
  parallelized; they're sequenced 3-then-4 here only because HTTP is the
  more visible payoff.
- ~~Phase 5 and 6 both depend on Phases 1–4 being done, not on each
  other~~ — this turned out to be wrong for Phase 5: the wizard only
  needs `project.Scaffold` from Phase 1, so it was pulled forward and
  landed right after Phase 1 (see Phase 5 above). Phase 6 (web UI) is a
  different story — once `sgo generate`/adapters exist, a dashboard
  showing their status is more useful, so it stays put unless the web UI
  becomes the priority over Phases 2–4 for some other reason.
- Phase 11 (wizard polish) ships first in this batch — small, and
  entirely independent of 12–16.
- Phase 12 (layout) is the riskiest and most novel piece of this batch —
  it touches every generator package that writes a file — so it's
  sequenced right after Phase 11 and before anything else in the batch:
  Phases 13–15 all consume paths Phase 12 changes, and there's no reason
  to build on top of a layout that might still move.
- Phases 13 and 14 are independent of each other and of Phase 15;
  sequenced 13-then-14 only because `sgo list endpoints` is the smaller
  of the two.
- Phase 15 (proto-defined HTTP paths) depends on nothing in this batch
  functionally, but is sequenced after 12–14 anyway: it's the largest,
  most novel piece here (new proto vendoring, a new custom option, per-
  framework path-parameter translation), and Phases 13/14 are more
  useful to have landed first since Phase 15 makes both of them richer
  for free (proto-defined routes show up in `sgo list endpoints` and
  generated OpenAPI docs without either needing changes) rather than the
  reverse.
- Phase 16 (docs) is deliberately last — it documents the layout,
  commands, and proto conventions Phases 11–15 land with, not a snapshot
  from partway through.
