package jrpc

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// GeneratorOptions contains configuration options for the Go type generator
type GeneratorOptions struct {
	PackageName     string          // Target Go package name (default: "types")
	CustomAcronyms  map[string]bool // Additional acronyms to handle specially
	IncludeComments bool            // Whether to include descriptions as comments (default: true)
	FormatOutput    bool            // Whether to run go fmt on output (default: true)
}

// DefaultAcronyms returns the default set of acronyms that should be capitalized
func DefaultAcronyms() map[string]bool {
	return map[string]bool{
		"id":      true,
		"uri":     true,
		"url":     true,
		"api":     true,
		"html":    true,
		"http":    true,
		"https":   true,
		"json":    true,
		"jsonrpc": true,
		"rpc":     true,
		"mime":    true,
		"sse":     true,
		"uuid":    true,
		"sql":     true,
		"tcp":     true,
		"udp":     true,
		"jwt":     true,
		"oauth":   true,
		"tls":     true,
		"ssl":     true,
		"xml":     true,
		"csv":     true,
		"pdf":     true,
	}
}

// GenerateTypes generates Go types from JSON/YAML schema files
// Supports JSON Schema Draft 4/6/7 and OpenRPC schemas
func GenerateTypes(destination string, schemaPath string, options *GeneratorOptions) error {
	if options == nil {
		options = &GeneratorOptions{
			PackageName:     "types",
			IncludeComments: true,
			FormatOutput:    true,
		}
	}

	// Set defaults
	if options.PackageName == "" {
		options.PackageName = "types"
	}

	// Merge acronyms
	acronyms := DefaultAcronyms()
	for k, v := range options.CustomAcronyms {
		acronyms[k] = v
	}
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	var schema map[string]interface{}

	switch {
	case strings.HasSuffix(schemaPath, ".json"):
		if err := json.Unmarshal(data, &schema); err != nil {
			return fmt.Errorf("failed to parse JSON schema: %w", err)
		}
	case strings.HasSuffix(schemaPath, ".yaml"), strings.HasSuffix(schemaPath, ".yml"):
		if err := yaml.Unmarshal(data, &schema); err != nil {
			return fmt.Errorf("failed to parse YAML schema: %w", err)
		}
	default:
		return fmt.Errorf("unsupported schema format: must be .json, .yaml, or .yml")
	}

	// Extract definitions from various schema structures
	definitions := extractDefinitions(schema)
	if len(definitions) == 0 {
		return fmt.Errorf("schema does not contain any type definitions")
	}

	outputFile, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() {
		if closeErr := outputFile.Close(); closeErr != nil {
			fmt.Printf("Warning: Failed to close output file: %v\n", closeErr)
		}
	}()

	// Analyze types to determine needed imports
	needsTime := false
	for _, definition := range definitions {
		if defMap, ok := definition.(map[string]interface{}); ok {
			if containsTimeType(defMap) {
				needsTime = true
				break
			}
		}
	}

	// Generate header with appropriate imports
	header := fmt.Sprintf(`// Code generated from JSON schema. DO NOT EDIT.
package %s

`, options.PackageName)

	if needsTime {
		header += `import "time"

`
	}

	if _, err := outputFile.WriteString(header); err != nil {
		return fmt.Errorf("failed to write file header: %w", err)
	}

	processedTypes := map[string]bool{}

	typeNames := make([]string, 0, len(definitions))
	for typeName := range definitions {
		typeNames = append(typeNames, typeName)
	}
	sort.Strings(typeNames)

	// First pass: generate enums
	for _, typeName := range typeNames {
		definition := definitions[typeName]

		defMap, ok := definition.(map[string]interface{})
		if !ok {
			continue
		}

		isEnum := false
		var enumValues []interface{}
		if enum, ok := defMap["enum"].([]interface{}); ok && len(enum) > 0 {
			isEnum = true
			enumValues = enum
		}

		if !isEnum {
			continue
		}

		if err := generateEnumType(outputFile, typeName, defMap, enumValues, acronyms, options); err != nil {
			return err
		}

		processedTypes[typeName] = true
	}

	// Second pass: generate other types
	for _, typeName := range typeNames {
		definition := definitions[typeName]

		defMap, ok := definition.(map[string]interface{})
		if !ok {
			continue
		}

		if processedTypes[typeName] {
			continue
		}

		if err := generateComplexType(outputFile, typeName, defMap, definitions, acronyms, options); err != nil {
			return err
		}
	}

	if options.FormatOutput {
		cmd := exec.Command("go", "fmt", destination)
		if err := cmd.Run(); err != nil {
			fmt.Printf("Warning: Failed to format %s: %v\n", destination, err)
		}
	}

	return nil
}

