# bootstrapgen

`go-bootstrapgen` reads declarations from `github.com/mayahiro/go-bootstrap/bootstrap` and generates Go code for startup wiring.

## Current Scope

- Parse constructors from `Provide`
- Resolve interfaces to concrete types through `Bind`
- Flatten reusable `Module` declarations through `Include`
- Apply high-precedence `Override` declarations for test and environment composition
- Resolve dependencies starting from `Entry`
- Expand multiple entry parameter structs that embed `bootstrap.In`
- Render lifecycle handling for `StartStop`, `Close`, and `HookFunc`
- Report diagnostic errors with source locations and dependency paths
- Generate `bootstrap_gen.go`

## Constraints

- Providers must be `func(...) T` or `func(...) (T, error)`
- Entries must be `func(...)` or `func(...) error`
- `bootstrap.In` is currently supported for entry parameter structs only
- `StartStop` requires method expressions on the same receiver type
- `Override` currently supports `Provide`, `Bind`, `Include`, and nested `Override`
- `HookFunc` callbacks must return either nothing or `error`
- The current implementation assumes one bootstrap spec per package

## Composition Notes

- Use `Module` to define reusable base wiring for a bounded area such as `server`, `cli`, or `storage`.
- Use `Include` to assemble the final app spec from those modules.
- Use `Override` to replace providers or bindings in tests and environment-specific entrypoints.
- Use `StartStop` for typed lifecycle methods and `HookFunc` for free functions or one-sided hooks.

## Usage

```bash
go tool bootstrapgen ./cmd/api
go tool bootstrapgen -version
go tool bootstrapgen -h
```

For local development in this monorepo, it is expected to be referenced through the repo root `go.work`.

## Tests

```bash
go test ./...
```
