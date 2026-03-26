# simplecli

`simplecli` is a one-shot CLI example built with `go-bootstrap`.

It demonstrates `bootstrap.CLI(...)`, grouped entry params through `bootstrap.In`, and setup/teardown through `bootstrap.Close(...)`.

## Layout

- `cmd/hello/bootstrap.go`
  - Declares the CLI bootstrap DSL
- `cmd/hello/bootstrap_gen.go`
  - Generated startup wiring
- `internal/audit`
  - Closable audit writer
- `internal/command`
  - Command implementation
- `internal/config`
  - Config provider

## Generate

```bash
go generate ./examples/simplecli/cmd/hello
```

## Build

```bash
go build -o /tmp/simplecli ./examples/simplecli/cmd/hello
```
