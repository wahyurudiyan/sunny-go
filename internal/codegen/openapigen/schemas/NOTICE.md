# Vendored OpenAPI meta-schemas

`openapi-3.0.json` and `openapi-3.1.json` are the official JSON Schema
meta-schemas published by the [OpenAPI Initiative](https://www.openapis.org/)
for validating OpenAPI 3.0.x and 3.1.x documents, used by
`internal/codegen/openapigen`'s checker (`sgo openapi validate`) to give
real spec-compliance answers instead of a hand-rolled approximation —
see ARCHITECTURE.md §13.

- Source: [OAI/OpenAPI-Specification](https://github.com/OAI/OpenAPI-Specification),
  `schemas/v3.0/schema.json` and `schemas/v3.1/schema.json`.
- License: Apache License 2.0 (the OpenAPI Specification's license —
  vendoring the validation meta-schema is exactly the "build a
  compliant implementation" use it's meant for).
- `$id`: `https://spec.openapis.org/oas/3.0/schema/2019-04-02` and
  `https://spec.openapis.org/oas/3.1/schema/2021-03-02` respectively —
  these `$id`/`id` values are what identify the exact iteration.
- Retrieved via the [`@apidevtools/openapi-schemas`](https://www.npmjs.com/package/@apidevtools/openapi-schemas)
  npm package (MIT-licensed wrapper; the schema *content* itself is
  Apache-2.0 OAI content, mirrored verbatim from the spec repo's own
  build step), not fetched directly from the spec repo — at the time
  these were vendored, `OAI/OpenAPI-Specification`'s `main` branch no
  longer carries a `schemas/` directory at all (moved out-of-repo to
  https://spec.openapis.org, which wasn't reachable from this
  environment); the last tag known to still have `schemas/v3.0/` was
  `3.0.3`, and no tag in this repo ever shipped `schemas/v3.1/` — 3.1's
  meta-schema has always been spec.openapis.org-only. Re-verify this if
  re-vendoring later; the npm package's own `README.md`/build script
  documents it as cloning straight from the spec repo.
- No external `$ref`s outside each file — both are self-contained, no
  network fetch needed to validate against them (consistent with sgo
  never requiring `protoc`/`buf`, extended to this IDL — ARCHITECTURE.md
  §2).
