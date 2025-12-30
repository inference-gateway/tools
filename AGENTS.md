# AGENTS.md - Inference Gateway Tools

## Project Overview

This repository contains a collection of shared internal tools used between different projects in the Inference Gateway ecosystem. The primary tool is a flexible, extensible code generation system that supports multiple schema formats and can be easily extended with new generators.

**Main Technologies:**
- **Go 1.25.4** - Primary programming language
- **Task** - Task runner for build automation
- **golangci-lint** - Code linting and quality checks
- **Semantic Release** - Automated versioning and changelog generation
- **Flox** - Development environment management

## Architecture and Structure

### Project Layout
```
.
├── cmd/generator/           # Main CLI application entry point
│   └── main.go             # CLI interface and command handling
├── codegen/                # Core code generation framework
│   ├── generator.go        # Generator interface and registry
│   ├── jrpc/              # JSON-RPC/JSON Schema generator
│   │   ├── generator.go   # JSON-RPC generator implementation
│   │   └── jrpc.go        # Core type generation logic
│   └── openapi/           # OpenAPI generator
│       └── generator.go   # OpenAPI generator implementation
├── .flox/                 # Flox environment configuration
├── .github/workflows/     # CI/CD pipelines
├── bin/                   # Build output directory
└── Taskfile.yml          # Task automation definitions
```

### Core Architecture

The code generation system follows a plugin-based architecture:

1. **Generator Interface** (`codegen/generator.go`): Defines the contract all generators must implement
2. **Registry System**: Central registry for discovering and accessing generators
3. **Generator Implementations**:
   - `jsonrpc`: Generates Go types from JSON-RPC specifications and JSON Schema files
   - `openapi`: Generates Go types from OpenAPI 3.x specifications

### Key Design Patterns
- **Plugin Architecture**: Easy to add new generators by implementing the `Generator` interface
- **Auto-detection**: CLI can auto-detect appropriate generator based on file format and content
- **Extensible Options**: Each generator can have its own configuration options
- **Backward Compatibility**: Maintains compatibility with existing A2A type generation

## Development Environment Setup

### Quick Start Options

#### Option 1: Using Flox Environment (Recommended)
```bash
# Activate the Flox environment
flox activate

# Install dependencies
go get .

# Install infer CLI (automatically installed on activation)
```

#### Option 2: Manual Setup
```bash
# Install Go 1.25.4 or later
# Install Task runner: https://taskfile.dev/installation/
# Install golangci-lint: https://golangci-lint.run/usage/install/

# Clone and build
git clone <repository>
cd tools
go mod download
```

## Key Commands

### Build and Development
```bash
# Show available tasks
task

# Build the application
task build

# Run golangci-lint
task lint

# Clean build artifacts
task clean

# Manual build (alternative)
go build -o bin/generator cmd/generator/main.go
```

### Code Generation Usage
```bash
# Basic usage with auto-detection
./bin/generator schema.json types.go

# List available generators
./bin/generator -list

# Specify generator explicitly
./bin/generator -generator jsonrpc -package models schema.yaml models.go

# Use custom options
./bin/generator -acronyms '{"api":true,"jwt":true}' -no-comments schema.json types.go

# Show detailed help
./bin/generator -help
```

### Testing and Quality
```bash
# Run all tests (if available)
go test ./...

# Run linting
golangci-lint run --timeout 5m

# Build all packages
go build -v ./...
```

## Testing Instructions

### Current Testing Approach
The project currently relies on:
- **CI Pipeline**: Automated linting and building via GitHub Actions
- **Manual Testing**: Schema validation and code generation testing
- **Integration Testing**: Through usage in dependent projects

### Adding Tests
To add tests to the project:

1. **Unit Tests**: Create `*_test.go` files alongside implementation files
2. **Integration Tests**: Test actual code generation with sample schemas
3. **Schema Validation Tests**: Test schema parsing and validation logic

Example test structure:
```go
// codegen/jrpc/generator_test.go
package jrpc

import (
    "testing"
)

func TestGenerateTypes(t *testing.T) {
    // Test cases for type generation
}

func TestValidateSchema(t *testing.T) {
    // Test cases for schema validation
}
```

## Project Conventions and Coding Standards

### Code Style
- **Go Format**: Use `go fmt` for consistent formatting
- **Linting**: Follow `golangci-lint` rules
- **Imports**: Group standard library, third-party, and local imports
- **Naming**: Use descriptive names, follow Go conventions

### File Organization
- **Package Structure**: One directory per logical component
- **Interface Definitions**: In `codegen/generator.go`
- **Implementation**: In respective generator directories
- **CLI**: In `cmd/generator/`

### Commit Conventions
The project uses **Conventional Commits** with semantic release:
- `feat`: New feature (minor release)
- `fix`: Bug fix (patch release)
- `impr`: Improvement (patch release)
- `refactor`: Code refactoring (patch release)
- `perf`: Performance improvement (patch release)
- `docs`: Documentation changes (patch release)
- `style`: Code style changes (patch release)
- `test`: Test-related changes (patch release)
- `build`: Build system changes (patch release)
- `ci`: CI configuration changes (patch release)
- `chore`: Maintenance tasks (patch release, unless scope is `release`)

### Editor Configuration
- **.editorconfig**: Defines consistent formatting rules
- **Go**: Tab indentation, 4 spaces width
- **YAML/Markdown**: Space indentation, 2 spaces width
- **Line Endings**: LF (Unix style)
- **Charset**: UTF-8
- **Trailing Whitespace**: Automatically trimmed

## Important Files and Configurations

### Core Configuration Files

