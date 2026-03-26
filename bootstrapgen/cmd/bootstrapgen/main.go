package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/mayahiro/go-bootstrap/bootstrapgen/internal/load"
	"github.com/mayahiro/go-bootstrap/bootstrapgen/internal/render"
	"github.com/mayahiro/go-bootstrap/bootstrapgen/internal/resolve"
	"github.com/mayahiro/go-bootstrap/bootstrapgen/internal/scan"
)

var readBuildInfo = debug.ReadBuildInfo

type options struct {
	output  string
	version bool
}

func main() {
	if err := runMain(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runMain(args []string, stdout io.Writer, stderr io.Writer) error {
	flags, opts := newFlagSet(stderr)
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	if opts.version {
		_, err := fmt.Fprintln(stdout, version())
		return err
	}

	pattern := "."
	switch flags.NArg() {
	case 0:
	case 1:
		pattern = flags.Arg(0)
	default:
		return fmt.Errorf("bootstrapgen accepts at most 1 package pattern")
	}

	return run(pattern, opts.output)
}

func newFlagSet(output io.Writer) (*flag.FlagSet, *options) {
	opts := &options{}
	flags := flag.NewFlagSet("bootstrapgen", flag.ContinueOnError)
	flags.SetOutput(output)
	flags.StringVar(&opts.output, "o", "bootstrap_gen.go", "output file name")
	flags.BoolVar(&opts.version, "version", false, "print version")
	flags.Usage = func() {
		fmt.Fprintf(output, "bootstrapgen %s\n\n", version())
		fmt.Fprintln(output, "Usage:")
		fmt.Fprintln(output, "  bootstrapgen [flags] [package]")
		fmt.Fprintln(output)
		fmt.Fprintln(output, "Flags:")
		flags.PrintDefaults()
	}
	return flags, opts
}

func version() string {
	info, ok := readBuildInfo()
	if !ok || info == nil || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return "devel"
	}
	return info.Main.Version
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