// extractDefinitions extracts type definitions from various schema structures
func extractDefinitions(schema map[string]interface{}) map[string]interface{} {
	definitions := make(map[string]interface{})

	// Try standard JSON Schema locations
	if defs, ok := schema["definitions"].(map[string]interface{}); ok {
		for k, v := range defs {
			definitions[k] = v
		}
	}

	// Try JSON Schema Draft 2019-09+ location
	if defs, ok := schema["$defs"].(map[string]interface{}); ok {
		for k, v := range defs {
			definitions[k] = v
		}
	}

	// Try OpenAPI/Swagger location
	if components, ok := schema["components"].(map[string]interface{}); ok {
		if schemas, ok := components["schemas"].(map[string]interface{}); ok {
			for k, v := range schemas {
				definitions[k] = v
			}
		}
	}

	// Try OpenRPC location
	if components, ok := schema["components"].(map[string]interface{}); ok {
		if schemas, ok := components["schemas"].(map[string]interface{}); ok {
			for k, v := range schemas {
				definitions[k] = v
			}
		}
		if contentDescriptors, ok := components["contentDescriptors"].(map[string]interface{}); ok {
			for k, v := range contentDescriptors {
				definitions[k] = v
			}
		}
	}

	// Try root-level schemas (some custom formats)
	if schemas, ok := schema["schemas"].(map[string]interface{}); ok {
		for k, v := range schemas {
			definitions[k] = v
		}
	}

	return definitions
}

// generateEnumType generates an enum type definition
func generateEnumType(outputFile *os.File, typeName string, defMap map[string]interface{}, enumValues []interface{}, acronyms map[string]bool, options *GeneratorOptions) error {
	description := ""
	if desc, ok := defMap["description"].(string); ok {
		description = desc
	}

	if description != "" && options.IncludeComments {
		formattedDescription := formatDescription(description)
		if _, err := outputFile.WriteString(formattedDescription + "\n"); err != nil {
			return err
		}
	}

	typeStr := "string"
	if t, ok := defMap["type"].(string); ok {
		typeStr = t
	}

	typeDecl := fmt.Sprintf("type %s %s\n\n", typeName, typeStr)
	if _, err := outputFile.WriteString(typeDecl); err != nil {
		return err
	}

	constDecl := fmt.Sprintf("// %s enum values\nconst (\n", typeName)
	if _, err := outputFile.WriteString(constDecl); err != nil {
		return err
	}

	enumStrings := make([]string, 0, len(enumValues))
	for _, val := range enumValues {
		if strVal, ok := val.(string); ok {
			enumStrings = append(enumStrings, strVal)
		}
	}
	sort.Strings(enumStrings)

	for _, val := range enumStrings {
		enumVal := fmt.Sprintf("\t%s%s %s = \"%s\"\n", typeName, convertToGoFieldName(val, acronyms), typeName, val)
		if _, err := outputFile.WriteString(enumVal); err != nil {
			return err
		}
	}

	if _, err := outputFile.WriteString(")\n\n"); err != nil {
		return err
	}

	return nil
}

