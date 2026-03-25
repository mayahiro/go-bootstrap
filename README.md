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

## Principles

- No runtime reflection
- Generate plain Go code
- Make dependencies explicit through constructors and entry points
- Focus on server and CLI startup wiring

## Public API

- `bootstrap.Server`
- `bootstrap.CLI`
- `bootstrap.Provide`
- `bootstrap.Bind`
- `bootstrap.Entry`
- `bootstrap.Lifecycle`
- `bootstrap.StartStop`
- `bootstrap.Close`

## Example

```go
var spec = bootstrap.Server(
	"api",
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
		bootstrap.StartStop((*httpserver.Server)(nil), "Start", "Stop"),
	),
	bootstrap.Entry(run),
)
```

The DSL is intended to be read by the generator through AST and type information.

## Local Development

The modules are connected through the repo root `go.work`.
