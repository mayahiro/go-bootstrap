# examples

The `examples` directory contains small runnable applications that show the intended style of `go-bootstrap`.

Generated examples should be run and built by package path, not by pointing `go run` or `go build` at a single `main.go` file. The generated `bootstrap_gen.go` file lives next to `main.go` in the same package, so commands such as `go run ./examples/gracefulhttp/cmd/api` work, while commands that target only `./examples/gracefulhttp/cmd/api/main.go` do not include the generated wiring.

- `simpleapi`
  - Minimal server bootstrap with `Module`, typed `StartStop`, and `In`
- `overrideapp`
  - Test override example for replacing providers and bindings
- `simplecli`
  - One-shot CLI with setup and teardown
- `modularapp`
  - Multi-module app with multiple entry param structs
- `gracefulhttp`
  - HTTP server with graceful shutdown driven by `HookFunc`
