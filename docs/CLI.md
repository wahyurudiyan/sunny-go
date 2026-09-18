# `sgo` command reference

Target command surface for the architecture in `ARCHITECTURE.md`. Not all
commands exist yet — see `PLAN.md` for phasing. Commands below marked
**(planned)** don't exist on the current branch.

## `sgo init`

```
sgo init [project-name] [flags]
```

Scaffolds a new project (ARCHITECTURE §3) and writes `sgo.yaml`.

- No flags, TTY attached → launches the interactive selection wizard
  (ARCHITECTURE §10): project name/module, HTTP framework, persistence
  mode, datastores. **(planned, Phase 5)**
- Flags provided, or non-TTY → scaffolds directly, no prompts.
  **(planned, Phase 1)**

Flags:

| Flag | Values | Default | Notes |
|---|---|---|---|
| `--module` | Go module path | `<project-name>` | |
| `--http-framework` | `gin`, `echo`, `chi` | `gin` | |
| `--persistence-mode` | `orm`, `self-managed` | `orm` | |
| `--db` | comma list of `postgres`, `mysql`, `mongo` | none | |
| `--cache` | comma list, currently only `redis` | none | |
| `--search` | comma list, currently only `elasticsearch` | none | |
| `--yes` | bool | `false` | Skip the wizard even with a TTY, use defaults/flags as given |

Replaces the current `sunny create`/`sunny new`.

## `sgo generate proto <name>`

```
sgo generate proto user
```

Creates `contract/pb/user.proto` from a starter service+CRUD template.
Fails if the file already exists (edit it directly, don't regenerate a
proto over hand-written changes). **(planned, Phase 2)**

## `sgo generate code <name>`

```
sgo generate code user
```

Reads `contract/pb/user.proto`, regenerates:

- `contract/gen/*.pb.go`, `*_grpc.pb.go`
- `internal/core/domain/user/user_gen.go`
- `internal/core/port/in/user_usecase.go`,
  `internal/core/port/out/user_repository.go`
- HTTP route registration + gRPC server binding for the framework set in
  `sgo.yaml`
- the wire↔domain mapper

...and **creates, but never overwrites**, the owned files:
`internal/core/domain/user/user.go`,
`internal/core/service/user_service.go`. If the port interface gained
methods since the last run, stubs are appended to the owned service file
instead of the whole file being rewritten — see ARCHITECTURE §6. Safe to
run repeatedly. **(planned, Phase 2, HTTP wiring lands in Phase 3)**

Supersedes the current `sunny generate contract proto` / `sunny generate
api` / `sunny generate service` three-step flow — the new `generate code`
is one step that does entity + ports + service + HTTP + gRPC together,
because they all derive from the same proto walk.

## `sgo list`

```
sgo list services
```

Lists services tracked in `sgo.yaml`, and for each: proto present?
`contract/gen` up to date? owned service file present? Same UX as the
current `sunny list services`, rewired to the new file layout.

## `sgo validate <proto-file>`

```
sgo validate contract/pb/user.proto
```

Runs `buf build`/`protoc` against the file and reports errors. Same as
current `sunny validate`.

## `sgo ui` **(planned, Phase 6)**

```
sgo ui [--port 4747]
```

Starts a localhost-only web UI (ARCHITECTURE §11) covering the same
selections as `sgo init`'s wizard, plus a project dashboard (services,
generated-vs-owned file status, `sgo.yaml` viewer/editor). Binds to
`127.0.0.1` only.

## Removed/renamed from the current CLI

| Current (`sunny`) | New (`sgo`) | Why |
|---|---|---|
| `sunny create` / `sunny new` | `sgo init` | Matches the spec's wording and common CLI convention (`npm init`, `git init`). |
| `sunny generate contract proto <name>` | `sgo generate proto <name>` | Same behavior, shorter path. |
| `sunny generate api <name>` + `sunny generate service <name>` | `sgo generate code <name>` | Collapsed into one command since both derive from the same proto descriptor walk; the old two-step flow also didn't actually read the proto contents (see prior analysis) — the new one does. |
| `--http-framework fiber\|gin\|echo` | `--http-framework gin\|echo\|chi` (in `sgo init`) | Fiber dropped, Chi added, per spec #4. Framework choice moves from a per-`create` flag with no follow-through to a `sgo.yaml`-persisted, adapter-backed choice. |
