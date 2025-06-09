// Package jrpc provides a Go code generator for JSON-RPC and JSON Schema specifications
package jrpc

import (
	"fmt"

	"github.com/inference-gateway/tools/codegen"
)

// JSONRPCGenerator implements the Generator interface for JSON-RPC schemas
type JSONRPCGenerator struct{}

// Name returns the unique identifier for this generator
func (g *JSONRPCGenerator) Name() string {
	return "jsonrpc"
}

// Description returns a human-readable description
func (g *JSONRPCGenerator) Description() string {
	return "Generates Go types from JSON-RPC specifications and JSON Schema files"
}

// SupportedFormats returns the file extensions this generator can process
func (g *JSONRPCGenerator) SupportedFormats() []string {
	return []string{".json", ".yaml", ".yml"}
}

// Options for the JSON-RPC generator
type Options struct {
	*GeneratorOptions
}

// Generate processes the schema and generates Go code
func (g *JSONRPCGenerator) Generate(config codegen.GenerateConfig) error {
	var options *GeneratorOptions

	if config.Options != nil {
		if opts, ok := config.Options.(*Options); ok && opts.GeneratorOptions != nil {
			options = opts.GeneratorOptions
		} else if opts, ok := config.Options.(*GeneratorOptions); ok {
			options = opts
		}
	}

	if options == nil {
		options = &GeneratorOptions{
			PackageName:     config.PackageName,
			IncludeComments: true,
			FormatOutput:    true,
		}
	}

	if config.PackageName != "" {
		options.PackageName = config.PackageName
	}

	return GenerateTypes(config.OutputPath, config.SchemaPath, options)
}

// ValidateSchema validates the input schema
func (g *JSONRPCGenerator) ValidateSchema(schemaPath string) error {
	return ValidateSchema(schemaPath)
}

// NewJSONRPCGenerator creates a new instance of the JSON-RPC generator
func NewJSONRPCGenerator() *JSONRPCGenerator {
	return &JSONRPCGenerator{}
}

// Register automatically registers the JSON-RPC generator with the default registry
func init() {
	generator := NewJSONRPCGenerator()
	if err := codegen.Register(generator); err != nil {
		panic(fmt.Sprintf("Failed to register JSON-RPC generator: %v", err))
	}
}
