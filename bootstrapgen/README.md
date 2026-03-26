# bootstrapgen

`bootstrapgen` reads declarations from `github.com/mayahiro/go-bootstrap/bootstrap` and generates ordinary Go for startup wiring.

The root README explains the overall library positioning and DSL concepts. This README focuses on the generator itself: CLI usage, accepted input style, and common failure modes.

## What It Does

- load a package
- find a bootstrap spec
- resolve providers, bindings, entrypoints, lifecycle hooks, modules, and overrides
- generate `bootstrap_gen.go`
- report source-located diagnostics when the spec cannot be resolved

## Quick Start

```bash
go get -tool github.com/mayahiro/go-bootstrap/bootstrapgen/cmd/bootstrapgen@latest
go tool bootstrapgen ./cmd/api
go build ./cmd/api
```

## Usage

```bash
go tool bootstrapgen ./cmd/api
go tool bootstrapgen -o custom_bootstrap.go ./cmd/api
go tool bootstrapgen -version
go tool bootstrapgen -h
```

## Expected Input Style

The generator works best when specs are easy to resolve from AST and type information.

- one bootstrap spec per package
- package-level `Server(...)` or `CLI(...)`
- package-level reusable modules
- explicit function references in `Provide`, `Entry`, `HookFunc`, and `StartStop`
- straightforward module composition through `Include` and `Override`

## Supported Scope

- `Provide`
- `Bind`
- `Module`
- `Include`
- `Override`
- `Entry`
- `In` on entry parameter structs
- `StartStop`
- `Close`
- `HookFunc`

## Constraints

- Providers must be `func(...) T` or `func(...) (T, error)`.
- Entries must be `func(...)` or `func(...) error`.
- `bootstrap.In` is supported for entry parameter structs only.
- Nested `bootstrap.In` is not supported.
- `StartStop` requires method expressions on the same receiver type.
- `Override` currently supports `Provide`, `Bind`, `Include`, and nested `Override`.
- `Override` does not replace `Entry` or `Lifecycle`.
- `HookFunc` callbacks must return either nothing or `error`.
- The generator assumes one bootstrap spec per package.

## Common Failure Modes

- No bootstrap spec found in the selected package
- More than one bootstrap spec found in the same package
- Missing provider for a required type
- Conflicting providers for the same required type
- Invalid `StartStop` method expressions
- Invalid contents inside `Override`
- Specs assembled in a style that is hard for AST/type-based analysis to follow

## Output

- The generated file defaults to `bootstrap_gen.go`.
- The generated code is plain Go and is meant to be readable.
- Generated files are build artifacts and should not be edited manually.

## Local Development

In this monorepo, local development is expected to go through the repo root `go.work`.

## Tests

```bash
go test ./...
```
