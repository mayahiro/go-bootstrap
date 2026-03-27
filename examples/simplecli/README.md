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

## Run or Build

```bash
go run ./examples/simplecli/cmd/hello
go build -o /tmp/simplecli ./examples/simplecli/cmd/hello
```

Use the package path, not `./examples/simplecli/cmd/hello/main.go`. The generated `bootstrap_gen.go` file is part of the same package.
