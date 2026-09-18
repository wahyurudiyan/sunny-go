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

## Phase 3 — HTTP framework adapters

**Depends on resolving ARCHITECTURE §12's `google.api.http` open
question first** — Phase 2's descriptor walk does not currently capture
HTTP route annotations (the starter proto template doesn't import them,
deliberately, to keep the pure-Go compiler dependency-free for Phase 2).
Either add that import and resolve it, or derive routes from the CRUD
RPC-naming convention the starter template already commits to
(Create→POST, Get→GET/{id}, List→GET, Update→PUT/{id}, Delete→DELETE/{id}
is the obvious mapping, given every proto sgo scaffolds already follows
it) — decide before starting the checklist below.

- [ ] `Router` interface (ARCHITECTURE §8.1) + `internal/registry/http.go`.
- [ ] Gin adapter (first, since it's the current `--http-framework`
      default).
- [ ] Echo adapter.
- [ ] Chi adapter.
- [ ] Generated `<entity>_routes_gen.go` per framework, driven by the
      `google.api.http` options captured in Phase 2's descriptor walk
      (so HTTP paths/methods come from the proto, not a hardcoded
      `/api/v1/<entity>s` guess like the current implementation).
- [ ] `cmd/<project>/main.go` template wires the framework selected in
      `sgo.yaml`.

**Exit criteria:** A generated project with any of the three frameworks
selected starts an HTTP server whose routes match the proto's
`google.api.http` annotations, and a gRPC server on a separate port,
both backed by the same service implementation.

## Phase 4 — Persistence, cache, search

- [ ] `core/port/out/<entity>_repository.go` self-managed adapter
      (`database/sql`/`sqlx`) for Postgres, MySQL.
- [ ] Same port, GORM-backed adapter, selectable via `sgo.yaml`
      `persistence.mode`.
- [ ] MongoDB adapter (`mongo-driver`), same repository port.
- [ ] Redis cache adapter behind `core/port/out/cache.go`.
- [ ] Elasticsearch adapter behind `core/port/out/search.go`.
- [ ] `internal/bootstrap/wire_gen.go` composition root wires whichever
      adapters `sgo.yaml` selected into the service layer.
- [ ] `docker-compose.yml` service blocks + `internal/adapter/out/*`
      connection config for each datastore (DSN/env vars).

**Exit criteria:** A project generated with Postgres+ORM+Redis boots,
connects to both from `docker-compose up`, and the generated repository
adapter satisfies the port interface (compiles against Phase 2's
generated port).

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

## Phase 6 — Web UI mode

- [ ] `sgo ui [--port 4747]`: localhost HTTP server, `go:embed` static
      frontend.
- [ ] `/api/*` reusing `internal/codegen` + `internal/config` — init
      wizard equivalent, `generate` trigger, `sgo.yaml` viewer/editor,
      generated-vs-owned file status per service.
- [ ] No auth (localhost-only binding); flagged as a later item if remote
      access is ever requested.

**Exit criteria:** `sgo ui` scaffolds a new project and runs
`generate code` on an existing one entirely from the browser, calling
the exact same code paths as the CLI (verified by one shared integration
test hitting both the CLI command and the `/api` handler).

## Phase 7 — Polish

- [ ] Ginkgo test coverage for the rest of `internal/codegen/*` (Phase 0
      set the pattern via `internal/config`; everything since should
      already have specs — this item is about closing gaps, not starting
      from zero).
- [ ] End-to-end test: `sgo init` → `sgo generate proto` → edit proto →
      `sgo generate code` → `go build` the generated project in CI, as a
      Ginkgo spec.
- [ ] Example project committed under `examples/` or generated in CI and
      thrown away — pick one, don't do both.
- [ ] Update root `README.md`, `ARCHITECTURE.md`, `docs/CLI.md` for drift
      accumulated during Phases 1–6.

## Non-goals (for now)

- Multi-service monorepo orchestration beyond one `sgo.yaml` per repo.
- Auth/remote access for `sgo ui`.
- Auto-migrating hand-written code when a proto field/method is removed
  (the compiler is the safety net, per ARCHITECTURE §6.4).
- Bundling/vendoring `buf` itself — documented prerequisite for now.

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
