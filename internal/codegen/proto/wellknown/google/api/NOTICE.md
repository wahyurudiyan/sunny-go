# Vendored `google.api.http` extension definitions

`http.proto` and `annotations.proto` are the real option/extension
definitions [Google](https://github.com/googleapis) publishes for gRPC
Transcoding (`google.api.http`), used by `sgo`'s proto-defined HTTP
routing (ARCHITECTURE.md §20) — real, standards-based proto extensions,
not a convention sgo invents.

- Source: [googleapis/googleapis](https://github.com/googleapis/googleapis),
  `google/api/http.proto` and `google/api/annotations.proto`, fetched
  directly from the `master` branch.
- License: Apache License 2.0.
- Only these two files are vendored, not the rest of the `googleapis`
  tree — `annotations.proto` itself only imports `google/api/http.proto`
  and `google/protobuf/descriptor.proto` (the latter already resolved
  internally by `bufbuild/protocompile`, Decision #5), so this is
  everything a project's own proto needs to
  `import "google/api/annotations.proto";` and use
  `option (google.api.http) = { ... };` on an RPC — no `protoc`/`buf`
  binary or network access required, the same property every other
  vendored schema in this tree (`buf/validate`, `sgo/options.proto`)
  already preserves.
- `sgo` itself reads the compiled `(google.api.http)` extension via the
  real, official generated Go bindings,
  `google.golang.org/genproto/googleapis/api/annotations`
  (`annotations.E_Http`, `*annotations.HttpRule`) — a direct dependency
  of this module, not a hand-rolled reader — the same "read a real
  extension through its real generated type" approach
  `internal/codegen/proto/options.go` already uses for
  `buf.validate.field`.
- No external `$ref`s/imports beyond the two files above — self-
  contained, consistent with `buf/validate/NOTICE.md`.
