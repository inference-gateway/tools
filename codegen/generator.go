// Package codegen provides a common interface and registry for code generators
package codegen

import (
	"fmt"
	"strings"
)

// Generator defines the interface that all code generators must implement
type Generator interface {
	// Name returns the unique name/identifier for this generator
	Name() string

	// Description returns a human-readable description of what this generator does
	Description() string

	// SupportedFormats returns a list of file extensions this generator can process
	// (e.g., [".json", ".yaml", ".yml"])
	SupportedFormats() []string

	// Generate processes the input schema and generates code
	Generate(config GenerateConfig) error

	// ValidateSchema validates the input schema before generation
	ValidateSchema(schemaPath string) error
}

// GenerateConfig contains all the configuration needed for code generation
type GenerateConfig struct {
	// Input schema file path
	SchemaPath string

	// Output file path
	OutputPath string

	// Target package name
	PackageName string

	// Generator-specific options (can be type-asserted by generators)
	Options interface{}
}

// Registry manages available generators
type Registry struct {
	generators map[string]Generator
}

// NewRegistry creates a new generator registry
func NewRegistry() *Registry {
	return &Registry{
		generators: make(map[string]Generator),
	}
}

// Register adds a generator to the registry
func (r *Registry) Register(generator Generator) error {
	name := generator.Name()
	if name == "" {
		return fmt.Errorf("generator name cannot be empty")
	}

	if _, exists := r.generators[name]; exists {
		return fmt.Errorf("generator with name '%s' already registered", name)
	}

	r.generators[name] = generator
	return nil
}

// Get retrieves a generator by name
func (r *Registry) Get(name string) (Generator, error) {
	generator, exists := r.generators[name]
	if !exists {
		return nil, fmt.Errorf("generator '%s' not found", name)
	}
	return generator, nil
}

// List returns all registered generator names
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.generators))
	for name := range r.generators {
		names = append(names, name)
	}
	return names
}

// GetByFormat finds generators that support the given file format
func (r *Registry) GetByFormat(filePath string) []Generator {
	var matches []Generator

	for _, generator := range r.generators {
		for _, format := range generator.SupportedFormats() {
			if strings.HasSuffix(strings.ToLower(filePath), format) {
				matches = append(matches, generator)
				break
			}
		}
	}

	return matches
}

// GetGeneratorInfo returns information about a specific generator
func (r *Registry) GetGeneratorInfo(name string) (GeneratorInfo, error) {
	generator, err := r.Get(name)
	if err != nil {
		return GeneratorInfo{}, err
	}

	return GeneratorInfo{
		Name:             generator.Name(),
		Description:      generator.Description(),
		SupportedFormats: generator.SupportedFormats(),
	}, nil
}

// ListGeneratorInfo returns information about all registered generators
func (r *Registry) ListGeneratorInfo() []GeneratorInfo {
	var infos []GeneratorInfo

	for _, generator := range r.generators {
		infos = append(infos, GeneratorInfo{
			Name:             generator.Name(),
			Description:      generator.Description(),
			SupportedFormats: generator.SupportedFormats(),
		})
	}

	return infos
}

// GeneratorInfo contains metadata about a generator
type GeneratorInfo struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	SupportedFormats []string `json:"supported_formats"`
}

// Default registry instance
var defaultRegistry = NewRegistry()

// Register adds a generator to the default registry
func Register(generator Generator) error {
	return defaultRegistry.Register(generator)
}

// Get retrieves a generator by name from the default registry
func Get(name string) (Generator, error) {
	return defaultRegistry.Get(name)
}

// List returns all registered generator names from the default registry
func List() []string {
	return defaultRegistry.List()
}

// GetByFormat finds generators that support the given file format from the default registry
func GetByFormat(filePath string) []Generator {
	return defaultRegistry.GetByFormat(filePath)
}

// GetGeneratorInfo returns information about a specific generator from the default registry
func GetGeneratorInfo(name string) (GeneratorInfo, error) {
	return defaultRegistry.GetGeneratorInfo(name)
}

// ListGeneratorInfo returns information about all registered generators from the default registry
func ListGeneratorInfo() []GeneratorInfo {
	return defaultRegistry.ListGeneratorInfo()
}
