package load

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTargetUsesDirectoryAsLoadRoot(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "cmd", "api")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	gotDir, gotPattern, err := target(root, "./cmd/api")
	if err != nil {
		t.Fatal(err)
	}

	if gotDir != dir {
		t.Fatalf("unexpected dir: %s", gotDir)
	}

	if gotPattern != "." {
		t.Fatalf("unexpected pattern: %s", gotPattern)
	}
}

func TestTargetPreservesPackagePattern(t *testing.T) {
	root := t.TempDir()

	gotDir, gotPattern, err := target(root, "./cmd/...")
	if err != nil {
		t.Fatal(err)
	}

	if gotDir != root {
		t.Fatalf("unexpected dir: %s", gotDir)
	}

	if gotPattern != "./cmd/..." {
		t.Fatalf("unexpected pattern: %s", gotPattern)
	}
}
