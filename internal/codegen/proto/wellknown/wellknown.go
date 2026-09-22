// Package wellknown embeds the proto files sgo vendors so a project's
// own protos can import them without a `protoc`/`buf` binary or network
// access: sgo's own custom options (sgo/options.proto, ARCHITECTURE.md
// §17), the vendored protovalidate schema (buf/validate/validate.proto),
// and the vendored `google.api.http` extension definitions
// (google/api/http.proto, google/api/annotations.proto,
// ARCHITECTURE.md §20).
package wellknown

import (
	"embed"
	"io"
	"io/fs"

	"github.com/bufbuild/protocompile"
)

//go:embed buf sgo google
var files embed.FS

// Resolver resolves an import path like "sgo/options.proto" or
// "buf/validate/validate.proto" against the embedded, vendored copies.
// Combine with a project's own local-filesystem resolver via
// protocompile.CompositeResolver, and wrap the result in
// protocompile.WithStandardImports for google/protobuf/*.proto.
func Resolver() protocompile.Resolver {
	return &protocompile.SourceResolver{
		Accessor: func(path string) (io.ReadCloser, error) {
			f, err := files.Open(path)
			if err != nil {
				return nil, err
			}
			return f, nil
		},
	}
}

// FS exposes the embedded tree directly, e.g. for a spec asserting a
// specific vendored file exists.
func FS() fs.FS { return files }
