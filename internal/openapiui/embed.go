// Package openapiui implements `sgo openapi ui` (ARCHITECTURE.md §19):
// a localhost-only HTTP server that renders a project's already-
// generated docs/openapi.<ext> through a vendored, embedded Redoc
// viewer — no CDN, no network access, no regeneration on every
// request (same "run `sgo generate openapi` first if it's missing"
// contract `sgo openapi validate` already has).
package openapiui

import "embed"

//go:embed vendor/redoc.standalone.js
var vendorFS embed.FS
