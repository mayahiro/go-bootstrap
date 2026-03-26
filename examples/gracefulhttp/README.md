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

## Build

```bash
go build -o /tmp/gracefulhttp ./examples/gracefulhttp/cmd/api
```
