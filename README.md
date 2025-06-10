<div align="center">

# Tools

[![Go Version](https://img.shields.io/badge/Go-1.24.3-00ADD8?style=flat&logo=go)](https://golang.org/)
[![Go Report Card](https://goreportcard.com/badge/github.com/inference-gateway/tools)](https://goreportcard.com/report/github.com/inference-gateway/tools)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Build Status](https://img.shields.io/github/actions/workflow/status/inference-gateway/tools/ci.yml?branch=main)](https://github.com/inference-gateway/tools/actions)
[![Release](https://img.shields.io/github/v/release/inference-gateway/tools)](https://github.com/inference-gateway/tools/releases)

_This repository contains a collection of shared internal tools used between different projects._

</div>

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
