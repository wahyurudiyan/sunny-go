# Vendored Redoc standalone bundle

`redoc.standalone.js` is [Redocly](https://redocly.com/)'s single-file
standalone build of [Redoc](https://github.com/Redocly/redoc), used by
`sgo openapi ui` (ARCHITECTURE.md §19) to render a project's generated
`docs/openapi.<ext>` as an interactive API reference in the browser,
entirely offline.

- Source: the `redoc` npm package, `bundles/redoc.standalone.js`.
- Version vendored: 2.5.4.
- License: MIT (`package/LICENSE` in the `redoc` npm package; the
  bundle's own header additionally credits a few small MIT-licensed
  dependencies it inlines, e.g. `classnames` and `Stickyfill`).
- Chosen over `swagger-ui-dist` for this phase specifically because it
  vendors smaller and simpler: one self-contained ~1.1 MB file versus
  swagger-ui-dist's several files (bundle + standalone-preset + CSS +
  favicons) totaling roughly 2 MB, per PLAN.md Phase 14's explicit
  "whichever vendors smaller" call.
- No CDN `<script>` tag, no network fetch at runtime — embedded via
  `go:embed` and served from the `sgo` binary itself, consistent with
  every other "works with no network access" choice this project has
  made (the pure-Go proto compiler, the vendored OpenAPI meta-schemas,
  the vendored `sgo/options.proto`/`buf/validate` proto schemas).
- Usage: the served `index.html` includes `<redoc spec-url="...">`
  (Redoc's own custom element, which this bundle registers) pointed at
  the project's own `/openapi.yaml`/`/openapi.json` route, so re-running
  `sgo generate openapi` and refreshing the browser picks up regenerated
  docs with no server restart needed.
- Re-vendor by downloading the `redoc` npm package's tarball and taking
  `bundles/redoc.standalone.js` — no build step required, it's a
  pre-built bundle.
