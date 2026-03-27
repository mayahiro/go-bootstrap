package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime/debug"
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
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	if err := run("./cmd/api", "bootstrap_gen.go"); err != nil {
		t.Fatal(err)
	}

	generated, err := os.ReadFile(filepath.Join(targetDir, "bootstrap_gen.go"))
	if err != nil {
		t.Fatal(err)
	}

	for _, fragment := range []string{
		"func runBootstrap(ctx context.Context) error {",
		"config2, err := config.Load()",
		"server2 := server.New(config2)",
		"return run(ctx, server2)",
	} {
		if !strings.Contains(string(generated), fragment) {
			t.Fatalf("generated code did not include %q:\n%s", fragment, string(generated))
		}
	}
}

func TestRunMainHelpUsesStableCommandName(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := runMain([]string{"-h"}, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}

	output := stderr.String()
	for _, fragment := range []string{
		"bootstrapgen devel",
		"Usage:",
		"bootstrapgen [flags] [package]",
		"-version",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("help output did not include %q:\n%s", fragment, output)
		}
	}

	if strings.Contains(output, "Usage of ") {
		t.Fatalf("help output still includes default flag path:\n%s", output)
	}
}

func TestRunMainVersionPrintsBuildInfoVersion(t *testing.T) {
	previous := readBuildInfo
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{
				Version: "v9.9.9",
			},
		}, true
	}
	defer func() {
		readBuildInfo = previous
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := runMain([]string{"-version"}, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}

	if stdout.String() != "v9.9.9\n" {
		t.Fatalf("unexpected version output: %q", stdout.String())
	}

	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr output: %q", stderr.String())
	}
}

func TestRunGeneratesBootstrapFileFromPackageDirectory(t *testing.T) {
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

func Load() *Config { return &Config{} }
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
		"config2 := config.Load()",
		"server2 := server.New(config2)",
		"return run(ctx, server2)",
	} {
		if !strings.Contains(string(generated), fragment) {
			t.Fatalf("generated code did not include %q:\n%s", fragment, string(generated))
		}
	}
}

func TestRunGeneratesCollisionSafeNames(t *testing.T) {
	dir := testutil.CreateModule(t, map[string]string{
		"cmd/api/bootstrap.go": `
package main

import (
	"context"

	"github.com/mayahiro/go-bootstrap/bootstrap"
	"example.com/test/internal/service"
	"example.com/test/internal/youtube"
)

type Params struct {
	bootstrap.In
	Service *youtube.Service
}

func run(context.Context, *service.Service, Params) error { return nil }

var spec = bootstrap.Server(
	"api",
	bootstrap.Provide(
		service.New,
		youtube.New,
	),
	bootstrap.Entry(run),
)
`,
		"internal/service/service.go": `
package service

type Service struct{}

func New() *Service { return &Service{} }
`,
		"internal/youtube/service.go": `
package youtube

type Service struct{}

func New() *Service { return &Service{} }
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
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	if err := run("./cmd/api", "bootstrap_gen.go"); err != nil {
		t.Fatal(err)
	}

	generated, err := os.ReadFile(filepath.Join(targetDir, "bootstrap_gen.go"))
	if err != nil {
		t.Fatal(err)
	}

	for _, fragment := range []string{
		"service2 := service.New()",
		"youtubeService := youtube.New()",
		"params := Params{",
		"Service: youtubeService,",
		"return run(ctx, service2, params)",
	} {
		if !strings.Contains(string(generated), fragment) {
			t.Fatalf("generated code did not include %q:\n%s", fragment, string(generated))
		}
	}
}
