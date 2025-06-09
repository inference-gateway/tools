# Tools

This repository contains a collection of shared internal tools used between different projects.

## Code Generator

A flexible, extensible code generation tool that supports multiple schema formats and can be easily extended with new generators.

### Available Generators

- **jsonrpc**: Generates Go types from JSON-RPC specifications and JSON Schema files
- **openapi**: Generates Go types from OpenAPI 3.x specifications

### Usage

```bash
# Basic usage with auto-detection
./generator schema.json types.go

# List available generators
./generator -list

# Specify generator explicitly
./generator -generator jsonrpc -package models schema.yaml models.go

# Use custom options
./generator -acronyms '{"api":true,"jwt":true}' -no-comments schema.json types.go

# Show detailed help
./generator -help
```

### Building

```bash
go build -o bin/generator cmd/generator/main.go
```
