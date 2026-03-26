package load

import (
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

type Loaded struct {
	Package *packages.Package
	Fset    *token.FileSet
}

func Package(cwd string, pattern string) (*Loaded, error) {
	fset := token.NewFileSet()
	dir, query, err := target(cwd, pattern)
	if err != nil {
		return nil, err
	}

	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo,
		Fset: fset,
		Dir:  dir,
	}

	pkgs, err := packages.Load(cfg, query)
	if err != nil {
		return nil, err
	}

	if len(pkgs) != 1 {
		paths := make([]string, 0, len(pkgs))
		for _, pkg := range pkgs {
			if pkg.PkgPath != "" {
				paths = append(paths, pkg.PkgPath)
			}
		}

		if len(paths) > 0 {
			return nil, fmt.Errorf("expected 1 package for %q from %s, got %d: %s", query, dir, len(pkgs), strings.Join(paths, ", "))
		}

		return nil, fmt.Errorf("expected 1 package for %q from %s, got %d", query, dir, len(pkgs))
	}

	return &Loaded{
		Package: pkgs[0],
		Fset:    fset,
	}, nil
}

func target(cwd string, pattern string) (string, string, error) {
	if cwd == "" {
		cwd = "."
	}

	if pattern == "" {
		pattern = "."
	}

	path := pattern
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, pattern)
	}

	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		return path, ".", nil
	}

	if err != nil && !os.IsNotExist(err) {
		return "", "", err
	}

	return cwd, pattern, nil
}
