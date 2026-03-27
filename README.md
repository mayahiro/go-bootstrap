# go-bootstrap

`go-bootstrap` is a monorepo for a small bootstrap DSL and the code generator built around it.

The root module provides the declarative API, `bootstrapgen` provides the generator CLI, and `examples/simpleapi` shows the intended style in a minimal app.

See [`examples/README.md`](examples/README.md) for the full example list.

## Why

`go-bootstrap` is aimed at the space between:

- hand-written startup wiring
- compile-time graph builders such as Wire
- runtime application frameworks such as Fx

The goal is to keep startup wiring explicit and inspectable while still giving you higher-level app-bootstrap concepts such as:

- composition roots
- reusable modules
- typed lifecycle hooks
- entrypoint params
- test and environment overrides

The generated output is ordinary Go. There is no runtime reflection and no general-purpose runtime container.

## Good Fit

`go-bootstrap` works well when you want:

- explicit constructors and generated startup wiring
- reusable composition for servers and CLIs
- a smaller API surface than a full runtime framework
- plain generated Go that can still be inspected and debugged

It is a worse fit when you want:

- fully dynamic runtime composition
- highly computed or ad-hoc spec assembly
- multiple unrelated bootstrap specs in the same package
- an edit loop with no regeneration step

## Modules

- `github.com/mayahiro/go-bootstrap`
- `github.com/mayahiro/go-bootstrap/bootstrapgen`
- `github.com/mayahiro/go-bootstrap/examples/simpleapi`
- `github.com/mayahiro/go-bootstrap/examples/overrideapp`
- `github.com/mayahiro/go-bootstrap/examples/simplecli`
- `github.com/mayahiro/go-bootstrap/examples/modularapp`

## Quick Start

Install the library and the generator:

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

Use package paths such as `./cmd/api`, not single-file paths such as `./cmd/api/main.go`. The generated `bootstrap_gen.go` file is compiled together with the rest of the package.

For generator-specific CLI details, input expectations, and generator-side constraints, see [`bootstrapgen/README.md`](bootstrapgen/README.md).

## Core Concepts

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
- `Entry`
  - Define the resolved entrypoint for the generated bootstrap function.
- `In`
  - Group entry parameters into readable structs instead of a long parameter list.
- `StartStop`
  - Use for the common typed lifecycle pair on one receiver type.
- `HookFunc`
  - Use for free functions, one-sided hooks, or hooks that are not naturally a typed start/stop pair.
- `Close`
  - Use when a value just needs `Close()` or `Close() error` handling.

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

Override for tests or environment-specific entrypoints:

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

## Constraints

The design is intentionally narrow so the generator can stay predictable, source-located diagnostics can stay clear, and the generated code can remain plain Go.

- The generator assumes one bootstrap spec per package.
- Specs are expected to be written in a generator-friendly style: package-level module or spec declarations, explicit constructor references, and predictable composition.
- `bootstrap.In` is supported for entry parameter structs, not as a general injection feature across the whole DSL.
- Nested `bootstrap.In` is not supported.
- `Override` is intended for provider and binding replacement, not for replacing the whole app shape.
- DSL changes require regeneration.

## How To Structure Packages

- Keep `bootstrap.go`, `main.go`, and the generated `bootstrap_gen.go` in the same entrypoint package.
- Put shared wiring in reusable `bootstrap.Module(...)` values in other packages.
- Use the entrypoint package as the composition root, and assemble shared modules there with `Include`.
- Run and build packages such as `./cmd/api`, not individual files such as `./cmd/api/main.go`.

## Choosing Lifecycle APIs

- Use `StartStop` when one receiver type has the main typed lifecycle pair for the app.
- Use `HookFunc` for free functions, one-sided hooks, or coordination logic that does not naturally belong on the lifecycle target.
- Use `Close` for simple teardown values that only need `Close()` or `Close() error`.

## What Override Is Not

- `Override` is not a second top-level app spec.
- `Override` is not meant to replace `Entry` or `Lifecycle`.
- `Override` is not intended to express highly dynamic runtime composition.
- `Override` is best treated as a narrow way to replace providers or bindings in tests and environment-specific entrypoints.

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
- Dynamic or heavily computed spec assembly
- Hiding constructors behind patterns that are hard for AST/type-based analysis to follow
- Using `Override` as a whole-app replacement mechanism

## Troubleshooting

- `bootstrap spec not found`
  - Check that the selected package contains a package-level `bootstrap.Server(...)` or `bootstrap.CLI(...)` declaration.
- `multiple bootstrap specs found`
  - Split the specs into separate packages or keep only one spec in the selected package.
- `provider not found`
  - Add a matching `Provide` or `Bind`, or move the required module into the final spec with `Include`.
- `StartStop` validation errors
  - Pass method expressions such as `(*Server).Start` and `(*Server).Stop` on the same receiver type.
- `go run ./cmd/api/main.go` fails
  - Run or build the package path such as `./cmd/api` so `bootstrap_gen.go` is compiled with the rest of the package.

## Generated Code

- Generated output is ordinary Go intended to be readable and debuggable.
- Generated files should be treated as build artifacts and not edited manually.
- The DSL is intentionally smaller than a general-purpose DI language so that generated code stays predictable.

## Repository Layout

- `bootstrap/`
  - Declarative DSL
- `bootstrapgen/`
  - Generator module
- `examples/`
  - Runnable examples for common composition patterns

## Local Development

The modules are connected through the repo root `go.work`.

## License

This project is licensed under the MIT License. See [`LICENSE`](LICENSE).
