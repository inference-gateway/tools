package main

import (
	"fmt"
	"log"
	"os"

	"github.com/inference-gateway/tools/codegen/jrpc"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <schema-file> <output-file> [package-name]\n", os.Args[0])
		os.Exit(1)
	}

	schemaFile := os.Args[1]
	outputFile := os.Args[2]
	packageName := "types"

	if len(os.Args) > 3 {
		packageName = os.Args[3]
	}

	if err := jrpc.ValidateSchema(schemaFile); err != nil {
		log.Fatalf("Schema validation failed: %v", err)
	}

	options := &jrpc.GeneratorOptions{
		PackageName:     packageName,
		IncludeComments: true,
		FormatOutput:    true,
		CustomAcronyms: map[string]bool{
			"api": true,
			"jwt": true,
		},
	}

	if err := jrpc.GenerateTypes(outputFile, schemaFile, options); err != nil {
		log.Fatalf("Failed to generate types: %v", err)
	}

	fmt.Printf("Successfully generated Go types in %s\n", outputFile)
}