// generateComplexType generates struct, interface, or other complex type definitions
func generateComplexType(outputFile *os.File, typeName string, defMap map[string]interface{}, definitions map[string]interface{}, acronyms map[string]bool, options *GeneratorOptions) error {
	description := ""
	if desc, ok := defMap["description"].(string); ok {
		description = desc
	}

	if description != "" && options.IncludeComments {
		formattedDescription := formatDescription(description)
		if _, err := outputFile.WriteString(formattedDescription + "\n"); err != nil {
			return err
		}
	}

	// Handle anyOf/oneOf/allOf as interface{}
	if _, hasAnyOf := defMap["anyOf"]; hasAnyOf {
		typeDecl := fmt.Sprintf("type %s interface{}\n\n", typeName)
		if _, err := outputFile.WriteString(typeDecl); err != nil {
			return err
		}
		return nil
	}

	if _, hasOneOf := defMap["oneOf"]; hasOneOf {
		typeDecl := fmt.Sprintf("type %s interface{}\n\n", typeName)
		if _, err := outputFile.WriteString(typeDecl); err != nil {
			return err
		}
		return nil
	}

	if _, hasAllOf := defMap["allOf"]; hasAllOf {
		// For allOf, we could try to merge properties, but for simplicity, use interface{}
		typeDecl := fmt.Sprintf("type %s interface{}\n\n", typeName)
		if _, err := outputFile.WriteString(typeDecl); err != nil {
			return err
		}
		return nil
	}

	// Handle object types (structs)
	structDef := fmt.Sprintf("type %s struct {\n", typeName)
	if _, err := outputFile.WriteString(structDef); err != nil {
		return err
	}

	properties, ok := defMap["properties"].(map[string]interface{})
	if ok {
		propNames := make([]string, 0, len(properties))
		for propName := range properties {
			propNames = append(propNames, propName)
		}
		sort.Strings(propNames)

		// Get required fields
		requiredFields := make(map[string]bool)
		if required, ok := defMap["required"].([]interface{}); ok {
			for _, field := range required {
				if fieldName, ok := field.(string); ok {
					requiredFields[fieldName] = true
				}
			}
		}

		for _, propName := range propNames {
			propDef := properties[propName]
			propMap, ok := propDef.(map[string]interface{})
			if !ok {
				continue
			}

			fieldName := convertToGoFieldName(propName, acronyms)
			propType := determineGoType(propMap, definitions)

			// Add pointer for optional fields (not required and not having default value)
			if !requiredFields[propName] && !hasDefaultValue(propMap) {
				if !strings.HasPrefix(propType, "*") && !strings.HasPrefix(propType, "[]") && !strings.HasPrefix(propType, "map[") {
					propType = "*" + propType
				}
			}

			jsonTag := fmt.Sprintf("`json:\"%s", propName)
			if !requiredFields[propName] {
				jsonTag += ",omitempty"
			}
			jsonTag += "\"`"

			propDefStr := fmt.Sprintf("\t%s %s %s\n", fieldName, propType, jsonTag)
			if _, err := outputFile.WriteString(propDefStr); err != nil {
				return err
			}
		}
	}

	if _, err := outputFile.WriteString("}\n\n"); err != nil {
		return err
	}

	return nil
}

// hasDefaultValue checks if a property has a default value
func hasDefaultValue(propMap map[string]interface{}) bool {
	_, hasDefault := propMap["default"]
	return hasDefault
}

// formatDescription formats a description string as proper Go comments
// with each line prefixed by "// "
func formatDescription(description string) string {
	if description == "" {
		return ""
	}

	lines := strings.Split(description, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			lines[i] = "// " + line
		} else {
			lines[i] = "//"
		}
	}

	return strings.Join(lines, "\n")
}