1. **`go.mod`** - Go module definition
   - Module: `github.com/inference-gateway/tools`
   - Go version: 1.25.4
   - Dependencies: `golang.org/x/text`, `gopkg.in/yaml.v3`

2. **`Taskfile.yml`** - Build automation
   - `task lint`: Runs golangci-lint
   - `task build`: Builds the Go application
   - `task clean`: Cleans build artifacts

3. **`.releaserc.yaml`** - Semantic release configuration
   - Automated versioning based on conventional commits
   - Changelog generation
   - GitHub releases

4. **`.flox/env/manifest.toml`** - Development environment
   - Go 1.25.4
   - golangci-lint 2.7.2
   - Automated infer CLI installation

### CI/CD Configuration

1. **`.github/workflows/ci.yml`** - Continuous Integration
   - Runs on push to main and pull requests
   - Sets up Go environment
   - Installs golangci-lint
   - Runs linting and builds

2. **`.github/workflows/release.yml`** - Release Pipeline
   - Manual trigger for releases
   - Uses GitHub App for authentication
   - Runs semantic-release
   - Creates GitHub releases

### Development Configuration

1. **`.flox/env/manifest.toml`** - Flox environment configuration
   - Go 1.25.4
   - golangci-lint 2.7.2
   - Automated infer CLI installation
   - Custom activation hooks for environment setup

## Code Generation System

### Available Generators

#### 1. JSON-RPC Generator (`jsonrpc`)
- **Description**: Generates Go types from JSON-RPC specifications and JSON Schema files
- **Supported Formats**: `.json`, `.yaml`, `.yml`
- **Features**:
  - Extracts definitions from `definitions`, `$defs`, `components/schemas`
  - Handles enums (including inline enums)
  - Supports JSON Schema Draft 4/6/7 and OpenRPC
  - Automatic acronym handling (ID, URL, API, etc.)
  - Optional comments from descriptions
  - Automatic `go fmt` formatting

#### 2. OpenAPI Generator (`openapi`)
- **Description**: Generates Go types from OpenAPI 3.x specifications
- **Supported Formats**: `.json`, `.yaml`, `.yml`
- **Features**:
  - Currently delegates to JSON-RPC generator for schema extraction
  - Validates OpenAPI-specific structure
  - Future: Client code generation

### Extending the System

To add a new generator:

1. **Create a new package** in `codegen/`
2. **Implement the `Generator` interface**:
   ```go
   type MyGenerator struct{}
   
   func (g *MyGenerator) Name() string { return "mygenerator" }
   func (g *MyGenerator) Description() string { return "My generator description" }
   func (g *MyGenerator) SupportedFormats() []string { return []string{".json"} }
   func (g *MyGenerator) Generate(config codegen.GenerateConfig) error { /* implementation */ }
   func (g *MyGenerator) ValidateSchema(schemaPath string) error { /* validation */ }
   ```

3. **Register in `init()` function**:
   ```go
   func init() {
       generator := NewMyGenerator()
       if err := codegen.Register(generator); err != nil {
           panic(fmt.Sprintf("Failed to register MyGenerator: %v", err))
       }
   }
   ```

4. **Add CLI support** in `cmd/generator/main.go` if needed

## Working with AI Agents

### Best Practices for AI Contributions

1. **Always Use Task Commands**: Use `task` commands instead of raw Go commands
2. **Follow Commit Conventions**: Use conventional commit format
3. **Run Linting**: Always run `task lint` before committing
4. **Check Dependencies**: Verify `go.mod` is up to date
5. **Test Generation**: Test with sample schemas when modifying generators

### Common Tasks for AI Agents

#### Adding New Features
1. Create a new branch: `git checkout -b feat/feature-name`
2. Implement the feature following existing patterns
3. Update documentation if needed
4. Run `task lint` and `task build`
5. Commit with conventional commit message
6. Create pull request

#### Fixing Bugs
1. Create a new branch: `git checkout -b fix/bug-description`
2. Reproduce and fix the issue
3. Add test cases if possible
4. Run `task lint` and `task build`
5. Commit with `fix:` prefix
6. Create pull request

#### Updating Dependencies
1. Update `go.mod` with `go get package@version`
2. Run `go mod tidy`
3. Test that everything still works
4. Commit with `chore(deps):` prefix

### Security Considerations
- **Schema Validation**: Always validate input schemas before processing
- **File Operations**: Use safe file paths, avoid directory traversal
- **Code Generation**: Ensure generated code is syntactically valid
- **Dependencies**: Keep dependencies updated, audit for vulnerabilities

## Troubleshooting

### Common Issues

1. **Build Failures**:
   ```bash
   # Clean and rebuild
   task clean
   task build
   
   # Check Go version
   go version
   ```

2. **Linting Errors**:
   ```bash
   # Run with verbose output
   golangci-lint run -v
   
   # Fix auto-fixable issues
   golangci-lint run --fix
   ```

3. **Code Generation Issues**:
   ```bash
   # Validate schema first
   ./bin/generator -generator jsonrpc schema.json output.go
   
   # Check schema format
   cat schema.json | jq .  # for JSON
   cat schema.yaml | yq .  # for YAML
   ```

4. **Flox Environment Issues**:
   - Reactivate environment: `flox activate`
   - Check Flox installation: `flox --version`
   - Verify environment packages: `flox list`

### Getting Help
- Check existing issues on GitHub
- Review CI logs for build errors
- Examine the CHANGELOG for recent changes
- Test with minimal schemas to isolate issues

---

*Last Updated: December 30, 2025*  
*Maintained by: Inference Gateway Team*