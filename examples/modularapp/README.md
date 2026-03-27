# modularapp

`modularapp` is a larger server example that focuses on module boundaries and multiple `bootstrap.In` entry params.

It demonstrates:

- separate modules for config, logging, health, and server wiring
- assembly through `Include`
- multiple entry param structs
- typed `StartStop`

## Layout

- `cmd/api/bootstrap.go`
  - Declares the final server composition root
- `cmd/api/bootstrap_gen.go`
  - Generated startup wiring
- `internal/app`
  - Shared modules and entry function
- `internal/config`
  - Config provider
- `internal/logger`
  - Logger provider
- `internal/health`
  - Health reporter module
- `internal/httpserver`
  - Server provider and lifecycle target

## Generate

```bash
go generate ./examples/modularapp/cmd/api
```

## Run or Build

```bash
go run ./examples/modularapp/cmd/api
go build -o /tmp/modularapp ./examples/modularapp/cmd/api
```

Use the package path, not `./examples/modularapp/cmd/api/main.go`. The generated `bootstrap_gen.go` file lives in the same package and must be compiled together with it.
