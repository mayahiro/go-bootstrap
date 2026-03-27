# simpleapi

`simpleapi` is a minimal server example built with `go-bootstrap` and `bootstrapgen`.

It demonstrates reusable `bootstrap.Module(...)` composition, typed `bootstrap.StartStop(...)`, and an entry parameter struct that embeds `bootstrap.In`.

## Layout

- `cmd/api/bootstrap.go`
  - Declares the bootstrap DSL
- `cmd/api/bootstrap_gen.go`
  - Generated startup wiring
- `internal/config`
  - Config provider
- `internal/logger`
  - Logger provider
- `internal/httpserver`
  - Server provider and lifecycle target

## Prerequisites

This example is intended to be used through the monorepo root `go.work`. Its `go.mod` also keeps `replace` directives for local development before any tags are published.

## Generate

```bash
go generate ./examples/simpleapi/cmd/api
```

## Run or Build

```bash
go run ./examples/simpleapi/cmd/api
go build -o /tmp/simpleapi ./examples/simpleapi/cmd/api
```

Use the package path, not `./examples/simpleapi/cmd/api/main.go`. The generated `bootstrap_gen.go` file lives in the same package and is only compiled when the package is selected as a whole.
