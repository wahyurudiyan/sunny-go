# sgo Development Plan

Phased roadmap implementing `ARCHITECTURE.md`. Each phase has a concrete
exit criterion so we know when to move on. Phases are ordered by
dependency, not necessarily by priority — see notes per phase.

## Phase 0 — Stabilize & clear the ground

Nothing user-facing changes yet; this makes the rest of the plan safe to
build on.

- [ ] Fix the duplicate `init()` double-registering `generateCmd` in
      `internal/commands/generate.go`.
- [ ] Restore/rewrite root `README.md` (currently contains a generated
      child project's README from a past accidental overwrite).
- [ ] Rename the binary/`Use` field from `sunny` to `sgo`; update
      `Makefile` build target accordingly. Module path stays
      `github.com/wahyurudiyan/sunny-go` (see ARCHITECTURE §Decisions #1).
- [ ] Delete `internal/templates/`, `internal/generate/`, and
      `internal/plugins/protoc-gen-go-http/` (confirmed dead/disconnected
      in the prior analysis) — superseded by `internal/codegen/` in later
      phases. Keep `internal/banner/` and the Cobra command skeleton.
- [ ] Add `internal/config` package with the `sgo.yaml` schema (read,
      write, validate) from ARCHITECTURE §7, with unit tests.

**Exit criteria:** `go build ./...` and `go vet ./...` clean, no dead
generator code left, `sgo.yaml` can be round-tripped (write → read →
equal), root README accurate.

## Phase 1 — Project scaffolding (`sgo init`, non-interactive)

- [ ] `sgo init <project> --http-framework <gin|echo|chi>
      --persistence-mode <orm|self-managed> --db <postgres,mysql,...>
      --cache redis --search elasticsearch` (flags only, no wizard yet).
- [ ] Directory scaffolding per ARCHITECTURE §3, conditioned on what was
      selected (no `mongo/` adapter dir if Mongo wasn't picked, etc.).
- [ ] `go:embed`-based template engine in `internal/template`, replacing
      the old string-constant approach.
- [ ] `docker/docker-compose.yml` generated with only the datastore
      services actually selected.
- [ ] `sgo.yaml` written with the resolved selections.

**Exit criteria:** `sgo init demo --http-framework gin --persistence-mode
orm --db postgres` produces a project that `go build`s (even though the
core/domain/service is still empty at this point) and whose
`docker-compose.yml` brings up exactly one Postgres service.

## Phase 2 — Proto → entities, ports, service skeletons

This is the heart of requirements #1–#3.

- [ ] `sgo generate proto <name>` → `contract/pb/<name>.proto` scaffold.
- [ ] `sgo generate code <name>`:
  - [ ] `buf build` (fallback: `protoc --descriptor_set_out=-`) → descriptor.
  - [ ] Walk descriptor via `protodesc`/`protoreflect`.
  - [ ] Run `protoc-gen-go`/`protoc-gen-go-grpc` into `contract/gen/`.
  - [ ] Generate `core/domain/<entity>/<entity>_gen.go` (fields) +
        `<entity>.go` (owned, created once).
  - [ ] Generate `core/port/in/<entity>_usecase.go` and
        `core/port/out/<entity>_repository.go`.
  - [ ] Generate `core/service/<entity>_service.go` (owned, created once
        with panic-stub method bodies for each use-case method).
  - [ ] Generate the wire↔domain mapper (`_gen.go`, always overwritten).
- [ ] **Safe-regeneration**: implement the `go/parser`-based method-set
      diff + stub-append described in ARCHITECTURE §6, with a test that
      asserts a hand-edited method body survives a second
      `sgo generate code` run after the proto gains a field/method.
- [ ] `sgo list` updated to reflect proto → entity → service → generated
      status per service (this already existed in a simpler form; keep
      the UX, rewire the data source).

**Exit criteria:** Editing a proto, running `generate code` twice, with a
hand-written method body added after the first run, and having that body
still present — verified in a test, not just by inspection.

## Phase 3 — HTTP framework adapters

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

## Phase 5 — `sgo init` interactive wizard

- [ ] Terminal wizard (`huh`, see ARCHITECTURE §10) covering project
      name/module, HTTP framework, persistence mode, datastores.
- [ ] Wizard calls the same scaffolding function as the Phase 1 flag-only
      path — no duplicated logic.
- [ ] Non-interactive flag path keeps working for CI/scripts (`--yes` or
      full flag set skips the prompt).

**Exit criteria:** `sgo init` with no flags launches the wizard;
`sgo init demo --http-framework gin ... ` still works non-interactively;
both produce an identical `sgo.yaml` for the same choices.

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

- [ ] Test coverage for `internal/codegen/*` (currently the whole repo
      has zero tests).
- [ ] End-to-end test: `sgo init` → `sgo generate proto` → edit proto →
      `sgo generate code` → `go build` the generated project in CI.
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
- Phase 5 and 6 both depend on Phases 1–4 being done, not on each other —
  6 could move earlier if the web UI is a priority over the terminal
  wizard, since both are thin frontends over the same engine.
