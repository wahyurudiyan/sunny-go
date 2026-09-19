package cachegen_test

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/cachegen"
)

var _ = Describe("ImportPath", func() {
	It("joins the module path with the Redis adapter's package location", func() {
		Expect(cachegen.ImportPath("demo")).To(Equal("demo/internal/adapter/out/cache/redis"))
	})
})

var _ = Describe("Generate", func() {
	var root string

	BeforeEach(func() {
		var err error
		root, err = os.MkdirTemp("", "sgo-cachegen-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })
	})

	It("writes the Cache port", func() {
		portDir := filepath.Join(root, "internal", "core", "port", "out")
		Expect(cachegen.GeneratePort(portDir)).To(Succeed())

		content, err := os.ReadFile(filepath.Join(portDir, "cache.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("type Cache interface"))
	})

	It("writes the Redis adapter", func() {
		destDir := filepath.Join(root, "internal", "adapter", "out", "cache", "redis")
		Expect(cachegen.GenerateRedis(destDir)).To(Succeed())

		content, err := os.ReadFile(filepath.Join(destDir, "cache_gen.go"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("github.com/redis/go-redis/v9"))
	})
})

// This exercises the generated Redis adapter against a real, locally
// running Redis — not just pattern-matching generated text — the same
// standard sqlgen's Postgres specs hold generated code to.
var _ = Describe("the generated Redis adapter, running for real", func() {
	It("gets, sets, and deletes keys against a real Redis", func() {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:6379", 500*time.Millisecond)
		if err != nil {
			Skip("no local Redis reachable on 127.0.0.1:6379: " + err.Error())
			return
		}
		conn.Close()

		root, err := os.MkdirTemp("", "sgo-cachegen-module-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		Expect(os.WriteFile(filepath.Join(root, "go.mod"), []byte("module demo\n\ngo 1.22\n"), 0644)).To(Succeed())

		destDir := filepath.Join(root, "internal", "adapter", "out", "cache", "redis")
		Expect(cachegen.GenerateRedis(destDir)).To(Succeed())

		main := `package main

import (
	"context"
	"fmt"
	"time"

	redis "demo/internal/adapter/out/cache/redis"
)

func main() {
	ctx := context.Background()
	client, err := redis.Connect(ctx)
	if err != nil {
		panic(err)
	}
	cache := redis.New(client)

	if err := cache.Set(ctx, "sgo:cachegen-test", "hello", 30*time.Second); err != nil {
		panic(err)
	}

	val, found, err := cache.Get(ctx, "sgo:cachegen-test")
	if err != nil {
		panic(err)
	}
	fmt.Printf("found=%v value=%q\n", found, val)

	if err := cache.Delete(ctx, "sgo:cachegen-test"); err != nil {
		panic(err)
	}

	_, found, err = cache.Get(ctx, "sgo:cachegen-test")
	if err != nil {
		panic(err)
	}
	fmt.Printf("found after delete=%v\n", found)
}
`
		mainDir := filepath.Join(root, "cmd", "harness")
		Expect(os.MkdirAll(mainDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(mainDir, "main.go"), []byte(main), 0644)).To(Succeed())

		tidy := exec.Command("go", "mod", "tidy")
		tidy.Dir = root
		tidyOut, err := tidy.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(tidyOut))

		cmd := exec.Command("go", "run", "./cmd/harness")
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(out))

		Expect(string(out)).To(ContainSubstring(`found=true value="hello"`))
		Expect(string(out)).To(ContainSubstring("found after delete=false"))
	})
})
