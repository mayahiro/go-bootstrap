# simplecli

`simplecli` is a minimal CLI example built with `go-bootstrap`.

It demonstrates `bootstrap.CLI(...)`, grouped entry params through `bootstrap.In`, and a one-sided `bootstrap.HookFunc(...)`.

## Layout

- `cmd/hello/bootstrap.go`
  - Declares the CLI bootstrap DSL
- `cmd/hello/bootstrap_gen.go`
  - Generated startup wiring
- `internal/audit`
  - Audit writer and stop hook
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
