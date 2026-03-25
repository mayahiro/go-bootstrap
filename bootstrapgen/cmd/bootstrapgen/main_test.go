package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mayahiro/go-bootstrap/bootstrapgen/internal/testutil"
)

func TestRunGeneratesBootstrapFile(t *testing.T) {
	dir := testutil.CreateModule(t, map[string]string{
		"cmd/api/bootstrap.go": `
package main

import (
	"context"

	"github.com/mayahiro/go-bootstrap/bootstrap"
	"example.com/test/internal/config"
	"example.com/test/internal/server"
)

func run(context.Context, *server.Server) error { return nil }

var spec = bootstrap.Server(
	"api",
	bootstrap.Provide(
		config.Load,
		server.New,
	),
	bootstrap.Entry(run),
)
`,
		"internal/config/config.go": `
package config

type Config struct{}

func Load() (*Config, error) { return &Config{}, nil }
`,
		"internal/server/server.go": `
package server

import "example.com/test/internal/config"

type Server struct{}

func New(*config.Config) *Server { return &Server{} }
`,
	})

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if chdirErr := os.Chdir(cwd); chdirErr != nil {
			t.Fatal(chdirErr)
		}
	}()

	targetDir := filepath.Join(dir, "cmd", "api")
	if err := os.Chdir(targetDir); err != nil {
		t.Fatal(err)
	}

	if err := run(".", "bootstrap_gen.go"); err != nil {
		t.Fatal(err)
	}

	generated, err := os.ReadFile(filepath.Join(targetDir, "bootstrap_gen.go"))
	if err != nil {
		t.Fatal(err)
	}

	for _, fragment := range []string{
		"func runBootstrap(ctx context.Context) error {",
		"config, err := config.Load()",
		"server := server.New(config)",
		"return run(ctx, server)",
	} {
		if !strings.Contains(string(generated), fragment) {
			t.Fatalf("generated code did not include %q:\n%s", fragment, string(generated))
		}
	}
}
