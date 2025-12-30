# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A code generation tool for the Inference Gateway ecosystem that generates Go types from various schema formats (JSON-RPC, JSON Schema, OpenAPI). Built with a plugin-based architecture for easy extensibility.

**Language**: Go 1.25.4

## Essential Commands

### Build and Run
```bash
# Build the generator binary
task build
# or manually:
go build -o bin/generator cmd/generator/main.go

# Run the generator
./bin/generator schema.json types.go
./bin/generator -generator jsonrpc -package models schema.yaml models.go
./bin/generator -list  # List available generators
```

### Linting and Quality
```bash
# Run linting (always do this before committing)
task lint
# or manually:
golangci-lint run --timeout 5m
```

### Testing
```bash
# Run all tests
go test ./...

# Build all packages (validates code)
go build -v ./...
```

### Clean Up
```bash
task clean  # Remove bin/ directory
```

## Architecture

### Plugin-Based Generator System

The codebase uses a registry pattern for code generators:

1. **Core Interface** (`codegen/generator.go`):
   - `Generator` interface defines the contract: `Name()`, `Description()`, `SupportedFormats()`, `Generate()`, `ValidateSchema()`
   - Default global registry accessible via package-level functions
   - Auto-detection of generators by file format

2. **Generator Implementations**:
   - `codegen/jrpc/`: JSON-RPC and JSON Schema generator
   - `codegen/openapi/`: OpenAPI 3.x generator (currently delegates to jrpc)
   - Each generator registers itself via `init()` function

3. **CLI Entry Point** (`cmd/generator/main.go`):
   - Parses command-line flags
   - Handles generator selection (explicit or auto-detect)
   - Passes configuration to selected generator

### Key Design Principles

- **Self-Registration**: Generators register themselves in `init()`, no manual wiring needed
- **Type-Safe Options**: `GenerateConfig.Options` uses `any` for generator-specific configuration
- **Format Detection**: Registry can find generators by file extension
- **Validation First**: All generators validate schemas before code generation

## Adding a New Generator

To add a new code generator:

1. Create new package under `codegen/`:
   ```go
   package mygenerator

   import "github.com/inference-gateway/tools/codegen"

   type Generator struct{}

   func (g *Generator) Name() string { return "mygenerator" }
   func (g *Generator) Description() string { return "Generates from my format" }
   func (g *Generator) SupportedFormats() []string { return []string{".myext"} }
   func (g *Generator) Generate(config codegen.GenerateConfig) error { /* ... */ }
   func (g *Generator) ValidateSchema(schemaPath string) error { /* ... */ }

   func init() {
       if err := codegen.Register(&Generator{}); err != nil {
           panic(err)
       }
   }
   ```

2. Import the new package in `cmd/generator/main.go` (side-effect import for registration):
   ```go
   import _ "github.com/inference-gateway/tools/codegen/mygenerator"
   ```

3. Test with sample schemas

## Code Structure

```
.
├── cmd/generator/main.go       # CLI entry point, flag parsing, generator orchestration
├── codegen/
│   ├── generator.go            # Generator interface, Registry, GenerateConfig
│   ├── jrpc/
│   │   ├── generator.go        # JRPCGenerator registration and validation
│   │   └── jrpc.go             # Core type generation from JSON Schema
│   └── openapi/
│       └── generator.go        # OpenAPIGenerator (delegates to jrpc)
├── bin/                        # Build output (gitignored)
└── Taskfile.yml               # Task automation
```

## Working with JSON-RPC Generator

The `jrpc` generator (`codegen/jrpc/jrpc.go`) is the most complex:

- **Schema Extraction**: Looks for definitions in `definitions`, `$defs`, or `components/schemas`
- **Type Mapping**: Maps JSON Schema types to Go types
- **Enum Handling**: Generates Go constants for enum values (including inline enums)
- **Acronym Support**: Auto-capitalizes common acronyms (ID, URL, API, HTTP, etc.)
- **Comment Generation**: Optionally includes descriptions as Go comments
- **Post-Processing**: Runs `go fmt` on generated code

Key functions:
- `GenerateTypes()`: Main entry point
- `generateTypeFromSchema()`: Recursive type generation
- `handleEnum()`: Enum constant generation
- `sanitizeIdentifier()`: Name transformation with acronym handling

## Commit Convention

This project uses Conventional Commits with semantic-release:

- `feat:` - New feature (triggers minor version)
- `fix:` - Bug fix (triggers patch version)
- `docs:` - Documentation only
- `refactor:` - Code refactoring
- `test:` - Adding/updating tests
- `chore:` - Maintenance tasks
- `ci:` - CI/CD changes

Format: `type(optional-scope): description`

Examples:
```
feat(jsonrpc): add support for oneOf schema
fix(openapi): handle missing components section
docs: update generator usage examples
```

## CI/CD

### Continuous Integration (`.github/workflows/ci.yml`)
- Triggers: Push to main, all PRs
- Steps: Checkout → Setup Go → Install golangci-lint → Lint → Build
- Go version: Read from `go.mod`
- Linter version: v2.7.2

### Release Pipeline (`.github/workflows/release.yml`)
- Manual trigger only
- Uses semantic-release for versioning
- Generates CHANGELOG.md automatically
- Creates GitHub releases

## Important Notes

- **No Tests Yet**: The project currently has no test files. CI only validates linting and building.
- **Flox Environment**: Optional development environment in `.flox/` directory
- **Editor Config**: `.editorconfig` defines formatting (tabs for Go, spaces for YAML/Markdown)
- **Backward Compatibility**: The jrpc generator maintains compatibility with existing A2A type generation

## Troubleshooting

### Build fails with "package not found"
```bash
go mod download
go mod tidy
```

### Generator not found when running `-list`
- Check that the generator package has an `init()` function that calls `codegen.Register()`
- Verify the generator package is imported in `cmd/generator/main.go`

### Generated code has syntax errors
- Check that `go fmt` is being applied post-generation
- Validate schema structure matches expected format
- Test with minimal schema to isolate issues
