// Package openapi provides a Go code generator for OpenAPI specifications
package openapi

import (
	"fmt"
	"os"
	"strings"

	"github.com/inference-gateway/tools/codegen"
	"github.com/inference-gateway/tools/codegen/jrpc"
)

// OpenAPIGenerator implements the Generator interface for OpenAPI schemas
type OpenAPIGenerator struct{}

// Name returns the unique identifier for this generator
func (g *OpenAPIGenerator) Name() string {
	return "openapi"
}

// Description returns a human-readable description
func (g *OpenAPIGenerator) Description() string {
	return "Generates Go types from OpenAPI 3.x specifications"
}

// SupportedFormats returns the file extensions this generator can process
func (g *OpenAPIGenerator) SupportedFormats() []string {
	return []string{".json", ".yaml", ".yml"}
}

// Options for the OpenAPI generator
type Options struct {
	// PackageName is the target Go package name
	PackageName string

	// IncludeComments determines whether to generate comments from descriptions
	IncludeComments bool

	// FormatOutput determines whether to run go fmt on the output
	FormatOutput bool

	// GenerateModels determines whether to generate model structs
	GenerateModels bool

	// GenerateClient determines whether to generate client code (future feature)
	GenerateClient bool
}

// Generate processes the OpenAPI schema and generates Go code
func (g *OpenAPIGenerator) Generate(config codegen.GenerateConfig) error {
	var options *Options

	if config.Options != nil {
		if opts, ok := config.Options.(*Options); ok {
			options = opts
		}
	}

	if options == nil {
		options = &Options{
			PackageName:     config.PackageName,
			IncludeComments: true,
			FormatOutput:    true,
			GenerateModels:  true,
			GenerateClient:  false,
		}
	}

	if config.PackageName != "" {
		options.PackageName = config.PackageName
	}

	// For now, delegate to the JSONRPC generator since OpenAPI schemas
	// are compatible with JSON Schema for the components/schemas section
	jrpcOptions := &jrpc.GeneratorOptions{
		PackageName:     options.PackageName,
		IncludeComments: options.IncludeComments,
		FormatOutput:    options.FormatOutput,
	}

	return jrpc.GenerateTypes(config.OutputPath, config.SchemaPath, jrpcOptions)
}

// ValidateSchema validates the OpenAPI schema
func (g *OpenAPIGenerator) ValidateSchema(schemaPath string) error {
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	content := string(data)
	if !strings.Contains(content, "openapi") && !strings.Contains(content, "swagger") {
		return fmt.Errorf("file does not appear to be an OpenAPI specification")
	}

	return jrpc.ValidateSchema(schemaPath)
}

// NewOpenAPIGenerator creates a new instance of the OpenAPI generator
func NewOpenAPIGenerator() *OpenAPIGenerator {
	return &OpenAPIGenerator{}
}

// Register automatically registers the OpenAPI generator with the default registry
func init() {
	generator := NewOpenAPIGenerator()
	if err := codegen.Register(generator); err != nil {
		panic(fmt.Sprintf("Failed to register OpenAPI generator: %v", err))
	}
}
