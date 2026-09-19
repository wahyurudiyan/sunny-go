package commands_test

// Phase 6's stated exit criterion (PLAN.md): "sgo ui scaffolds a new
// project and runs generate code on an existing one entirely from the
// browser, calling the exact same code paths as the CLI (verified by
// one shared integration test hitting both the CLI command and the
// /api handler)." This is that test: it drives the real sgo binary
// through init -> generate proto -> generate code on one project, drives
// internal/webui's handlers through the equivalent POST /api/init ->
// POST /api/services -> POST /api/services/{name}/generate on another,
// and asserts they produce the same generated file set and an
// equivalent sgo.yaml, not just "both happen to work".

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wahyurudiyan/sunny-go/internal/config"
	"github.com/wahyurudiyan/sunny-go/internal/webui"
)

// relativeFiles walks root and returns every regular file's path
// relative to root, sorted, for a structural (not path-string) diff
// between two differently-named generated projects.
func relativeFiles(root string) []string {
	GinkgoHelper()

	var files []string
	Expect(filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		Expect(err).NotTo(HaveOccurred())
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		Expect(err).NotTo(HaveOccurred())
		files = append(files, rel)
		return nil
	})).To(Succeed())

	sort.Strings(files)
	return files
}

func apiJSON(handler http.Handler, method, path string, body any) map[string]any {
	GinkgoHelper()

	var buf bytes.Buffer
	if body != nil {
		Expect(json.NewEncoder(&buf).Encode(body)).To(Succeed())
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	Expect(rec.Code).To(BeNumerically("<", 300), rec.Body.String())

	var parsed map[string]any
	Expect(json.Unmarshal(rec.Body.Bytes(), &parsed)).To(Succeed())
	return parsed
}

var _ = Describe("the web UI and the CLI produce the same result", func() {
	It("scaffolds a project and generates a service identically through both surfaces", func() {
		root, err := os.MkdirTemp("", "sgo-webui-parity-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })

		// Path A: the real CLI binary, exactly as a user would run it.
		cliRoot := filepath.Join(root, "cli")
		Expect(os.MkdirAll(cliRoot, 0755)).To(Succeed())

		out, err := runSgo(cliRoot, "init", "shop",
			"--http-framework", "gin", "--persistence-mode", "orm")
		Expect(err).NotTo(HaveOccurred(), out)
		cliProjectDir := filepath.Join(cliRoot, "shop")

		out, err = runSgo(cliProjectDir, "generate", "proto", "product")
		Expect(err).NotTo(HaveOccurred(), out)

		out, err = runSgo(cliProjectDir, "generate", "code", "product")
		Expect(err).NotTo(HaveOccurred(), out)

		// Path B: internal/webui's handlers, driven the way the frontend
		// drives them — no shelling out to the binary at all.
		apiRoot := filepath.Join(root, "api")
		Expect(os.MkdirAll(apiRoot, 0755)).To(Succeed())
		h := webui.NewServer(apiRoot).Handler()

		apiJSON(h, http.MethodPost, "/api/init", map[string]any{
			"name": "shop", "httpFramework": "gin", "persistenceMode": "orm",
		})
		apiProjectDir := filepath.Join(apiRoot, "shop")

		apiJSON(h, http.MethodPost, "/api/services", map[string]any{"name": "product"})
		apiJSON(h, http.MethodPost, "/api/services/product/generate", nil)

		// Same generated file set, entity by entity, layer by layer —
		// not just "a project got created and something got generated".
		// Both runs used the same project name ("shop"), so every file's
		// content should be byte-identical too, not just its path — the
		// strongest available proof that the two surfaces are calling the
		// same generators rather than two implementations that happen to
		// agree on a file list.
		cliFiles := relativeFiles(cliProjectDir)
		apiFiles := relativeFiles(apiProjectDir)
		Expect(apiFiles).To(Equal(cliFiles))

		for _, rel := range cliFiles {
			cliContent, err := os.ReadFile(filepath.Join(cliProjectDir, rel))
			Expect(err).NotTo(HaveOccurred())
			apiContent, err := os.ReadFile(filepath.Join(apiProjectDir, rel))
			Expect(err).NotTo(HaveOccurred())
			Expect(apiContent).To(Equal(cliContent), "content differs for "+rel)
		}

		// Same sgo.yaml shape, modulo the field (module) that's
		// necessarily project-name-derived.
		cliCfg, err := config.Load(cliProjectDir)
		Expect(err).NotTo(HaveOccurred())
		apiCfg, err := config.Load(apiProjectDir)
		Expect(err).NotTo(HaveOccurred())
		Expect(apiCfg.HTTPFramework).To(Equal(cliCfg.HTTPFramework))
		Expect(apiCfg.Persistence).To(Equal(cliCfg.Persistence))
		Expect(apiCfg.Services).To(Equal(cliCfg.Services))

		// And both are real, buildable Go projects — a byte-identical
		// file list proves nothing if neither compiles.
		for _, dir := range []string{cliProjectDir, apiProjectDir} {
			build := exec.Command("go", "build", "./...")
			build.Dir = dir
			buildOut, err := build.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(buildOut))
		}
	})
})
