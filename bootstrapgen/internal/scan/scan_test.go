package scan

import (
	"strings"
	"testing"

	"github.com/mayahiro/go-bootstrap/bootstrapgen/internal/testutil"
)

func TestPackageRecordsPositions(t *testing.T) {
	dir := testutil.CreateModule(t, map[string]string{
		"cmd/api/bootstrap.go": `
package main

import (
	"context"

	"github.com/mayahiro/go-bootstrap/bootstrap"
)

type Config struct{}
type Server struct{}

func NewConfig() *Config { return &Config{} }
func NewServer(*Config) *Server { return &Server{} }
func run(context.Context, *Server) error { return nil }

var spec = bootstrap.Server(
	"api",
	bootstrap.Provide(
		NewConfig,
		NewServer,
	),
	bootstrap.Entry(run),
)
`,
	})

	pkg, fset := testutil.LoadPackage(t, dir, "./cmd/api")
	spec, err := Package(pkg, fset)
	if err != nil {
		t.Fatal(err)
	}

	if spec.Position.File != "bootstrap.go" || spec.Position.Line == 0 {
		t.Fatalf("unexpected spec position: %+v", spec.Position)
	}

	if len(spec.Providers) != 2 {
		t.Fatalf("unexpected provider count: %d", len(spec.Providers))
	}

	if spec.Providers[0].Position.File != "bootstrap.go" || spec.Entry.Position.File != "bootstrap.go" {
		t.Fatalf("positions were not recorded: provider=%+v entry=%+v", spec.Providers[0].Position, spec.Entry.Position)
	}
}

func TestPackageErrorIncludesPosition(t *testing.T) {
	dir := testutil.CreateModule(t, map[string]string{
		"cmd/api/bootstrap.go": `
package main

import "github.com/mayahiro/go-bootstrap/bootstrap"

var spec = bootstrap.Server("api")
`,
	})

	pkg, fset := testutil.LoadPackage(t, dir, "./cmd/api")
	_, err := Package(pkg, fset)
	if err == nil {
		t.Fatal("expected error")
	}

	message := err.Error()
	if !strings.Contains(message, "bootstrap.go:5:12") {
		t.Fatalf("error did not include position: %s", message)
	}

	if !strings.Contains(message, "Entry is required") {
		t.Fatalf("error did not include message: %s", message)
	}
}
