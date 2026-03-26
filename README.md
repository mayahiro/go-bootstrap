# go-bootstrap

`go-bootstrap` is a monorepo that contains a small DSL for declaring Go composition roots and the code generation tool built around it.

The root module provides the declarative API, `bootstrapgen` is the generator module, and `examples/simpleapi` is the example module.

## Layout

- `bootstrap/`
  - Declarative DSL
- `bootstrapgen/`
  - Generator module
- `examples/simpleapi/`
  - Minimal server example

## Modules

- `github.com/mayahiro/go-bootstrap`
- `github.com/mayahiro/go-bootstrap/bootstrapgen`
- `github.com/mayahiro/go-bootstrap/examples/simpleapi`

## Quick Start

```bash
go get github.com/mayahiro/go-bootstrap@latest
go get -tool github.com/mayahiro/go-bootstrap/bootstrapgen/cmd/bootstrapgen@latest
```

Create a package-level spec:

```go
package main

import (
	"context"

	"github.com/mayahiro/go-bootstrap/bootstrap"
)

type Config struct{}
type Server struct{}

func LoadConfig() (*Config, error) { return &Config{}, nil }
func NewServer(*Config) *Server { return &Server{} }
func (server *Server) Start(context.Context) error { return nil }
func (server *Server) Stop(context.Context) error { return nil }
func run(context.Context, *Server) error { return nil }

var spec = bootstrap.Server(
	"api",
	bootstrap.Provide(
		LoadConfig,
		NewServer,
	),
	bootstrap.Lifecycle(
		bootstrap.StartStop((*Server).Start, (*Server).Stop),
	),
	bootstrap.Entry(run),
)
```

Generate and build:

```bash
go tool bootstrapgen ./cmd/api
go build ./cmd/api
```

## Principles

- No runtime reflection
- Generate plain Go code
- Make dependencies explicit through constructors and entry points
- Focus on server and CLI startup wiring

## Positioning

`go-bootstrap` is aimed at the space between hand-written wiring, compile-time DI helpers such as Wire, and runtime application frameworks such as Fx.

- Compared with hand-written DI:
  - It gives you a small declarative layer for composition roots, lifecycle wiring, and test or environment overrides.
- Compared with compile-time graph builders:
  - It stays focused on app bootstrap concerns such as entrypoints, lifecycle hooks, and reusable modules.
- Compared with runtime DI frameworks:
  - It keeps the output as plain generated Go, without runtime reflection or a general-purpose container.

In practice, this library works best when you want:

- Explicit constructors and generated startup wiring
- Reusable composition for servers and CLIs
- A smaller surface area than a full runtime framework
- Plain Go output that can still be inspected and debugged

It is a worse fit when you want:

- Fully dynamic runtime composition
- Arbitrary DSL expressiveness
- Multiple unrelated bootstrap specs living naturally in the same package
- A workflow where every edit should be reflected immediately without regeneration

## When To Use What

- `Provide`
  - Register explicit constructors for concrete values.
- `Bind`
  - Resolve an interface to a concrete implementation type already produced by a provider.
- `Module`
  - Group a bounded area of wiring such as `server`, `cli`, `database`, or `observability`.
- `Include`
  - Assemble the final app spec from reusable modules.
- `Override`
  - Replace providers or bindings in tests or environment-specific entrypoints.
- `StartStop`
  - Use for the common typed lifecycle pair on one receiver type.
- `HookFunc`
  - Use for free functions, one-sided hooks, or hooks that are not naturally a typed start/stop pair.
- `In`
  - Use when an entrypoint is easier to read as grouped params instead of a long parameter list.

## Public API

- `bootstrap.Server`
- `bootstrap.CLI`
- `bootstrap.Provide`
- `bootstrap.Bind`
- `bootstrap.Module`
- `bootstrap.Include`
- `bootstrap.Override`
- `bootstrap.Entry`
- `bootstrap.In`
- `bootstrap.Lifecycle`
- `bootstrap.StartStop`
- `bootstrap.Close`
- `bootstrap.HookFunc`

## Example

```go
type runParams struct {
	bootstrap.In
	Runner httpserver.Runner
}

var serverModule = bootstrap.Module(
	bootstrap.Provide(
		config.Load,
		logger.New,
		httpserver.New,
	),
	bootstrap.Bind(
		(*httpserver.Runner)(nil),
		(*httpserver.Server)(nil),
	),
	bootstrap.Lifecycle(
		bootstrap.StartStop((*httpserver.Server).Start, (*httpserver.Server).Stop),
	),
)

var spec = bootstrap.Server(
	"api",
	bootstrap.Include(serverModule),
	bootstrap.Entry(run),
)
```

## Override Example

```go
var testModule = bootstrap.Module(
	bootstrap.Include(serverModule),
	bootstrap.Override(
		bootstrap.Provide(
			fakelogger.New,
		),
	),
)
```

Use `StartStop` for the common typed start/stop pair on a lifecycle target. Use `HookFunc` when the hook is not a method pair on the same receiver or when only one side is needed.

The DSL is intended to be read by the generator through AST and type information.

That design choice is deliberate. The DSL is intentionally smaller than a general-purpose DI language, and specs are expected to be written in a generator-friendly style: package-level module or spec declarations, explicit constructors, and predictable composition.

## Constraints

- The generator currently assumes one bootstrap spec per package.
- `bootstrap.In` is supported for entry parameter structs, not as a general parameter injection feature across the whole DSL.
- `Override` is intended for provider and binding replacement, not for replacing the whole app shape.
- The development loop includes regeneration after DSL changes.

## Supported Patterns

- Package-level `var spec = bootstrap.Server(...)` or `bootstrap.CLI(...)`
- Package-level `var module = bootstrap.Module(...)`
- Explicit constructor references passed to `Provide`
- Method expressions passed to `StartStop`
- Entry parameter structs that embed `bootstrap.In`
- Module composition through `Include`
- Test or environment replacement through `Override`

## Unsupported or Discouraged Patterns

- Multiple bootstrap specs in one package
- Nested `bootstrap.In`
- Dynamic or heavily computed spec assembly
- Hiding constructors behind patterns that are hard for AST/type-based analysis to follow
- Using `Override` as a whole-app replacement mechanism

## Generated Code

- Generated output is ordinary Go intended to be readable and debuggable.
- Generated files should be treated as build artifacts and not edited manually.
- The DSL is intentionally smaller than a general-purpose DI language so that generated code stays predictable.

## Local Development

The modules are connected through the repo root `go.work`.
