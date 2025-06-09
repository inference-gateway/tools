package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/inference-gateway/tools/codegen"

	"github.com/inference-gateway/tools/codegen/jrpc"
	_ "github.com/inference-gateway/tools/codegen/openapi"
)

func main() {
	var (
		generatorName  = flag.String("generator", "", "Specific generator to use (optional, auto-detected if not specified)")
		packageName    = flag.String("package", "types", "Target Go package name")
		listGens       = flag.Bool("list", false, "List available generators")
		showHelp       = flag.Bool("help", false, "Show detailed help")
		customAcronyms = flag.String("acronyms", "", "JSON object of custom acronyms (e.g., '{\"api\":true,\"jwt\":true}')")
		noComments     = flag.Bool("no-comments", false, "Disable generation of comments from descriptions")
		noFormat       = flag.Bool("no-format", false, "Disable automatic go fmt on output")
	)

	flag.Parse()

	if *showHelp {
		showDetailedHelp()
		return
	}

	if *listGens {
		listGenerators()
		return
	}

	args := flag.Args()
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <schema-file> <output-file>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Use -help for detailed usage information\n")
		os.Exit(1)
	}

	schemaFile := args[0]
	outputFile := args[1]

	var generator codegen.Generator
	var err error

	if *generatorName != "" {
		generator, err = codegen.Get(*generatorName)
		if err != nil {
			log.Fatalf("Generator not found: %v", err)
		}
	} else {
		generators := codegen.GetByFormat(schemaFile)
		if len(generators) == 0 {
			log.Fatalf("No generators found that support file format of %s", schemaFile)
		}
		if len(generators) > 1 {
			var names []string
			for _, g := range generators {
				names = append(names, g.Name())
			}
			log.Printf("Multiple generators support this format: %s. Using '%s'. Use -generator flag to specify.",
				strings.Join(names, ", "), generators[0].Name())
		}
		generator = generators[0]
	}

	if err := generator.ValidateSchema(schemaFile); err != nil {
		log.Fatalf("Schema validation failed: %v", err)
	}

	var options interface{}

	switch generator.Name() {
	case "jsonrpc":
		jrpcOptions := &jrpc.GeneratorOptions{
			PackageName:     *packageName,
			IncludeComments: !*noComments,
			FormatOutput:    !*noFormat,
		}

		if *customAcronyms != "" {
			var acronyms map[string]bool
			if err := json.Unmarshal([]byte(*customAcronyms), &acronyms); err != nil {
				log.Fatalf("Failed to parse custom acronyms JSON: %v", err)
			}
			jrpcOptions.CustomAcronyms = acronyms
		}

		options = &jrpc.Options{GeneratorOptions: jrpcOptions}

	case "openapi":
		openapiOptions := &struct {
			PackageName     string
			IncludeComments bool
			FormatOutput    bool
			GenerateModels  bool
			GenerateClient  bool
		}{
			PackageName:     *packageName,
			IncludeComments: !*noComments,
			FormatOutput:    !*noFormat,
			GenerateModels:  true,
			GenerateClient:  false,
		}

		options = openapiOptions
	}

	config := codegen.GenerateConfig{
		SchemaPath:  schemaFile,
		OutputPath:  outputFile,
		PackageName: *packageName,
		Options:     options,
	}

	if err := generator.Generate(config); err != nil {
		log.Fatalf("Failed to generate code: %v", err)
	}

	fmt.Printf("Successfully generated Go types using '%s' generator in %s\n", generator.Name(), outputFile)
}

func showDetailedHelp() {
	fmt.Printf(`Code Generator Tool

USAGE:
    %s [flags] <schema-file> <output-file>

ARGUMENTS:
    <schema-file>   Path to the input schema file (JSON, YAML, or YML)
    <output-file>   Path where the generated Go code will be written

FLAGS:
    -generator string
        Specific generator to use. If not specified, the tool will auto-detect
        based on the schema file format and content.
        
    -package string
        Target Go package name for the generated code (default: "types")
        
    -acronyms string
        JSON object defining custom acronyms that should be capitalized in 
        generated Go field names. Example: '{"api":true,"jwt":true}'
        
    -no-comments
        Disable generation of Go comments from schema descriptions
        
    -no-format
        Disable automatic 'go fmt' formatting of the output file
        
    -list
        List all available generators and their descriptions
        
    -help
        Show this detailed help message

EXAMPLES:
    # Auto-detect generator and use default settings
    %s schema.json types.go
    
    # Specify generator explicitly and custom package name
    %s -generator jsonrpc -package models schema.yaml models.go
    
    # Use custom acronyms and disable comments
    %s -acronyms '{"api":true,"http":true}' -no-comments schema.json types.go
    
    # List available generators
    %s -list

`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

func listGenerators() {
	fmt.Println("Available Generators:")
	fmt.Println()

	infos := codegen.ListGeneratorInfo()
	if len(infos) == 0 {
		fmt.Println("No generators registered.")
		return
	}

	for _, info := range infos {
		fmt.Printf("  %s\n", info.Name)
		fmt.Printf("    Description: %s\n", info.Description)
		fmt.Printf("    Supported formats: %s\n", strings.Join(info.SupportedFormats, ", "))
		fmt.Println()
	}
}
