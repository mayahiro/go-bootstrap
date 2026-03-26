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

## Positioning

`bootstrapgen` is not trying to be a general-purpose DI container or a fully dynamic bootstrap runtime.

Its job is narrower:

- read a small bootstrap DSL
- resolve a dependency graph from explicit constructors
- generate ordinary Go for startup wiring

This makes it a good fit when you want compile-time visibility and app-bootstrap concepts such as entrypoints, lifecycle hooks, modules, and overrides in one place.

It is a worse fit when you need:

- highly dynamic composition
- ad-hoc or heavily computed spec construction
- broad injection features across every function shape
- multiple independent bootstrap specs in one package

## Quick Start

```bash
go get -tool github.com/mayahiro/go-bootstrap/bootstrapgen/cmd/bootstrapgen@latest
go tool bootstrapgen ./cmd/api
go build ./cmd/api
```

## Constraints

- Providers must be `func(...) T` or `func(...) (T, error)`
- Entries must be `func(...)` or `func(...) error`
- `bootstrap.In` is currently supported for entry parameter structs only
- Nested `bootstrap.In` is not supported
- `StartStop` requires method expressions on the same receiver type
- `Override` currently supports `Provide`, `Bind`, `Include`, and nested `Override`
- `Override` does not replace `Entry` or `Lifecycle`
- `HookFunc` callbacks must return either nothing or `error`
- The current implementation assumes one bootstrap spec per package

## Composition Notes

- Use `Module` to define reusable base wiring for a bounded area such as `server`, `cli`, or `storage`.
- Use `Include` to assemble the final app spec from those modules.
- Use `Override` to replace providers or bindings in tests and environment-specific entrypoints.
- Use `StartStop` for typed lifecycle methods and `HookFunc` for free functions or one-sided hooks.
- Keep specs easy for the generator to read: prefer package-level declarations and explicit constructor references over dynamic assembly.

## Supported Input Style

- One bootstrap spec per package
- Package-level `Server(...)` or `CLI(...)`
- Package-level reusable modules
- Explicit function references in `Provide`, `Entry`, `HookFunc`, and `StartStop`
- Straightforward module composition through `Include` and `Override`

## Unsupported or Discouraged Input Style

- Nested `bootstrap.In`
- Dynamically computed specs
- Patterns that depend on runtime state to define the dependency graph
- Treating `Override` as a replacement for `Entry` or `Lifecycle`

## Common Failure Modes

- No bootstrap spec found in the selected package
- More than one bootstrap spec found in the same package
- Missing provider or conflicting providers for a required type
- Invalid `StartStop` method expressions
- Invalid contents inside `Override`

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
