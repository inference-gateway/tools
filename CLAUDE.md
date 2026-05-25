# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`inference-gateway/tools` — shared internal tooling for the Inference Gateway ecosystem. Currently the only tool is a code generator (`cmd/generator`) that produces Go types from JSON Schema, JSON-RPC / OpenRPC, and OpenAPI 3.x specs.

## Commands

```bash
# Build (binary used by the README and by users)
go build -o bin/generator cmd/generator/main.go

# Build via Taskfile (note: outputs bin/myapp, not bin/generator — keep this in mind when invoking)
task build

# Lint
task lint            # golangci-lint run --timeout 5m
golangci-lint run    # CI invocation

# Run the generator
./bin/generator [flags] <schema-file> <output-file>
./bin/generator -list                                  # list registered generators
./bin/generator -help                                  # detailed flag docs
./bin/generator -generator jsonrpc -package models schema.yaml models.go
```

There are no Go tests in this repo (no `_test.go` files); CI only runs `golangci-lint run` and `go build -v ./...`.

The Flox environment (`.flox/manifest.toml`) pins Go and `golangci-lint` versions. `flox activate` gets you a matching shell; nothing in the build assumes Flox is active.

## Architecture

The generator is built around a **plugin registry**. Each format-specific generator implements the `codegen.Generator` interface and self-registers from `init()`. The CLI doesn't know about specific generators — it asks the registry.

```
cmd/generator/main.go
    └─ imports codegen, codegen/jrpc, and _ "codegen/openapi" (blank import → triggers init/register)
        └─ codegen.Registry           (codegen/generator.go)
            ├─ jrpc.JSONRPCGenerator  (codegen/jrpc/generator.go)
            └─ openapi.OpenAPIGenerator (codegen/openapi/generator.go)
                └─ delegates Generate() to jrpc.GenerateTypes — OpenAPI shares
                   JSON-Schema-compatible types in components/schemas, so the
                   openapi package currently only adds an OpenAPI-specific
                   ValidateSchema and otherwise reuses jrpc.
```

To add a new generator: create `codegen/<name>/`, implement `codegen.Generator`, register in `init()`, and add a blank-import line in `cmd/generator/main.go` so the init runs.

### The actual generation logic (codegen/jrpc/jrpc.go)

`GenerateTypes` is where everything happens — both `jrpc` and `openapi` end up calling it. Things worth knowing before editing it:

- **Definition extraction** (`extractDefinitions`) reads from `definitions`, `$defs`, `components.schemas`, `components.contentDescriptors`, and `schemas` — one function handles JSON Schema, OpenAPI, and OpenRPC inputs.
- **Inline enums** (enums declared inline inside struct properties) are hoisted into named Go types. The name is derived from the common prefix of the enum values (`TASK_STATE_RUNNING`, `TASK_STATE_DONE` → `TaskState`); falls back to the property name if there's no meaningful prefix. See `extractInlineEnums` and `deriveEnumTypeName`.
- **Pointer rules**: optional fields (not in `required` and without a `default`) are pointer-wrapped, except slices and maps which stay as-is.
- **Field naming** (`convertToGoFieldName`) splits on `_`, `-`, `.`, ` ` and camelCase boundaries, then re-casing each part. Acronyms in the `DefaultAcronyms()` set (or user-supplied via `-acronyms`) are upper-cased entirely (`api` → `API`). The special case `_meta` → `Meta` is hardcoded.
- **`time.Time` import** is only emitted if some definition has a `date-time`/`date`/`time` format (see `containsTimeType`); otherwise the generated file has no imports.
- **`go fmt`** runs on the output by default (disable with `-no-format`). The exec runs after the file is written.
- Composite schemas (`oneOf`/`anyOf`/`allOf`) and bare `const` currently generate as `any` — there is no discriminated-union codegen.

## Commits and releases

Releases use semantic-release with the **Conventional Commits** preset (`.releaserc.yaml`). Commit type determines the version bump and changelog section:

- `feat:` → minor • `fix:` / `refactor:` / `perf:` / `chore:` / `docs:` / `ci:` / etc. → patch
- `chore(release):` is excluded from triggering a release
- A `BREAKING CHANGE:` footer triggers a major bump

The Release workflow is `workflow_dispatch` only — releases are cut manually from the Actions tab, not on push.
