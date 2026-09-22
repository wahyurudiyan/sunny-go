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

## Phase 9 — Loading indicators ✅

Small, self-contained, and unrelated to Phase 10 other than sharing a
terminal — sequenced first so it ships fast rather than waiting on the
much bigger `sgo run` work. Full design in ARCHITECTURE.md §14.

- [x] `internal/progress`: a minimal terminal spinner (own
      goroutine + ticker printing `\r<frame> <message>`, cleared on
      completion) wrapping a blocking call — no new dependency; the
      existing `charmbracelet/huh`/`bubbletea` stack is a heavier fit
      for a single inline spinner than it's worth pulling into every
      command.
    - TTY-aware like the wizard already is (`isatty`): animates only
      when stdout is a real terminal; otherwise prints a single
      "<Verb>…" line so piped/CI/script output stays clean — the same
      reasoning `sgo init`'s wizard-vs-flags branch already uses
      (ARCHITECTURE.md §10).
- [x] Wired into `sgo init` (scaffolding), `sgo generate proto` (stub
      write), `sgo generate code`, and `sgo generate openapi`.
      **Revised from the plan above**: one spinner per whole command,
      not one per pipeline stage inside `generate code`. Per-stage
      spinners would mean threading a progress-reporting callback
      through `internal/codegen`'s generator functions, which are
      deliberately UI-agnostic (the same functions the web UI's
      handlers call directly, ARCHITECTURE.md Decision #8) — not worth
      it for a command that already completes in well under a second
      against real generated projects.
- [x] Ginkgo specs for `internal/progress` itself (5 specs, ~96%
      coverage: non-TTY plain-line output, TTY animation and
      line-clearing, error propagation on both paths). Existing CLI e2e
      specs needed no changes — `runSgo`'s piped, non-TTY stdin/stdout
      already exercises the plain-line path, and its output assertions
      were already substring checks that still hold.

**Exit criteria — met:** every generation command shows visible
progress on a real terminal (verified with a real pty via `script
-qc`), and piped/non-TTY output is unchanged from before this phase.

## Phase 10 — `sgo run` and the `--debug` config dashboard ✅

Full design in ARCHITECTURE.md §15. Decided via `AskUserQuestion` before
any code, same as Phase 8:

- **v1 concretely implements one config source: `.env`.** Everything
  else (a remote config repo, KMS/Vault-style secret managers) is a
  defined, real Go interface (`envsource.Source`) with no concrete
  adapter shipped yet — same shape sgo already uses for persistence/
  cache/search engines (one interface, adapters added over time), not a
  redesign later. No cloud SDK dependency added in this phase.
- **"repo" as a config source means a separate, remote config
  repository** (e.g. an org's central config repo), not a file already
  sitting in the project — noted now so the eventual adapter's contract
  is unambiguous, even though it isn't built yet.
- **The `--debug` dashboard is fully read/write, including write-back to
  whatever source a value came from** (the interface design reflects
  this even though only `.env` write-back is real today) — not a
  read-only viewer.
- **The dashboard is a new view under `sgo run --debug`**, not folded
  into `sgo ui` — a runtime/live-process concern, separate from `sgo
  ui`'s project-scaffolding one.

- [x] **`envsource.Source` interface** (`Name`, `Fetch`, `Write`) plus
      the one real implementation, `DotEnvSource` (parses/writes
      `.env`, preserving line order; a `Write` on a source that can't
      support it — a hypothetical future read-only adapter — returns a
      clear error, never a silent no-op). Also `envsource.Merge`, the
      shared "source values, then the real process environment last"
      precedence helper both `sgo run`'s own startup and the dashboard's
      restart path call, rather than two implementations of the same
      rule.
- [x] **`sgo run [--debug] [--debug-port 4748]`** — finds the project's
      one `cmd/<name>/` (there's only ever one), merges `.env` values
      under real OS environment variables (OS env wins — the same
      convention most `.env` tooling uses, so a real prod/CI env var
      already set is never silently shadowed by a leftover local
      `.env`), and runs it via `go run ./cmd/<name>`, matching "runs
      like go run" — no separate build-and-run-binary step to manage.
      Stops the child cleanly on Ctrl-C either way; non-debug mode races
      a blocking wait for the child against the interrupt so a real
      crash is still reported.
- [x] **Process supervisor** (`internal/run`): owns the child `go run`
      process; `Stop` kills its whole process group (`Setpgid` + a
      negative-PID `SIGKILL`), not just the `go` toolchain process, so
      `go run`'s spawned compiled binary is never orphaned. `--debug`
      mode kills and respawns it with a new merged environment when the
      dashboard writes a change, instead of requiring a manual
      Ctrl-C/rerun. Scoped to config-triggered restarts only — not a
      general file-watching auto-reload tool (nobody asked for that;
      revisit only if requested).
- [x] **The dashboard** (`internal/dashboard`, same shape as
      `internal/webui`): `127.0.0.1`-only, no auth (same stance as `sgo
      ui`, ARCHITECTURE.md §11). Lists every key any configured source
      declares, its source, and whether it's writable; editing a
      writable value calls `Source.Write` then triggers a supervised
      restart. Real-time updates over Server-Sent Events (stdlib
      `net/http`, no new dependency) rather than polling or a WebSocket
      library — a no-payload "refresh" signal tells connected browsers
      to refetch, nothing is ever pushed.
    - **Secret values are masked by default** (a reveal toggle per
      value, not shown-by-default) — a default chosen rather than asked
      about directly: a debug dashboard showing raw secrets in a browser
      tab by default is a real shoulder-surfing/screen-share risk for
      something meant to run during normal day-to-day development.
      **Revised from the plan above**: rather than a per-key `Secret
      bool` hint on `envsource.Value` (heuristic, and risks a false
      negative missing a real secret), every value is masked by
      default and revealed only on request — simpler, and safe by
      construction rather than by guessing which keys are secret.
      `GET /api/config` never includes a value at all; only
      `GET /api/config/{key}` does, called only when a client clicks
      Reveal.
- [x] Ginkgo specs: `envsource` (parse/write/precedence, ~89%
      coverage), the supervisor's start/restart/stop lifecycle against a
      tiny fixture program (~92% coverage, confirms no process — `go
      run` or its spawned binary — survives `Stop`), the dashboard API
      via `httptest` plus a real listener for the SSE spec (~80%
      coverage), and a real CLI e2e spec — builds the actual `sgo`
      binary, runs it against a minimal fixture project, hits the
      dashboard API, edits a value, and confirms the child process's own
      stdout shows the new value after restart. Plus a real-browser pass
      (Playwright against the pre-installed headless Chromium) for the
      dashboard UI itself, the same standard `sgo ui` was held to:
      confirmed a secret value never appears in the `/api/config` list
      response or the DOM before Reveal is clicked, and that editing
      through the real form saves and restarts.

**Exit criteria — met:** `sgo run` on a generated project starts it
with `.env` values loaded (OS env still wins); `sgo run --debug`
additionally serves a dashboard where editing a `.env`-sourced value
restarts the service with the new value in effect, verified against the
real running child process (its own stdout), not just the dashboard's
own state.

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

## Phase 12 — Replace the generated architecture with DDD tactical patterns **(mostly done — shared kernel and GORM value-object embedding deferred, see Exit criteria)**

**By far the largest, riskiest phase in this plan — larger than
everything else in this batch combined.** Redefined from an earlier,
much smaller "flatten the current layout, add a `layout:` override"
version of this phase, per an explicit follow-up decision
(`AskUserQuestion`): this isn't a directory rename any more. sgo's
generated architecture moves from today's flat CRUD-hexagonal model
(one struct per proto message, a fixed 5-method repository, no domain
events, no invariant enforcement) to real DDD tactical patterns —
aggregates that enforce invariants, value objects, domain events,
CQRS-separated application services — **replacing the current default
outright**, not as a selectable alternative alongside it. Full design
in ARCHITECTURE.md §17.

**Read this note if you only read one thing in this phase:** proto3
has no native way to say "this message is an aggregate root with these
invariants" — plain RPCs and messages don't carry that meaning by
themselves. Getting from "proto file" to "real enforced business
rules" needs *something* sgo can mechanically act on beyond today's
message/field types. The concrete answer below is real, standards-based
proto extensions (the same "vendor a real spec, don't invent one"
principle already used for `google.api.http` and OpenAPI's meta-schemas)
rather than a second hand-rolled modeling file — proto stays the only
source of truth (§2/Decision #8's principle, unbroken). That's a
specific technical proposal, not something the earlier `AskUserQuestion`
round drilled into — flagged here explicitly for you to sanity-check
before implementation starts, same as every other phase's design gets
reviewed via this PR before code.

Decided via `AskUserQuestion`: **full semantic upgrade**, not a cosmetic
rename (real invariant enforcement, real event collection); **replaces
the default outright**, no permanent dual-style selector; **event bus
(Kafka/RabbitMQ/NATS) is explicitly deferred** — this phase builds the
seam (an `EventPublisher` interface + a real no-op default), not a
broker adapter, since a real pluggable message-bus system is its own
undertaking and you mentioned having your own idea for that shape to
bring later.

### Vendored proto options + validation (the foundation everything else needs)

- [x] **`sgo/options.proto`** (vendored, `go:embed`, pure-Go compiler
      resolves it — no `protoc`/`buf` binary, same property Decision #5
      already established): `MessageOptions` extensions
      `(sgo.aggregate_root)`, `(sgo.value_object)`, `(sgo.domain_event)`;
      `MethodOptions` extensions `(sgo.command)`/`(sgo.query)` (explicit
      CQRS override when the naming convention below doesn't fit). This
      file is now the *origin* of sgo's custom-options mechanism —
      Phases 15/16 (`(sgo.base_path)`, `(sgo.repository_query)`,
      `(sgo.hide_route)`) add fields to this same file rather than each
      vendoring their own, and are resequenced after this phase so they
      can (see Sequencing notes).
- [x] **Vendor `buf/validate/validate.proto`** (protovalidate — the
      modern, widely-adopted successor to `protoc-gen-validate`; CEL-
      expression field/message constraints, e.g.
      `option (buf.validate.field).string.min_len = 1;`) the same way,
      plus a new direct dependency in a *generated project's* `go.mod`:
      `github.com/bufbuild/protovalidate-go`. Real invariant enforcement
      without sgo hand-rolling CEL evaluation itself — a generated
      `Validate() error` method calls the real library against the
      compiled descriptor's constraints.

### Domain layer (`internal/domain/`)

- [x] **Role inference**: within one proto file, the message marked
      `(sgo.aggregate_root)` (or matching the entity name by the
      existing convention, if none is explicitly marked) is the
      Aggregate Root; a message marked `(sgo.value_object)` generates an
      immutable Go type (no setters, constructor-validated); a message
      marked `(sgo.domain_event)` generates a plain event struct in
      `events.go`; anything else referenced by the aggregate becomes a
      child Entity within it.
- [x] **Generated sibling** (`<context>/<context>_gen.go`, same
      generated/owned split principle as today, Decision #3): the flat
      field struct (as today), plus new scaffolding — an internal
      `events []DomainEvent` slice, a `PullEvents() []DomainEvent`
      method, and a `Validate() error` method wired to protovalidate
      against whatever `buf.validate` constraints the message declares.
- [x] **Owned `aggregate.go`**, same contract as today's owned
      `<entity>.go` (created once, never overwritten): this is where
      real business methods and invariant-triggered event appends
      actually get hand-written — sgo scaffolds the *mechanism*
      (validation, event collection), not arbitrary business rules it
      has no way to know.
- [x] **`domain/<context>/repository.go`** — same port role as today's
      repository port (and still open to Phase 16's
      `(sgo.repository_query)` extension), now typed against the
      aggregate root instead of a flat struct.
- [x] **`domain/<context>/errors.go`** — sentinel errors, same as
      today's port-level errors, relocated/renamed to match.
- [ ] **Shared kernel** (`domain/shared/`): a value-object message
      declared under `contract/pb/shared/*.proto` and `import`-ed by
      multiple entity protos (e.g. `shared.Money` used by both `Order`
      and `Invoice`) generates once into `internal/domain/shared/`,
      not duplicated per context — the proto resolver already handles
      cross-file imports (it resolves `google/protobuf/*` and, after
      Phase 15, `google/api/*`); this extends the same resolution to a
      project's own `contract/pb/shared/` imports. Idempotency matters
      here specifically: regenerating two different entities that both
      reference `shared.Money` must produce identical output, not a
      conflict.

### Application layer (`internal/application/`) — CQRS

- [x] **`application/<context>/{command,query}.go`** — an RPC's request
      message becomes a `<Rpc>Command` or `<Rpc>Query` DTO, classified
      by the same naming convention HTTP routing already uses
      (`Create`/`Update`/`Delete` → command, `Get`/`List` → query),
      overridable per-RPC via `(sgo.command)`/`(sgo.query)` for anything
      that doesn't fit — the same "convention with an explicit override"
      shape Phase 15/16's options already use, kept consistent rather
      than inventing a third pattern.
- [x] **`application/<context>/service.go`** — replaces today's
      `<entity>_service.go`; same owned-file, stub-appended-per-new-
      command/query contract (Decision #12), but now actually
      orchestrates: load the aggregate via the repository → call its
      mutating method (which validates + collects events internally,
      per the domain-layer scaffolding above) → save via the repository
      → pull events and hand them to the `EventPublisher` seam. A real
      behavioral upgrade over today's service, which calls the
      repository directly with no aggregate/invariant/event step at
      all.
- [x] **`application/ports/event_publisher.go`** — the seam the future
      messaging phase plugs into:
      ```go
      type EventPublisher interface {
          Publish(ctx context.Context, events ...domain.Event) error
      }
      ```
      Generated with one real, working default: a no-op implementation
      wired into `wire_gen.go`. No `infrastructure/messaging/` package
      in v1 — an empty placeholder directory with nothing generated
      into it isn't something sgo does anywhere else, so it isn't
      introduced here either; the interface alone is the extension
      point.

### Infrastructure layer (`internal/infrastructure/`)

- [x] **`infrastructure/persistence/<engine>/`** — same generated
      repository adapters as today (Postgres/MySQL/MongoDB/memory),
      relocated, now implementing the aggregate-aware repository
      interface. For SQL/GORM mode, a value object (e.g. `Money`) maps
      via GORM's native embedded-struct support (`gorm:"embedded"`),
      not a new mapping mechanism.
- [x] **`infrastructure/transport/{http,grpc}/`** — same HTTP/gRPC
      adapters as today, relocated; still map wire proto messages to
      Command/Query DTOs and call the application service, same shape
      as before under new names.
- [x] **`infrastructure/bootstrap/wire_gen.go`** — same composition-root
      role as today's `internal/bootstrap/wire_gen.go`, relocated under
      `infrastructure/` to fit the new three-layer top level; now also
      wires the no-op `EventPublisher`.
- [x] **No `infrastructure/clients/`** in v1 — the reference structure's
      outbound-API-client convention (its `Stripe` example) has nothing
      concrete behind it in sgo today; an empty placeholder directory
      isn't generated for the same reason `messaging/` isn't.
- [x] **`internal/adapter/mapper/`** → relocated, but to
      `infrastructure/transport/<entity>_mapper_gen.go` (one shared
      package, entity-prefixed files — the pre-Phase-12 `mapper/`
      package's own convention), **not** to a `mapper.go` per
      persistence engine as originally planned here. The old mapper's
      only real caller was the gRPC adapter's wire↔domain conversion,
      never persistence — sqlgen/mongogen/memgen already map straight
      between the domain aggregate and their own engine-specific
      row/document model inline in their own templates, no shared
      mapper package involved on the persistence side at all. Placing
      it under `persistence/<engine>/` would have been placing it next
      to the one adapter family that doesn't use it. `GenerateInfraMapper`
      also grew a wire↔application direction (`<Msg>ToApp`, for the
      gRPC adapter's own request-side mapping), which this plan didn't
      anticipate — the HTTP adapter needs no such conversion, since it
      binds JSON straight into the application DTO.

### Everything downstream

- [x] `cmd/<name>/main.go` unchanged in role (owned, thin, calls
      `infrastructure/bootstrap`) — the reference structure's own
      comment agrees ("Wire dependencies & boot adapters").
- [x] Updated docs/CLI.md, README.md, and every e2e spec's path/content
      assertion across `internal/codegen` and `internal/commands`
      (including the real running-server HTTP test and the real
      compiled-binary CLI e2e test) to the new layout — all green.
      ARCHITECTURE.md §3 (the directory tree), §2 (the "hexagonal core"
      overview bullet), and §4 (the workflow's generator list) are
      updated; §5/§8's prose wasn't fully re-audited line by line — real
      remaining doc-drift risk, not verified clean.
- [ ] Ginkgo specs, by sub-area above: role inference (aggregate root /
      value object / domain event classification, including the
      convention fallback when nothing's explicitly marked) — **done**;
      protovalidate wiring (a violated constraint actually fails
      `Validate()`, generated against a real vendored constraint, not a
      hand-rolled check) — **done**, both in isolation and through a
      real `go run` exercising acceptance/rejection; event collection
      (`PullEvents` actually returns what an owned aggregate method
      appended, and is called by the generated application service) —
      **not done**: nothing in this batch's specs calls an owned
      aggregate method that appends an event and asserts `PullEvents`
      sees it, or that the application service pulls and publishes them
      — the mechanism (the `events []DomainEvent` slice + `PullEvents`)
      is generated and compiles, but its actual use is unexercised; the
      shared-kernel dedup case — **not done**, the shared kernel itself
      is deferred (see below); GORM embedded-value-object round-tripping
      against a real local Postgres, same standard
      the existing persistence suites already hold adapters to — **not
      done**: sqlgen still maps every field as its own flat column, the
      same as before this phase; it was never changed to give a value
      object GORM's native embedded-struct treatment, so there's nothing
      here for a spec to exercise yet; a full CLI e2e spec through the
      new layout end to end — **done** (`internal/commands/e2e_test.go`).

**Exit criteria:** a freshly generated project has the
`domain/application/infrastructure` layout above — **met**; an
aggregate's generated `Validate()` actually rejects data violating a
real `buf.validate` constraint — **met**, exercised through a real
`go run`, not just a content assertion; a hand-written aggregate
method's appended event is retrievable via `PullEvents()` and reaches
the no-op `EventPublisher` — **not met**: the mechanism is generated and
compiles, but nothing exercises an owned method actually appending an
event and the application service actually publishing it; a shared
value object referenced by two entities generates once, not twice —
**not met**, the shared kernel is deferred (task tracked separately,
unchanged from the earlier explicit `AskUserQuestion` deferral this
phase's intro already notes); the full e2e suite (init → generate proto
→ hand edit → generate code → `go build`) passes against the new
architecture end to end — **met**.

**Known, documented limitation carried over unchanged:** this
architecture is chosen once, at `sgo init` time (there's no reason to
think a later `layout:`-style override wouldn't hit the exact same
already-generated-files problem the earlier version of this phase
flagged — logged in Non-goals below, not solved by this phase either).

## Phase 13 — `sgo list endpoints` **(done)**

Small, independent of Phases 12/14/15 — ships fast, gets more useful
once Phase 15 lands proto-defined paths but doesn't depend on it. Full
design in ARCHITECTURE.md §16.

- [x] `sgo list endpoints [<service>]` — prints every HTTP route
      currently derived for the project's registered services (all of
      them with no argument, one with it): method, path, and which
      RPC/service it comes from. Reuses
      `internal/codegen/httpgen.BuildRoutes` directly — the exact same
      route list `sgo generate openapi` already turns into
      `docs/openapi.<ext>` and the generated `*_routes_gen.go` actually
      registers, not a second derivation.
- [x] Ginkgo specs (content per framework selection) plus a CLI e2e spec
      asserting the printed table matches the generated route file's own
      registrations, same cross-check `openapigen/generate_test.go`
      already does for the OpenAPI document.

**Exit criteria:** `sgo list endpoints` on a project with generated
services prints exactly the routes the generated HTTP adapter actually
registers — verified against the generated route file, not just
`BuildRoutes`'s own output a second time.

## Phase 14 — OpenAPI discoverability + a live Swagger/Redoc UI **(done)**

Full design in ARCHITECTURE.md §17. `sgo generate openapi` and `sgo
openapi validate` already exist (Phase 8) — this phase covers both
halves of what was actually missing: nothing in the terminal points you
at them, and there's no way to *browse* the generated document short of
opening the raw YAML/JSON.

- [x] **Discoverability** — `sgo generate code` prints a "Next steps"
      hint mentioning `sgo generate openapi` once a service exists, the
      same way `sgo init` already prints one for `sgo generate proto`;
      README's status line and quick start call it out explicitly
      (partly done already, finish the rest).
- [x] **`sgo openapi ui [--port 4749]`** — serves the project's
      generated `docs/openapi.<ext>` through an embedded interactive API
      doc viewer at `http://127.0.0.1:<port>` (same `127.0.0.1`-only, no
      auth stance as `sgo ui`/`sgo run --debug`, §11/§15). Reads
      whatever's already on disk — same "run `sgo generate openapi`
      first if it doesn't exist yet" UX `sgo openapi validate` already
      has, not auto-regenerating on every request.
- [x] Vendor a specific pinned version of a standalone doc-viewer bundle
      (evaluating Redoc's single-file standalone build against
      `swagger-ui-dist`'s multi-file one — whichever is smaller wins,
      same self-contained-by-default stance as the vendored OpenAPI
      meta-schemas in Phase 8) via `go:embed`, not a CDN `<script>` tag —
      works offline, consistent with every other "no external dependency
      at runtime" choice this project has made (pure-Go proto compiler,
      pre-installed browser in dev, vendored meta-schemas).
- [x] Ginkgo specs (`internal/openapiui`'s own `httptest`-based content
      specs, plus a real CLI e2e spec in `internal/commands` that starts
      the actual compiled binary as a background process and drives it
      over a real socket) plus a real-browser Playwright pass confirming
      the page actually renders the project's real paths — same standard
      `sgo ui` and the Phase 10 dashboard were held to. The Playwright
      pass was a manual one-off verification (screenshotted, not
      committed as an automated test), the same as task-list item
      "Manually verify sgo ui in a real browser" was for `sgo ui` itself
      — not part of `go test ./...`.

**Exit criteria:** `sgo generate code` visibly points at `sgo generate
openapi`; `sgo openapi ui` on a project with a generated doc serves a
real interactive viewer showing that project's actual endpoints,
verified in a real browser, no internet access required.

## Phase 15 — Proto-defined HTTP paths (`google.api.http`) + root path **(done)**

The riskiest and most novel piece of this batch — resolves the
`google.api.http` open question ARCHITECTURE.md §12/Decision #14 already
flagged as "a possible future upgrade if the convention-based routing
proves too rigid," rather than reversing that decision outright. Full
design in ARCHITECTURE.md §18. Decided via `AskUserQuestion` before any
code: **annotations are an optional override, not a replacement** — an
RPC with no `google.api.http` option keeps today's naming-convention
routing (`Create*`→`POST`, etc., §8.1); one with the option uses it
instead. Every existing generated project keeps working unchanged.

- [x] **Vendor `google/api/http.proto` + `google/api/annotations.proto`**
      (small, Apache-2.0, just the extension/option definitions — not
      the whole `googleapis` tree) so `sgo generate proto`-authored files
      can `import "google/api/annotations.proto";` and the pure-Go
      compiler (`bufbuild/protocompile`, Decision #5) can resolve that
      import from the vendored copy without a `protoc`/`buf` binary or
      network access — preserves the property Decision #5 already
      established, doesn't reopen it.
- [x] **`sgo`'s own `base_path` service option** — `google.api.http` has
      no concept of a service-level path prefix, only per-method rules,
      so a root path needs sgo's own small option:
      `option (sgo.base_path) = "/v1";` on the service (a proto
      extension in the 50000–99999 organization-reserved range) — a new
      field on `sgo/options.proto`, the same vendored file Phase 12
      already introduces for its own domain-modeling options
      (`(sgo.aggregate_root)` etc.), not a second vendored file. Applies
      to every route on that service, annotation-derived or
      convention-derived — replacing today's hardcoded `/api/v1` prefix
      (§8.1) with this as the default when unset, so nothing changes for
      a proto that doesn't opt in. (As implemented: an annotation-derived
      route only gets the prefix when `base_path` is explicitly set,
      since `google.api.http` paths are meant to already be complete —
      unconditionally prepending the default would double-prefix them.)
- [x] **`internal/codegen/httpgen.BuildRoutes`** — for each RPC, check
      for a `google.api.http` option first (method/path from
      `get`/`post`/`put`/`delete`/`patch`, path parameters from `{name}`
      bindings, request body from the `body` field); fall back to
      today's naming-convention derivation when absent. Path parameters
      translate to each framework's own syntax (Gin/Echo `:name`, Chi
      `{name}`) same as the existing `{id}` handling already does — not
      limited to a field literally named `id` any more, resolving the
      other open question §12 already flagged alongside this one. v1
      states, rather than silently mishandles, what it doesn't cover: a
      `custom` (non-verb) pattern, a wildcard segment, a
      `{name=sub/pattern}` variable, a dotted field path, a path
      parameter bound to a non-scalar field, and `body:"<field>"` each
      reject with a clear error instead of guessing at a mapping.
- [x] `sgo generate openapi` and `sgo list endpoints` (Phase 13) need no
      changes at all — both already just consume `BuildRoutes`'s output,
      so proto-defined paths show up in generated OpenAPI docs and
      `sgo list endpoints` for free once `BuildRoutes` itself understands
      them. Confirmed: neither needed any code change beyond the
      mechanical `PathParams`-loop generalization `BuildRoutes`'s own
      signature change required.
- [x] Ginkgo specs: a proto with an explicit `google.api.http` option
      generates the annotated route, not the convention-derived one; a
      proto without one is byte-identical to today's output (regression
      guard that the override really is optional); path-parameter
      translation per framework; the `base_path` override; plus a real
      end-to-end spec (`httpgen/annotated_e2e_test.go`) building a real
      project from an annotated proto, binding two independent path
      parameters on one route, and hitting the actual generated route
      over a real socket, same standard the HTTP CRUD e2e suite (Phase
      3/4) already holds generated adapters to. Ten focused specs
      (`httpgen/route_test.go`) cover every v1 constraint's rejection
      path individually.

**Exit criteria:** an RPC with a `google.api.http` option gets exactly
that route; one without keeps today's convention-derived route,
unchanged; a service with `(sgo.base_path)` set gets that prefix instead
of the default `/api/v1`; every existing generated-project e2e spec
still passes with no proto changes. All met — full repo suite green,
`gofmt`/`go vet` clean.

## Phase 16 — Repository-port methods beyond fixed CRUD **(done)**

Full design in ARCHITECTURE.md §21. Closes a real gap in the safe-
regeneration story (§6): the usecase port (`internal/core/port/in/
<entity>_usecase.go`, pre-Phase-12; now `internal/application/<entity>/
service.go`) already grows a new method for free when you add an RPC to
the proto and re-run `sgo generate code` — verified live against a real
generated project before writing this: adding an `ArchiveUser` RPC
produced `ArchiveUser` on the usecase interface and appended a matching
stub to the owned `user_service.go`, with the hand-written `CreateUser`
body untouched. But the repository port (`internal/core/port/out/
<entity>_repository.go`, pre-Phase-12; now `internal/domain/<entity>/
repository.go`) is *permanently* fixed at `Create/Get/List/Update/Delete`
(Decision #11) — adding `ArchiveUser` to the proto left it exactly as it
was. There is currently no supported way to add a repository method like
`FindByEmail(ctx, email string) (*User, error)`; hand-editing the
generated file works until the next `generate code` run silently
reverts it, since nothing protects it the way owned files are
protected.

Two decisions locked in via `AskUserQuestion` before any code:
**declaration is an option on an existing RPC**, not a new proto
convention for method signatures outside the service's RPC list — reuses
the RPC's own Request/Response message types rather than inventing a
second way to express a typed method signature. And **the in-memory
adapter auto-implements simple cases** (a single-field equality query
becomes a generated linear scan) rather than requiring a hand-written
stub everywhere, keeping local dev fully runnable with zero manual
adapter edits for the common case — the same "always runnable" promise
Decision #15 already makes for the fixed CRUD set.

- [x] **`option (sgo.repository_query) = true;`** on an RPC method —
      another field on the same vendored `sgo/options.proto` Phase 12
      introduces (extended by Phase 15 for `(sgo.base_path)`), not a
      third vendored file. A `MethodOptions` extension. Marks that RPC
      as *also* needing
      a repository-port counterpart, generated alongside its usual
      application-service method, HTTP route, and gRPC method — by
      design, per the decision above, this makes the RPC both a public
      endpoint and a repository method, not repository-only.
- [x] **`option (sgo.hide_route) = true;`** — a second, independent
      option on the same RPC that skips HTTP route registration for it
      (still generates the gRPC method and, if also marked
      `repository_query`, the repository counterpart). Cheap once the
      options-plumbing from the first bullet exists, and closes the
      "now it's forced to be a public HTTP endpoint too" tradeoff the
      `AskUserQuestion` answer explicitly accepted — worth having rather
      than leaving that as a flat limitation.
- [x] **Repository port generation** — for an RPC marked
      `repository_query`, derive the method name by removing every
      occurrence of the entity name from the RPC name (`FindUserByEmail`
      on entity `User` → `FindByEmail`; `ArchiveUser` → `Archive`),
      otherwise keep the RPC name as-is (`PurgeStale` stays `PurgeStale`).
      Parameters come from the request message's fields, flattened to
      individual scalar Go parameters in declaration order (matching
      `Get(ctx, id string)`'s existing style, not the application
      service's opaque `*Request` style) — **v1 constraint, stated
      plainly rather than silently unsupported**: every request field
      must be a scalar (no nested messages, no repeated fields); one
      that isn't fails generation with a clear error instead of guessing
      how to flatten it. A derived name colliding with the port's fixed
      `Create`/`Get`/`List`/`Update`/`Delete` methods is also rejected
      with a clear error, rather than surfacing as a confusing duplicate-
      method Go compiler error later. Return type: v1 only supports the
      single-entity shape (`*User, error`, the same shape `Get` already
      returns) — a list-shaped response (pagination) isn't supported
      yet, logged as a Non-goal below.
- [x] **Real persistence engines (Postgres/MySQL/MongoDB)** — each
      engine's adapter package gains a new *owned* companion file
      alongside its existing generated one (e.g. `postgres/
      user_repository.go` next to `postgres/user_repository_gen.go`),
      created once with a `panic("sgo: TODO implement FindByEmail")`
      stub per custom method, using the *exact same* append-only-new-
      stubs mechanism (Decision #12) already proven for
      `internal/application/<entity>/service.go` — a hand-written
      implementation survives every later `generate code` run, and a
      newly added custom method gets a fresh stub appended without
      touching what's already there. Implemented as one shared helper
      (`core.GenerateRepositoryQueryStubs`) all three generators call,
      not three separate implementations.
- [x] **In-memory adapter** — auto-implements a custom method when its
      signature is exactly one scalar parameter whose name
      case-insensitively matches an exported field on the domain entity
      (`email string` → `Email` field): generates a linear scan
      (`for _, e := range r.data { if e.Email == email { ... } }`)
      directly in the generated (not owned) memory adapter file, no
      hand-editing needed. Falls back to the same stub-in-an-owned-file
      pattern the real engines use when it can't confidently infer the
      mapping (more than one parameter, or no matching field) — fails
      toward "you write it," never toward a guess that silently returns
      wrong data.
- [x] Ginkgo specs: option parsing (a marked RPC produces the expected
      repository method; an unmarked one doesn't); the scalar-field and
      name-collision rejections with a clear error; the name-stripping
      rule (prefix, interior, and no-match cases); a full generate-code
      run per persistence engine (memgen/sqlgen/mongogen package suites,
      plus a real `sgo init` → proto → `sgo generate code` → `go build`/
      `go run` round-trip in `internal/codegen/generate_test.go`)
      asserting the owned stub file's existence and content on first
      generation, then that a hand-written implementation survives a
      second run (the two-survives-regeneration pattern already used
      throughout `internal/codegen`'s existing suites); the in-memory
      auto-implementation actually returning the right entity for a
      simple case, and correctly falling back to a stub for a
      multi-parameter one; `hide_route` actually suppressing the HTTP
      route while leaving the rest of the service unaffected.

**Exit criteria:** an RPC marked `(sgo.repository_query)` produces a
matching method on the repository port; a hand-written implementation
in each real engine's owned companion file survives regeneration, the
same way `service.go` already does; the in-memory adapter answers a
simple single-field query correctly with zero hand-editing; an RPC
additionally marked `(sgo.hide_route)` gets no HTTP route but keeps its
gRPC method and repository counterpart; every existing generated-project
e2e spec still passes with no proto changes. All met — full repo suite
green, `gofmt`/`go vet` clean.

## Phase 17 — Docs: README as a getting-started guide, CONTRIBUTING.md **(done)**

Deliberately last in this batch — it should describe the layout,
commands, and proto conventions Phases 11–16 actually land with, not
what's true today. Mechanical relative to the rest of this batch.

- [x] **README.md** restructured into a fuller guide: overview, install,
      getting started/quick start, full command reference pointer
      (`docs/CLI.md`), architecture pointer (`ARCHITECTURE.md`), FAQ/
      troubleshooting section if anything recurring surfaced by then.
      Still the repo's actual `README.md` (GitHub renders it on the repo
      homepage) — "as a wiki" means comprehensive and navigable, not a
      literal GitHub Wiki, which is a separate, harder-to-review,
      harder-to-PR surface than a file already in the repo. Landed with
      a table of contents plus dedicated sections on the generated-vs-
      owned file split, the `sgo.*`/`google.api.http` proto option
      reference (table + worked example), HTTP routing, repository
      queries beyond CRUD, persistence/cache/search, OpenAPI, the web
      UI, and a project-layout tree captured from a real `sgo init` +
      `generate proto` + `generate code` run rather than hand-typed.
- [x] **`CONTRIBUTING.md`** — the existing "Contributing / development"
      section moved out of README.md into its own file (standard
      GitHub convention: `CONTRIBUTING.md`, not `CONTRIBUTE.md` — GitHub
      links to it automatically from the "Contributing" prompt on a new
      issue/PR when it's named this way), expanded with the phase-based
      workflow this project actually uses (plan → `AskUserQuestion` →
      PLAN.md/ARCHITECTURE.md → implement → PR, not merged by the author)
      so an outside contributor understands the process before opening
      one.
- [x] Cross-check every command/path mentioned in both files against
      `docs/CLI.md` and the current generated layout — this phase is the
      one place drift between "what the docs say" and "what `sgo`
      actually does" gets caught for this whole batch. Caught and fixed
      along the way: `docs/CLI.md`'s `sgo init` flags table was missing
      `--openapi-version`/`--openapi-format`; the required Go version
      (1.25 → 1.26, bumped during Phase 15) was stale in the old README.

**Exit criteria:** README.md covers overview → install → getting started
end to end without needing `ARCHITECTURE.md`/`docs/CLI.md` open
side-by-side for a first-time user; `CONTRIBUTING.md` exists and is
what GitHub links to from a new issue/PR; nothing in either file
contradicts `docs/CLI.md` or the actual current layout. All met — every
path/tree in the new README was captured from a real generated project,
and the two docs-drift bugs found while cross-checking were fixed, not
just noted.

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
- Concrete KMS/Vault/cloud-secret-manager and remote-config-repo
  adapters — Phase 10 ships the `envsource.Source` interface and `.env`
  only; building a real adapter needs a concrete provider choice and
  real credentials to verify against, neither decided yet. Revisit once
  a specific provider is requested.
- General file-watching auto-reload (à la `air`/`nodemon`) for
  `sgo run` — Phase 10's supervisor only restarts on a dashboard-driven
  config change, nobody asked for source-file watching. Revisit only if
  requested.
- **Changing the generated architecture after a project already
  exists** — Phase 12's DDD structure is chosen once, at `sgo init`
  time, same as HTTP framework or persistence engine already are; safe
  regeneration (§6) finds owned files at their *current* expected path
  and shape, so retargeting that for an already-generated project needs
  a real migration this phase doesn't build.
- **A selectable architecture style (DDD vs. today's simpler
  hexagonal)** — explicitly decided against (`AskUserQuestion`): Phase
  12 replaces the default outright, it doesn't add a second option
  alongside it. Revisit only if a real need for the simpler shape
  resurfaces.
- **A real event-bus/message-broker adapter** (Kafka/RabbitMQ/NATS) —
  explicitly deferred (`AskUserQuestion`): Phase 12 builds the
  `EventPublisher` seam and a working no-op default, not a broker
  integration. Revisit once there's a concrete design for it (mentioned
  as a separate idea to bring later).
- **`infrastructure/clients/`** (outbound third-party API clients, e.g.
  a Stripe client) — nothing in sgo today generates an outbound API
  client of any kind; an empty placeholder directory isn't generated
  for a capability that doesn't exist yet, consistent with the same
  reasoning for `infrastructure/messaging/` above.
- A generic templating/plugin system for arbitrary custom generators —
  Phase 12 changes *what* sgo's fixed set of generators produce and
  *where* it lands, it doesn't let someone add a wholly new kind of
  generated file of their own design. Nobody's asked for that yet.
- A repository-only method that never becomes a public RPC — Phase 16's
  declaration mechanism is deliberately an option on an existing RPC
  (`AskUserQuestion`-confirmed), which always keeps that RPC's
  application-service method, and its gRPC method unless a fundamentally
  different mechanism is built later; `(sgo.hide_route)` only suppresses
  the HTTP route, not the whole public surface. Revisit if a genuinely
  internal-only query (no gRPC exposure either) turns out to be needed.
- A custom repository method whose request message has a nested-message
  field, or whose response is list-shaped (pagination) — Phase 16 only
  flattens scalar request fields and only supports the existing single-
  entity response wrapper shape (`Get`/`Create`/`Update`'s convention).
  Both fail generation with a clear error rather than guessing. Revisit
  if a real use case needs either.

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
- Phase 9 (loading indicators) is independent of Phase 10 and much
  smaller — sequenced first so it ships without waiting on the process
  supervisor/dashboard work, not because Phase 10 depends on it.
- Phase 11 (wizard polish) ships first in this batch — small, and
  entirely independent of 12–17.
- **Phase 12 (DDD architecture replacement) is now by a wide margin the
  riskiest, largest, and most foundational piece of this entire batch —
  larger than Phases 13–17 combined.** It touches every generator
  package that writes a file, same reasoning the original layout-only
  version of this phase had, but now also introduces the vendored
  `sgo/options.proto` custom-options mechanism Phases 15 and 16 both
  build on (`(sgo.base_path)`, `(sgo.repository_query)`,
  `(sgo.hide_route)` all become fields on the file Phase 12 vendors,
  not separately vendored). Sequenced right after Phase 11 and before
  everything else for both reasons at once: Phases 13–16 consume paths
  Phase 12 changes, and Phases 15/16 consume options infrastructure
  Phase 12 introduces — building either on top of a Phase 12 that
  hasn't landed yet would mean redoing work.
- Phases 13 and 14 are independent of each other and of Phase 15;
  sequenced 13-then-14 only because `sgo list endpoints` is the smaller
  of the two.
- Phase 15 (proto-defined HTTP paths) depends on nothing *functionally*
  in this batch beyond Phase 12's options mechanism, but is sequenced
  after 12–14 anyway: it's the second-largest, most novel piece here
  (a second vendored proto extension, per-framework path-parameter
  translation), and Phases 13/14 are more useful to have landed first
  since Phase 15 makes both of them richer for free (proto-defined
  routes show up in `sgo list endpoints` and generated OpenAPI docs
  without either needing changes) rather than the reverse.
- Phase 16 (repository-port methods) is sequenced right after Phase 15
  for the same options-file reason Phase 12 documents above — one
  vendored file, extended twice, not vendored three times.
- Phase 17 (docs) is deliberately last — it documents the architecture,
  commands, and proto conventions Phases 11–16 land with, not a
  snapshot from partway through. Given how much Phase 12 alone changes,
  this phase is doing real work here, not a light touch-up.
