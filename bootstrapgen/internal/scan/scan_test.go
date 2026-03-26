package scan

import (
	"strings"
	"testing"

	"github.com/mayahiro/go-bootstrap/bootstrapgen/internal/model"
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

func TestPackageParsesIncludedModuleParamsAndHookFunc(t *testing.T) {
	dir := testutil.CreateModule(t, map[string]string{
		"cmd/api/bootstrap.go": `
package main

import (
	"context"

	"github.com/mayahiro/go-bootstrap/bootstrap"
	"example.com/test/internal/di"
	"example.com/test/internal/server"
)

type Params struct {
	bootstrap.In
	Runner server.Runner
}

func startAudit(context.Context, *server.Server) error { return nil }
func stopAudit(*server.Server) {}
func run(ctx context.Context, params Params) error { return params.Runner.Run(ctx) }

var spec = bootstrap.Server(
	"api",
	bootstrap.Include(di.Module),
	bootstrap.Lifecycle(
		bootstrap.HookFunc(startAudit, stopAudit),
	),
	bootstrap.Entry(run),
)
`,
		"internal/di/module.go": `
package di

import (
	"github.com/mayahiro/go-bootstrap/bootstrap"
	"example.com/test/internal/config"
	"example.com/test/internal/server"
)

var Module = bootstrap.Module(
	bootstrap.Provide(
		config.Load,
		server.New,
	),
	bootstrap.Bind(
		(*server.Runner)(nil),
		(*server.Server)(nil),
	),
)
`,
		"internal/config/config.go": `
package config

type Config struct{}

func Load() *Config { return &Config{} }
`,
		"internal/server/server.go": `
package server

import (
	"context"

	"example.com/test/internal/config"
)

type Runner interface {
	Run(context.Context) error
}

type Server struct{}

func New(*config.Config) *Server { return &Server{} }
func (server *Server) Run(context.Context) error { return nil }
`,
	})

	pkg, fset := testutil.LoadPackage(t, dir, "./cmd/api")
	spec, err := Package(pkg, fset)
	if err != nil {
		t.Fatal(err)
	}

	if len(spec.Providers) != 2 {
		t.Fatalf("unexpected provider count: %d", len(spec.Providers))
	}

	if len(spec.Bindings) != 1 {
		t.Fatalf("unexpected binding count: %d", len(spec.Bindings))
	}

	if len(spec.Entry.Inputs) != 2 {
		t.Fatalf("unexpected entry input count: %d", len(spec.Entry.Inputs))
	}

	if len(spec.Entry.Inputs[1].Fields) != 1 || spec.Entry.Inputs[1].Fields[0].Name != "Runner" {
		t.Fatalf("unexpected entry params fields: %+v", spec.Entry.Inputs[1].Fields)
	}

	if len(spec.Lifecycles) != 1 {
		t.Fatalf("unexpected lifecycle count: %d", len(spec.Lifecycles))
	}

	if spec.Lifecycles[0].Kind != model.HookFuncLifecycle {
		t.Fatalf("unexpected lifecycle kind: %s", spec.Lifecycles[0].Kind)
	}

	if spec.Lifecycles[0].OnStart == nil || spec.Lifecycles[0].OnStart.Name != "startAudit" {
		t.Fatalf("unexpected start hook: %+v", spec.Lifecycles[0].OnStart)
	}
}

func TestPackageParsesTypedStartStopAndMultipleParams(t *testing.T) {
	dir := testutil.CreateModule(t, map[string]string{
		"cmd/api/bootstrap.go": `
package main

import (
	"context"

	"github.com/mayahiro/go-bootstrap/bootstrap"
)

type Config struct{}
type Server struct{}

type ServerParams struct {
	bootstrap.In
	Server *Server
}

type ConfigParams struct {
	bootstrap.In
	Config *Config
}

func NewConfig() *Config { return &Config{} }
func NewServer(*Config) *Server { return &Server{} }
func (server *Server) Start(context.Context) error { return nil }
func (server *Server) Stop(context.Context) error { return nil }
func run(context.Context, ServerParams, ConfigParams) error { return nil }

var spec = bootstrap.Server(
	"api",
	bootstrap.Provide(NewConfig, NewServer),
	bootstrap.Lifecycle(
		bootstrap.StartStop((*Server).Start, (*Server).Stop),
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

	if len(spec.Entry.Inputs) != 3 {
		t.Fatalf("unexpected entry input count: %d", len(spec.Entry.Inputs))
	}

	if len(spec.Entry.Inputs[1].Fields) != 1 || len(spec.Entry.Inputs[2].Fields) != 1 {
		t.Fatalf("unexpected entry params: %+v", spec.Entry.Inputs)
	}

	if len(spec.Lifecycles) != 1 {
		t.Fatalf("unexpected lifecycle count: %d", len(spec.Lifecycles))
	}

	if spec.Lifecycles[0].OnStart == nil || spec.Lifecycles[0].OnStart.Name != "Start" {
		t.Fatalf("unexpected start method: %+v", spec.Lifecycles[0].OnStart)
	}

	if spec.Lifecycles[0].OnStop == nil || spec.Lifecycles[0].OnStop.Name != "Stop" {
		t.Fatalf("unexpected stop method: %+v", spec.Lifecycles[0].OnStop)
	}
}

func TestPackageRejectsNestedIn(t *testing.T) {
	dir := testutil.CreateModule(t, map[string]string{
		"cmd/api/bootstrap.go": `
package main

import (
	"context"

	"github.com/mayahiro/go-bootstrap/bootstrap"
)

type Nested struct {
	bootstrap.In
}

type Params struct {
	bootstrap.In
	Nested Nested
}

func run(context.Context, Params) error { return nil }

var spec = bootstrap.Server(
	"api",
	bootstrap.Entry(run),
)
`,
	})

	pkg, fset := testutil.LoadPackage(t, dir, "./cmd/api")
	_, err := Package(pkg, fset)
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "nested bootstrap.In is not supported") {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestPackageRejectsInvalidOverrideContents(t *testing.T) {
	dir := testutil.CreateModule(t, map[string]string{
		"cmd/api/bootstrap.go": `
package main

import (
	"context"

	"github.com/mayahiro/go-bootstrap/bootstrap"
)

type Server struct{}

func NewServer() *Server { return &Server{} }
func run(context.Context, *Server) error { return nil }

var spec = bootstrap.Server(
	"api",
	bootstrap.Override(
		bootstrap.Provide(NewServer),
		bootstrap.Lifecycle(
			bootstrap.StartStop((*Server).Start, (*Server).Stop),
		),
	),
	bootstrap.Entry(run),
)

func (server *Server) Start(context.Context) error { return nil }
func (server *Server) Stop(context.Context) error { return nil }
`,
	})

	pkg, fset := testutil.LoadPackage(t, dir, "./cmd/api")
	_, err := Package(pkg, fset)
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "Lifecycle is not allowed inside Override") {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestPackageRejectsModuleCycle(t *testing.T) {
	dir := testutil.CreateModule(t, map[string]string{
		"cmd/api/bootstrap.go": `
package main

import (
	"context"

	"github.com/mayahiro/go-bootstrap/bootstrap"
)

func run(context.Context) error { return nil }

var moduleA = bootstrap.Module(
	bootstrap.Include(moduleB),
)

var moduleB = bootstrap.Module(
	bootstrap.Include(moduleA),
)

var spec = bootstrap.Server(
	"api",
	bootstrap.Include(moduleA),
	bootstrap.Entry(run),
)
`,
	})

	pkg, fset := testutil.LoadPackage(t, dir, "./cmd/api")
	_, err := Package(pkg, fset)
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "module include cycle detected") {
		t.Fatalf("unexpected error: %s", err)
	}
}
