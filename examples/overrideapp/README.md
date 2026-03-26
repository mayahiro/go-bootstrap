# overrideapp

`overrideapp` demonstrates environment-specific composition with `bootstrap.Override(...)`.

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

## Build

```bash
go build -o /tmp/overrideapp-prod ./examples/overrideapp/cmd/prod
go build -o /tmp/overrideapp-test ./examples/overrideapp/cmd/testapp
```