// convertToGoFieldName converts a JSON property name to a properly capitalized Go field name
func convertToGoFieldName(name string, acronyms map[string]bool) string {
	if name == "" {
		return ""
	}

	// Handle special cases
	if name == "_meta" {
		return "Meta"
	}

	// Remove leading underscores and numbers that would make invalid Go identifiers
	name = strings.TrimLeft(name, "_0123456789")
	if name == "" {
		return "Field"
	}

	// Replace hyphens and other non-alphanumeric characters with underscores
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, " ", "_")

	// Remove any remaining non-alphanumeric characters except underscores
	var cleanName strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			cleanName.WriteRune(r)
		}
	}
	name = cleanName.String()

	// Split on camelCase boundaries and underscores
	var parts []string
	var current strings.Builder

	for i, r := range name {
		if i > 0 && (r >= 'A' && r <= 'Z') && current.Len() > 0 {
			// Check if this is actually a camelCase boundary or just an acronym
			prevRune := rune(name[i-1])
			if prevRune >= 'a' && prevRune <= 'z' {
				parts = append(parts, current.String())
				current.Reset()
			}
		}
		current.WriteRune(r)
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	// Split further on underscores
	var finalParts []string
	for _, part := range parts {
		subParts := strings.Split(part, "_")
		for _, subPart := range subParts {
			if subPart != "" {
				finalParts = append(finalParts, subPart)
			}
		}
	}

	// Capitalize each part appropriately
	for i, part := range finalParts {
		lowerPart := strings.ToLower(part)
		if acronyms[lowerPart] {
			finalParts[i] = strings.ToUpper(lowerPart)
		} else {
			finalParts[i] = cases.Title(language.English).String(lowerPart)
		}
	}

	result := strings.Join(finalParts, "")

	// Ensure the result starts with a capital letter
	if len(result) > 0 && result[0] >= 'a' && result[0] <= 'z' {
		result = strings.ToUpper(string(result[0])) + result[1:]
	}

	// If result is empty or starts with a number, prefix with "Field"
	if result == "" || (len(result) > 0 && result[0] >= '0' && result[0] <= '9') {
		result = "Field" + result
	}

	return result
}

// determineGoType determines the Go type for a JSON schema property
func determineGoType(propMap map[string]interface{}, definitions map[string]interface{}) string {
	// Handle $ref first
	if ref, ok := propMap["$ref"].(string); ok {
		parts := strings.Split(ref, "/")
		refType := parts[len(parts)-1]
		return refType
	}

	// Handle arrays
	if propType, ok := propMap["type"].(string); ok && propType == "array" {
		if items, ok := propMap["items"].(map[string]interface{}); ok {
			itemType := determineGoType(items, definitions)
			return "[]" + itemType
		}
		return "[]interface{}"
	}

	// Handle basic types
	if propType, ok := propMap["type"].(string); ok {
		format := ""
		if fmt, ok := propMap["format"].(string); ok {
			format = fmt
		}

		switch propType {
		case "string":
			switch format {
			case "date-time":
				return "time.Time"
			case "date":
				return "time.Time"
			case "time":
				return "time.Time"
			case "email":
				return "string"
			case "hostname":
				return "string"
			case "ipv4":
				return "string"
			case "ipv6":
				return "string"
			case "uri":
				return "string"
			case "uuid":
				return "string"
			case "byte":
				return "[]byte"
			case "binary":
				return "[]byte"
			default:
				return "string"
			}
		case "integer":
			switch format {
			case "int32":
				return "int32"
			case "int64":
				return "int64"
			default:
				return "int"
			}
		case "number":
			switch format {
			case "float":
				return "float32"
			case "double":
				return "float64"
			default:
				return "float64"
			}
		case "boolean":
			return "bool"
		case "object":
			// Check if it has additionalProperties or properties
			if additionalProps, ok := propMap["additionalProperties"]; ok {
				if additionalPropsMap, ok := additionalProps.(map[string]interface{}); ok {
					valueType := determineGoType(additionalPropsMap, definitions)
					return "map[string]" + valueType
				} else if additionalProps == true {
					return "map[string]interface{}"
				}
			}

			if _, hasProperties := propMap["properties"]; hasProperties {
				// This should probably be its own type, but for now use map
				return "map[string]interface{}"
			}

			return "map[string]interface{}"
		case "null":
			return "interface{}"
		}
	}

	// Handle oneOf/anyOf/allOf
	if oneOf, ok := propMap["oneOf"].([]interface{}); ok && len(oneOf) > 0 {
		return "interface{}"
	}

	if anyOf, ok := propMap["anyOf"].([]interface{}); ok && len(anyOf) > 0 {
		return "interface{}"
	}

	if allOf, ok := propMap["allOf"].([]interface{}); ok && len(allOf) > 0 {
		return "interface{}"
	}

	// Handle const (single value enum)
	if _, ok := propMap["const"]; ok {
		return "interface{}"
	}

	// Handle enum without type
	if enum, ok := propMap["enum"].([]interface{}); ok && len(enum) > 0 {
		// Try to infer type from enum values
		for _, val := range enum {
			switch val.(type) {
			case string:
				return "string"
			case float64:
				return "float64"
			case int:
				return "int"
			case bool:
				return "bool"
			}
		}
		return "interface{}"
	}

	return "interface{}"
}

