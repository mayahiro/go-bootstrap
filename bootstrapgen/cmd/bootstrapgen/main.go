package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mayahiro/go-bootstrap/bootstrapgen/internal/load"
	"github.com/mayahiro/go-bootstrap/bootstrapgen/internal/render"
	"github.com/mayahiro/go-bootstrap/bootstrapgen/internal/resolve"
	"github.com/mayahiro/go-bootstrap/bootstrapgen/internal/scan"
)

func main() {
	var output string

	flag.StringVar(&output, "o", "bootstrap_gen.go", "output file name")
	flag.Parse()

	pattern := "."
	if flag.NArg() > 0 {
		pattern = flag.Arg(0)
	}

	if err := run(pattern, output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(pattern string, output string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	pkg, err := load.Package(cwd, pattern)
	if err != nil {
		return err
	}

	spec, err := scan.Package(pkg.Package, pkg.Fset)
	if err != nil {
		return err
	}

	plan, err := resolve.Build(spec)
	if err != nil {
		return err
	}

	code, err := render.Go(plan)
	if err != nil {
		return err
	}

	target := filepath.Join(spec.Directory, output)
	return os.WriteFile(target, code, 0o644)
}
