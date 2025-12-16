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

	if options.PackageName == "" {
		options.PackageName = "types"
	}

	acronyms := DefaultAcronyms()
	for k, v := range options.CustomAcronyms {
		acronyms[k] = v
	}
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	var schema map[string]any

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

	needsTime := false
	for _, definition := range definitions {
		if defMap, ok := definition.(map[string]any); ok {
			if containsTimeType(defMap) {
				needsTime = true
				break
			}
		}
	}

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

	inlineEnums := extractInlineEnums(definitions, acronyms)

	inlineEnumNames := make([]string, 0, len(inlineEnums))
	for enumName := range inlineEnums {
		inlineEnumNames = append(inlineEnumNames, enumName)
	}
	sort.Strings(inlineEnumNames)

	for _, enumName := range inlineEnumNames {
		enumDef := inlineEnums[enumName]
		if err := generateEnumType(outputFile, enumName, enumDef.typeInfo, enumDef.values, acronyms, options); err != nil {
			return err
		}
		processedTypes[enumName] = true
	}

	typeNames := make([]string, 0, len(definitions))
	for typeName := range definitions {
		typeNames = append(typeNames, typeName)
	}
	sort.Strings(typeNames)

	for _, typeName := range typeNames {
		definition := definitions[typeName]

		defMap, ok := definition.(map[string]any)
		if !ok {
			continue
		}

		isEnum := false
		var enumValues []any
		if enum, ok := defMap["enum"].([]any); ok && len(enum) > 0 {
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

	for _, typeName := range typeNames {
		definition := definitions[typeName]

		defMap, ok := definition.(map[string]any)
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

// inlineEnumDef holds information about an inline enum extracted from a struct property
type inlineEnumDef struct {
	values   []any
	typeInfo map[string]any
}

// extractInlineEnums scans all definitions for inline enums in struct properties
// and extracts them as separate enum types
func extractInlineEnums(definitions map[string]any, acronyms map[string]bool) map[string]inlineEnumDef {
	inlineEnums := make(map[string]inlineEnumDef)

	for _, definition := range definitions {
		defMap, ok := definition.(map[string]any)
		if !ok {
			continue
		}

		properties, ok := defMap["properties"].(map[string]any)
		if !ok {
			continue
		}

		for propName, propDef := range properties {
			propMap, ok := propDef.(map[string]any)
			if !ok {
				continue
			}

			if enumValues, ok := propMap["enum"].([]any); ok && len(enumValues) > 0 {
				enumTypeName := deriveEnumTypeName(enumValues, propName, acronyms)

				inlineEnums[enumTypeName] = inlineEnumDef{
					values: enumValues,
					typeInfo: map[string]any{
						"description": propMap["description"],
						"type":        "string",
					},
				}
			}
		}
	}

	return inlineEnums
}

// deriveEnumTypeName derives a meaningful enum type name from enum values or property name
// It tries to extract a common prefix from enum values (e.g., "TASK_STATE_XXX" -> "TaskState")
// If no common prefix is found, it uses the property name
func deriveEnumTypeName(enumValues []any, propName string, acronyms map[string]bool) string {
	var stringValues []string
	for _, val := range enumValues {
		if strVal, ok := val.(string); ok {
			stringValues = append(stringValues, strVal)
		}
	}

	if len(stringValues) == 0 {
		return convertToGoFieldName(propName, acronyms)
	}

	commonPrefix := findCommonPrefix(stringValues)
	if commonPrefix != "" {
		commonPrefix = strings.TrimSuffix(commonPrefix, "_")
		typeName := convertToGoFieldName(commonPrefix, acronyms)
		if typeName != "" && typeName != "Field" {
			return typeName
		}
	}

	return convertToGoFieldName(propName, acronyms)
}

// findCommonPrefix finds the common prefix of all strings
// Returns empty string if there's no meaningful common prefix
func findCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}

	prefix := strs[0]

	for _, str := range strs[1:] {
		for !strings.HasPrefix(str, prefix) {
			prefix = prefix[:len(prefix)-1]
			if prefix == "" {
				return ""
			}
		}
	}

	if len(prefix) > 2 && strings.HasSuffix(prefix, "_") {
		return prefix
	}

	for _, str := range strs {
		if len(str) > len(prefix) && str[len(prefix)] == '_' {
			return prefix + "_"
		}
	}

	return ""
}

// extractDefinitions extracts type definitions from various schema structures
func extractDefinitions(schema map[string]any) map[string]any {
	definitions := make(map[string]any)

	if defs, ok := schema["definitions"].(map[string]any); ok {
		for k, v := range defs {
			definitions[k] = v
		}
	}

	if defs, ok := schema["$defs"].(map[string]any); ok {
		for k, v := range defs {
			definitions[k] = v
		}
	}

	if components, ok := schema["components"].(map[string]any); ok {
		if schemas, ok := components["schemas"].(map[string]any); ok {
			for k, v := range schemas {
				definitions[k] = v
			}
		}
	}

	if components, ok := schema["components"].(map[string]any); ok {
		if schemas, ok := components["schemas"].(map[string]any); ok {
			for k, v := range schemas {
				definitions[k] = v
			}
		}
		if contentDescriptors, ok := components["contentDescriptors"].(map[string]any); ok {
			for k, v := range contentDescriptors {
				definitions[k] = v
			}
		}
	}

	if schemas, ok := schema["schemas"].(map[string]any); ok {
		for k, v := range schemas {
			definitions[k] = v
		}
	}

	return definitions
}

// generateEnumType generates an enum type definition
func generateEnumType(outputFile *os.File, typeName string, defMap map[string]any, enumValues []any, acronyms map[string]bool, options *GeneratorOptions) error {
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

	commonPrefix := findCommonPrefix(enumStrings)
	if commonPrefix != "" {
		commonPrefix = strings.TrimSuffix(commonPrefix, "_")
	}

	for _, val := range enumStrings {
		constName := val
		if commonPrefix != "" && strings.HasPrefix(val, commonPrefix+"_") {
			constName = strings.TrimPrefix(val, commonPrefix+"_")
		}

		enumVal := fmt.Sprintf("\t%s%s %s = \"%s\"\n", typeName, convertToGoFieldName(constName, acronyms), typeName, val)
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
func generateComplexType(outputFile *os.File, typeName string, defMap map[string]any, definitions map[string]any, acronyms map[string]bool, options *GeneratorOptions) error {
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

	if _, hasType := defMap["type"].(string); hasType {
		if _, hasProperties := defMap["properties"]; !hasProperties {
			goType := determineGoType(defMap, definitions)
			typeDecl := fmt.Sprintf("type %s = %s\n\n", typeName, goType)
			if _, err := outputFile.WriteString(typeDecl); err != nil {
				return err
			}
			return nil
		}
	}

	if _, hasAnyOf := defMap["anyOf"]; hasAnyOf {
		typeDecl := fmt.Sprintf("type %s any\n\n", typeName)
		if _, err := outputFile.WriteString(typeDecl); err != nil {
			return err
		}
		return nil
	}

	if _, hasOneOf := defMap["oneOf"]; hasOneOf {
		typeDecl := fmt.Sprintf("type %s any\n\n", typeName)
		if _, err := outputFile.WriteString(typeDecl); err != nil {
			return err
		}
		return nil
	}

	if _, hasAllOf := defMap["allOf"]; hasAllOf {
		typeDecl := fmt.Sprintf("type %s any\n\n", typeName)
		if _, err := outputFile.WriteString(typeDecl); err != nil {
			return err
		}
		return nil
	}

	structDef := fmt.Sprintf("type %s struct {\n", typeName)
	if _, err := outputFile.WriteString(structDef); err != nil {
		return err
	}

	properties, ok := defMap["properties"].(map[string]any)
	if ok {
		propNames := make([]string, 0, len(properties))
		for propName := range properties {
			propNames = append(propNames, propName)
		}
		sort.Strings(propNames)

		requiredFields := make(map[string]bool)
		if required, ok := defMap["required"].([]any); ok {
			for _, field := range required {
				if fieldName, ok := field.(string); ok {
					requiredFields[fieldName] = true
				}
			}
		}

		for _, propName := range propNames {
			propDef := properties[propName]
			propMap, ok := propDef.(map[string]any)
			if !ok {
				continue
			}

			fieldName := convertToGoFieldName(propName, acronyms)

			var propType string
			if enumValues, hasEnum := propMap["enum"].([]any); hasEnum && len(enumValues) > 0 {
				propType = deriveEnumTypeName(enumValues, propName, acronyms)
			} else {
				propType = determineGoType(propMap, definitions)
			}

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
func hasDefaultValue(propMap map[string]any) bool {
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

	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, " ", "_")

	var cleanName strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			cleanName.WriteRune(r)
		}
	}
	name = cleanName.String()

	var parts []string
	var current strings.Builder

	for i, r := range name {
		if i > 0 && (r >= 'A' && r <= 'Z') && current.Len() > 0 {
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

	var finalParts []string
	for _, part := range parts {
		subParts := strings.Split(part, "_")
		for _, subPart := range subParts {
			if subPart != "" {
				finalParts = append(finalParts, subPart)
			}
		}
	}

	for i, part := range finalParts {
		lowerPart := strings.ToLower(part)
		if acronyms[lowerPart] {
			finalParts[i] = strings.ToUpper(lowerPart)
		} else {
			finalParts[i] = cases.Title(language.English).String(lowerPart)
		}
	}

	result := strings.Join(finalParts, "")

	if len(result) > 0 && result[0] >= 'a' && result[0] <= 'z' {
		result = strings.ToUpper(string(result[0])) + result[1:]
	}

	if result == "" || (len(result) > 0 && result[0] >= '0' && result[0] <= '9') {
		result = "Field" + result
	}

	return result
}

// determineGoType determines the Go type for a JSON schema property
func determineGoType(propMap map[string]any, definitions map[string]any) string {
	if ref, ok := propMap["$ref"].(string); ok {
		parts := strings.Split(ref, "/")
		refType := parts[len(parts)-1]
		return refType
	}

	if propType, ok := propMap["type"].(string); ok && propType == "array" {
		if items, ok := propMap["items"].(map[string]any); ok {
			itemType := determineGoType(items, definitions)
			return "[]" + itemType
		}
		return "[]any"
	}

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
			if additionalProps, ok := propMap["additionalProperties"]; ok {
				if additionalPropsMap, ok := additionalProps.(map[string]any); ok {
					valueType := determineGoType(additionalPropsMap, definitions)
					return "map[string]" + valueType
				} else if additionalProps == true {
					return "map[string]any"
				}
			}

			if _, hasProperties := propMap["properties"]; hasProperties {
				return "map[string]any"
			}

			return "map[string]any"
		case "null":
			return "any"
		}
	}

	if oneOf, ok := propMap["oneOf"].([]any); ok && len(oneOf) > 0 {
		return "any"
	}

	if anyOf, ok := propMap["anyOf"].([]any); ok && len(anyOf) > 0 {
		return "any"
	}

	if allOf, ok := propMap["allOf"].([]any); ok && len(allOf) > 0 {
		return "any"
	}

	if _, ok := propMap["const"]; ok {
		return "any"
	}

	// Handle enum without type
	if enum, ok := propMap["enum"].([]any); ok && len(enum) > 0 {
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
		return "any"
	}

	return "any"
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

	var schema map[string]any

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
func containsTimeType(defMap map[string]any) bool {
	if format, ok := defMap["format"].(string); ok {
		switch format {
		case "date-time", "date", "time":
			return true
		}
	}

	if properties, ok := defMap["properties"].(map[string]any); ok {
		for _, prop := range properties {
			if propMap, ok := prop.(map[string]any); ok {
				if containsTimeType(propMap) {
					return true
				}
			}
		}
	}

	if items, ok := defMap["items"].(map[string]any); ok {
		if containsTimeType(items) {
			return true
		}
	}

	for _, key := range []string{"anyOf", "oneOf", "allOf"} {
		if schemas, ok := defMap[key].([]any); ok {
			for _, schema := range schemas {
				if schemaMap, ok := schema.(map[string]any); ok {
					if containsTimeType(schemaMap) {
						return true
					}
				}
			}
		}
	}

	return false
}