// GenerateA2ATypes provides backward compatibility with the original function
func GenerateA2ATypes(destination string, schemaPath string) error {
	options := &GeneratorOptions{
		PackageName:     "a2a",
		IncludeComments: true,
		FormatOutput:    true,
		CustomAcronyms: map[string]bool{
			"a2a": true,
		},
	}
	return GenerateTypes(destination, schemaPath, options)
}

// GenerateFromOpenRPC generates Go types from an OpenRPC specification
func GenerateFromOpenRPC(destination string, specPath string, options *GeneratorOptions) error {
	if options == nil {
		options = &GeneratorOptions{
			PackageName:     "jsonrpc",
			IncludeComments: true,
			FormatOutput:    true,
		}
	}

	if options.PackageName == "" {
		options.PackageName = "jsonrpc"
	}

	return GenerateTypes(destination, specPath, options)
}

// GenerateFromJSONSchema generates Go types from a JSON Schema file
func GenerateFromJSONSchema(destination string, schemaPath string, packageName string) error {
	options := &GeneratorOptions{
		PackageName:     packageName,
		IncludeComments: true,
		FormatOutput:    true,
	}

	if packageName == "" {
		options.PackageName = "schema"
	}

	return GenerateTypes(destination, schemaPath, options)
}

// ValidateSchema performs basic validation on the schema structure
func ValidateSchema(schemaPath string) error {
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	var schema map[string]interface{}

	switch {
	case strings.HasSuffix(schemaPath, ".json"):
		if err := json.Unmarshal(data, &schema); err != nil {
			return fmt.Errorf("invalid JSON schema: %w", err)
		}
	case strings.HasSuffix(schemaPath, ".yaml"), strings.HasSuffix(schemaPath, ".yml"):
		if err := yaml.Unmarshal(data, &schema); err != nil {
			return fmt.Errorf("invalid YAML schema: %w", err)
		}
	default:
		return fmt.Errorf("unsupported schema format: must be .json, .yaml, or .yml")
	}

	definitions := extractDefinitions(schema)
	if len(definitions) == 0 {
		return fmt.Errorf("schema does not contain any type definitions")
	}

	return nil
}

// containsTimeType recursively checks if a schema definition contains time-related types
func containsTimeType(defMap map[string]interface{}) bool {
	// Check current level format
	if format, ok := defMap["format"].(string); ok {
		switch format {
		case "date-time", "date", "time":
			return true
		}
	}

	// Check properties
	if properties, ok := defMap["properties"].(map[string]interface{}); ok {
		for _, prop := range properties {
			if propMap, ok := prop.(map[string]interface{}); ok {
				if containsTimeType(propMap) {
					return true
				}
			}
		}
	}

	// Check array items
	if items, ok := defMap["items"].(map[string]interface{}); ok {
		if containsTimeType(items) {
			return true
		}
	}

	// Check anyOf/oneOf/allOf
	for _, key := range []string{"anyOf", "oneOf", "allOf"} {
		if schemas, ok := defMap[key].([]interface{}); ok {
			for _, schema := range schemas {
				if schemaMap, ok := schema.(map[string]interface{}); ok {
					if containsTimeType(schemaMap) {
						return true
					}
				}
			}
		}
	}

	return false
}
