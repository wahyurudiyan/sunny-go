// Package wiregen produces contract/gen's wire types by driving the real
// protoc-gen-go and protoc-gen-go-grpc plugins — the same generators
// `protoc`/`buf` would invoke — over a descriptor built by
// internal/codegen/proto, without requiring protoc or buf on PATH.
//
// The plugins are Go programs themselves, so they're run with
// `go run <module>@<version>` pinned to versions this repo already
// depends on (see protocGenGoVersion/protocGenGoGRPCVersion below). That
// invocation resolves independently of the caller's working directory —
// it works from inside a freshly generated project just as well as from
// this repo — so the only prerequisite is a Go toolchain, which anyone
// using sgo to generate a Go project already has.
package wiregen

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// Keep these in sync with the versions in go.mod.
const (
	protocGenGoVersion     = "v1.36.11"
	protocGenGoGRPCVersion = "v1.6.2"
)

// Generate runs protoc-gen-go and protoc-gen-go-grpc against fd and
// writes their output under destDir (contract/gen).
func Generate(fd protoreflect.FileDescriptor, destDir string) error {
	protoFiles := collectFileDescriptors(fd, map[string]bool{}, nil)

	if err := runPlugin("google.golang.org/protobuf/cmd/protoc-gen-go@"+protocGenGoVersion, fd.Path(), protoFiles, destDir); err != nil {
		return fmt.Errorf("protoc-gen-go: %w", err)
	}

	if err := runPlugin("google.golang.org/grpc/cmd/protoc-gen-go-grpc@"+protocGenGoGRPCVersion, fd.Path(), protoFiles, destDir); err != nil {
		return fmt.Errorf("protoc-gen-go-grpc: %w", err)
	}

	return nil
}

// collectFileDescriptors returns fd's FileDescriptorProto along with
// those of its full transitive import closure, so plugins can resolve
// any types the file imports (google/protobuf/timestamp.proto, etc.).
func collectFileDescriptors(fd protoreflect.FileDescriptor, seen map[string]bool, out []*descriptorpb.FileDescriptorProto) []*descriptorpb.FileDescriptorProto {
	if seen[fd.Path()] {
		return out
	}
	seen[fd.Path()] = true

	imports := fd.Imports()
	for i := 0; i < imports.Len(); i++ {
		out = collectFileDescriptors(imports.Get(i).FileDescriptor, seen, out)
	}

	return append(out, protodesc.ToFileDescriptorProto(fd))
}

func runPlugin(pkg, fileToGenerate string, protoFiles []*descriptorpb.FileDescriptorProto, destDir string) error {
	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{fileToGenerate},
		Parameter:      proto.String("paths=source_relative"),
		ProtoFile:      protoFiles,
	}

	reqBytes, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal CodeGeneratorRequest: %w", err)
	}

	cmd := exec.Command("go", "run", pkg)
	cmd.Stdin = bytes.NewReader(reqBytes)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, stderr.String())
	}

	var resp pluginpb.CodeGeneratorResponse
	if err := proto.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return fmt.Errorf("failed to parse plugin response: %w", err)
	}

	if resp.GetError() != "" {
		return fmt.Errorf("%s", resp.GetError())
	}

	for _, f := range resp.File {
		path := filepath.Join(destDir, f.GetName())
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", path, err)
		}
		if err := os.WriteFile(path, []byte(f.GetContent()), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}
	}

	return nil
}
