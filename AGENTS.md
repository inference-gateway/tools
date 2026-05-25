# Repository Guidelines

## Project Structure & Module Organization

This repository is a Go module for shared Inference Gateway tools. The current executable is the code generator in `cmd/generator/main.go`. Shared generator interfaces and registry logic live in `codegen/generator.go`; format-specific implementations live under `codegen/jrpc/` and `codegen/openapi/`. Built binaries are written to `bin/`. CI and release automation are in `.github/workflows/`; release rules are in `.releaserc.yaml`.

There are no checked-in test files or static assets. Add tests beside the package they cover using Go's standard `*_test.go` convention.

## Build, Test, and Development Commands

Run commands from the repository root:

```bash
go build -v ./...                         # compile all packages, as CI does
go build -o bin/generator cmd/generator/main.go
task build                                # Taskfile build; currently outputs bin/myapp
task lint                                 # run golangci-lint with timeout
golangci-lint run                         # exact lint command used by CI
go test ./...                             # run all Go tests when tests are present
./bin/generator -list                     # list registered generators
```

The Flox environment pins Go and `golangci-lint`; use `flox activate` for the project toolchain.

## Coding Style & Naming Conventions

Follow `.editorconfig`: Go files use tabs with width 4; Markdown and YAML use 2-space indentation. Use LF endings, final newlines, and trimmed trailing whitespace. Run `gofmt` on Go changes. Keep package names short and lowercase, and name new generator packages after their schema format, for example `codegen/openapi`.

New generators should implement `codegen.Generator`, register themselves in `init()`, and be blank-imported by `cmd/generator/main.go` so registration happens at startup.

## Testing Guidelines

Use the standard Go testing package. Place tests next to implementation files, name files `*_test.go`, and prefer table-driven tests for schema-to-Go generation behavior. For generator changes, cover parsing, type naming, required/optional field handling, enum generation, imports, and error paths. Run `go test ./...`, `golangci-lint run`, and `go build -v ./...` before opening a PR.

## Commit & Pull Request Guidelines

Commits follow Conventional Commits because semantic-release drives changelogs and versions. Use examples such as `feat: add protobuf generator`, `fix(jrpc): handle nested enums`, or `chore(deps): bump golangci-lint`. A `BREAKING CHANGE:` footer triggers a major release.

Pull requests should include a problem statement, implementation summary, test results, and linked issues when applicable. For generated-code changes, include a minimal input schema and describe the output impact.

## Security & Configuration Tips

Do not commit local secrets from `.env` or generated build outputs from `bin/`. Keep dependency changes intentional and verify them with lint and build commands.
