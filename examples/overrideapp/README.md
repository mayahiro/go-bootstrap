# overrideapp

`overrideapp` is the minimal test override example for `bootstrap.Override(...)`.

It uses a shared base module and two entrypoints:

- `cmd/prod`
  - Uses the production greeter
- `cmd/testapp`
  - Replaces the greeter through `Override`

## Layout

- `cmd/prod/bootstrap.go`
  - Production composition root
- `cmd/testapp/bootstrap.go`
  - Test-oriented composition root with `Override`
- `internal/app`
  - Shared modules and entry function
- `internal/config`
  - Config provider
- `internal/greeter`
  - Shared interface
- `internal/prodgreeter`
  - Production implementation
- `internal/fakegreeter`
  - Override implementation

## Generate

```bash
go generate ./examples/overrideapp/cmd/prod
go generate ./examples/overrideapp/cmd/testapp
```

## Run or Build

```bash
go run ./examples/overrideapp/cmd/prod
go run ./examples/overrideapp/cmd/testapp
go build -o /tmp/overrideapp-prod ./examples/overrideapp/cmd/prod
go build -o /tmp/overrideapp-test ./examples/overrideapp/cmd/testapp
```

Use the package paths, not `main.go` file paths. Each entrypoint depends on the generated `bootstrap_gen.go` file in the same package.

This example is intended to answer:

- how to replace a production provider with a fake
- how to replace a binding together with that provider
- how to keep a shared base module and vary only the final entrypoint composition
