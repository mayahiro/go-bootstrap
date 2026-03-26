# bootstrapgen

`go-bootstrapgen` reads declarations from `github.com/mayahiro/go-bootstrap/bootstrap` and generates Go code for startup wiring.

## Current Scope

- Parse constructors from `Provide`
- Resolve interfaces to concrete types through `Bind`
- Flatten reusable `Module` declarations through `Include`
- Resolve dependencies starting from `Entry`
- Expand entry parameter structs that embed `bootstrap.In`
- Render lifecycle handling for `StartStop`, `Close`, and `HookFunc`
- Report diagnostic errors with source locations and dependency paths
- Generate `bootstrap_gen.go`

## Constraints

- Providers must be `func(...) T` or `func(...) (T, error)`
- Entries must be `func(...)` or `func(...) error`
- `bootstrap.In` is currently supported for entry parameter structs only
- `HookFunc` callbacks must return either nothing or `error`
- The current implementation assumes one bootstrap spec per package

## Usage

```bash
go tool bootstrapgen ./cmd/api
```

For local development in this monorepo, it is expected to be referenced through the repo root `go.work`.

## Tests

```bash
go test ./...
```
