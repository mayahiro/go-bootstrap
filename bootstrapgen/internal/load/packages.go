package load

import (
	"fmt"
	"go/token"

	"golang.org/x/tools/go/packages"
)

type Loaded struct {
	Package *packages.Package
	Fset    *token.FileSet
}

func Package(pattern string) (*Loaded, error) {
	fset := token.NewFileSet()
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
	}

	pkgs, err := packages.Load(cfg, pattern)
	if err != nil {
		return nil, err
	}

	if len(pkgs) != 1 {
		return nil, fmt.Errorf("expected 1 package, got %d", len(pkgs))
	}

	return &Loaded{
		Package: pkgs[0],
		Fset:    fset,
	}, nil
}
