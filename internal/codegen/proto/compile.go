package proto

import (
	"context"
	"fmt"
	"strings"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"github.com/bufbuild/protocompile/reporter"
)

// Compile parses and links relPath (found under protoDir, e.g.
// "user.proto" under "contract/pb") into a fully-linked file, using a
// pure-Go compiler — no protoc or buf binary required on PATH.
func Compile(protoDir, relPath string) (linker.File, error) {
	var errs []string

	compiler := protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{ImportPaths: []string{protoDir}},
		Reporter: reporter.NewReporter(func(err reporter.ErrorWithPos) error {
			errs = append(errs, err.Error())
			return nil
		}, nil),
	}

	files, err := compiler.Compile(context.Background(), relPath)
	if err != nil {
		if len(errs) > 0 {
			return nil, fmt.Errorf("failed to compile %s:\n%s", relPath, strings.Join(errs, "\n"))
		}
		return nil, fmt.Errorf("failed to compile %s: %w", relPath, err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no file compiled for %s", relPath)
	}

	return files[0], nil
}
