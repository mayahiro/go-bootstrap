package testutil

import (
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

func CreateModule(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	module := "module example.com/test\n\ngo 1.26.1\n\nrequire github.com/mayahiro/go-bootstrap v0.0.0\n\nreplace github.com/mayahiro/go-bootstrap => " + repoRoot(t) + "\n"

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0o644); err != nil {
		t.Fatal(err)
	}

	for name, body := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(path, []byte(strings.TrimLeft(body, "\n")), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return dir
}

func LoadPackage(t *testing.T, dir string, pattern string) (*packages.Package, *token.FileSet) {
	t.Helper()

	fset := token.NewFileSet()
	cfg := &packages.Config{
		Dir: dir,
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo,
		Fset: fset,
	}

	pkgs, err := packages.Load(cfg, pattern)
	if err != nil {
		t.Fatal(err)
	}

	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}

	return pkgs[0], fset
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}

	root := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(file))))
	return root
}
