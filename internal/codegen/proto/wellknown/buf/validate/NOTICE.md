# Vendored protovalidate schema

`validate.proto` is the official [protovalidate](https://protovalidate.com/)
schema — the `MessageOptions`/`FieldOptions`/`OneofOptions` extensions
(`buf.validate.message`, `buf.validate.field`, `buf.validate.oneof`) a
proto file uses to declare real, standards-based validation constraints,
used by `internal/domain`'s generated `Validate()` methods
(ARCHITECTURE.md §17) — real invariant enforcement via
[`buf.build/go/protovalidate`](https://buf.build/go/protovalidate) at
runtime in a generated project, not sgo hand-rolling CEL evaluation
itself.

- Source: [bufbuild/protovalidate](https://github.com/bufbuild/protovalidate),
  `proto/protovalidate/buf/validate/validate.proto`, `main` branch.
- License: Apache License 2.0 (included verbatim in the file's own
  header) — vendoring a schema to build a compliant implementation
  against is exactly what it's meant for, same reasoning as the
  vendored OpenAPI meta-schemas (`internal/codegen/openapigen/schemas/`).
- `syntax = "proto2";` — protovalidate's own file uses proto2 (needed
  for its extension-field declarations); a proto3 file can still
  `import` it and use its options normally, and sgo's compiler
  (`bufbuild/protocompile`) resolves the mixed-syntax import without
  issue — verified directly against a real compile, not assumed.
- Vendored whole, not a hand-trimmed subset — the same "vendor the real
  spec" principle used for OpenAPI's meta-schemas and (ARCHITECTURE.md
  §20) `google.api.http`, so a proto author gets protovalidate's actual,
  full constraint vocabulary rather than whatever slice sgo happened to
  need for its own generated constraints.
- No external `$ref`/import outside `google/protobuf/descriptor.proto`,
  which `bufbuild/protocompile` already resolves internally
  (`protocompile.WithStandardImports`) — no network fetch needed to
  compile against it, consistent with sgo never requiring `protoc`/`buf`
  (ARCHITECTURE.md §2/Decision #5).
