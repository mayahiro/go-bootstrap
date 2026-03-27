# gracefulhttp

`gracefulhttp` is an HTTP server example with graceful shutdown.

It demonstrates:

- typed `bootstrap.StartStop(...)` for server lifecycle
- `bootstrap.HookFunc(...)` for background signal handling
- an entrypoint that waits for shutdown coordination

## Layout

- `cmd/api/bootstrap.go`
  - Declares the server bootstrap DSL
- `cmd/api/bootstrap_gen.go`
  - Generated startup wiring
- `internal/app`
  - Application entry and shutdown coordination
- `internal/config`
  - Config provider
- `internal/httpserver`
  - HTTP server provider and lifecycle target
- `internal/logger`
  - Logger provider

## Generate

```bash
go generate ./examples/gracefulhttp/cmd/api
```

## Run or Build

Run or build the package path, not a single file path. `bootstrap_gen.go` is generated into `cmd/api`, so it is compiled together with `main.go` only when the package is selected as a whole.

```bash
go run ./examples/gracefulhttp/cmd/api
go build -o /tmp/gracefulhttp ./examples/gracefulhttp/cmd/api
```

This does not work the same way:

```bash
go run ./examples/gracefulhttp/cmd/api/main.go
go build ./examples/gracefulhttp/cmd/api/main.go
```

## Build
Use the package path command above.
