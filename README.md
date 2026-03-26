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

## Local Development

The modules are connected through the repo root `go.work`.
