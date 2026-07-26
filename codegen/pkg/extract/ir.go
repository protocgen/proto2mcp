package extract

import (
	"encoding/json"
)

// FileIR represents the extraction result for an entire proto file.
type FileIR struct {
	// FileName is the proto file name (e.g., "myapp/v1/patient.proto").
	FileName string
	// Services is the list of services extracted from the file.
	Services []ServiceIR
	// Warnings contains file-level linter warnings.
	Warnings []Warning
	// Skip is true if the file was annotated with skip=true.
	Skip bool
}

// ServiceIR represents a protobuf service extracted for MCP generation.
type ServiceIR struct {
	// Name is the Go name of the service.
	Name string
	// FullName is the fully qualified protobuf name of the service.
	FullName string
	// Description is derived from proto comments.
	Description string
	// Tools is the list of MCP tools derived from service methods.
	Tools []ToolIR
	// Options holds the service-level MCP options.
	Options ServiceOptions
}

// ToolIR represents a single MCP tool derived from a protobuf method.
type ToolIR struct {
	// Name is the MCP tool name (e.g., "PatientService_GetPatient").
	Name string
	// MethodName is the original proto method name.
	MethodName string
	// Description is the LLM-facing description.
	Description string
	// InputSchema is the JSON Schema for the input.
	InputSchema json.RawMessage
	// InputTypeName is the fully qualified proto input type.
	InputTypeName string
	// OutputTypeName is the fully qualified proto output type.
	OutputTypeName string
	// IsResource indicates if the tool should be exposed as an MCP Resource (V2).
	IsResource bool
	// ResourceURI is the URI template for resources (V2).
	ResourceURI string
	// IsReadOnly hints if the tool is read-only (V2).
	IsReadOnly bool
	// IsDestructive hints if the tool is destructive (V2).
	IsDestructive bool
	// IsDeprecated indicates if the method is deprecated.
	IsDeprecated bool
	// Version is the version number (V2).
	Version int32
	// SubTools represents macro-tool sub-tools (V3).
	SubTools []ToolRef
	// MacroType classifies how a macro-tool executes (V3).
	MacroType MacroType
	// Warnings contains any LLM Linter warnings.
	Warnings []Warning
}

// ToolRef references another tool by name (for macro-tools).
type ToolRef struct {
	// ToolName is the name of the referenced tool.
	ToolName string
	// Parallel indicates if the tool can be run in parallel.
	Parallel bool
	// OutputKey is the key under which the output should be saved.
	OutputKey string
}

// MacroType classifies how a macro-tool executes.
type MacroType int

const (
	// MacroTypeNone implies a normal tool (not a macro).
	MacroTypeNone MacroType = iota
	// MacroTypeSequential means steps run sequentially (V3).
	MacroTypeSequential
	// MacroTypeParallel means independent steps run concurrently (V3).
	MacroTypeParallel
	// MacroTypeTemporal means Azra-specific Temporal workflow (V3).
	MacroTypeTemporal
)

// ServiceOptions holds extracted service-level MCP options.
type ServiceOptions struct {
	// ToolNamePrefix is an optional prefix for tool names.
	ToolNamePrefix string
	// Description is a service-level description.
	Description string
	// GenerateConnect enables ConnectRPC forwarding code generation.
	GenerateConnect bool
}

// Warning represents an LLM Linter warning emitted during extraction.
type Warning struct {
	// Severity indicates the warning level.
	Severity WarningLevel
	// Method is the method that triggered the warning.
	Method string
	// Message is the warning message.
	Message string
}

// WarningLevel indicates the severity of a linter warning.
type WarningLevel int

const (
	// WarnInfo represents informational warnings.
	WarnInfo WarningLevel = iota
	// WarnWarning represents a typical warning.
	WarnWarning
	// WarnError represents a hard build error (e.g., streaming, Any).
	WarnError
)

// SchemaField represents a field in a JSON Schema.
type SchemaField struct {
	// Name of the field.
	Name string
	// Title of the schema, typically the message name.
	Title string `json:"title,omitempty"`
	// Type of the field: "string", "integer", "number", "boolean", "object", "array".
	Type string
	// Description of the field.
	Description string
	// Required indicates if the field is mandatory.
	Required bool
	// Format is the format (e.g., "date-time" for Timestamp).
	Format string
	// Properties represents nested fields for objects.
	Properties []SchemaField
	// Items represents the item type for arrays.
	Items *SchemaField
	// AdditionalProperties represents the value type for proto maps (*SchemaField)
	// or a boolean (*bool) for strict schemas to prevent hallucinations.
	AdditionalProperties any `json:"additionalProperties,omitempty"`
	// OneOf represents mutually exclusive alternatives (proto oneof).
	OneOf []SchemaField
	// Enum holds possible values for proto enums.
	Enum []string
	// Constraints holds validation constraints (from buf.validate):
	// keys are JSON Schema keywords like "minLength", "pattern", "minimum".
	Constraints map[string]any
}

// jsonSchema is the internal representation used for JSON Schema marshaling.
type jsonSchema struct {
	Type                 string         `json:"type,omitempty"`
	Description          string         `json:"description,omitempty"`
	Format               string         `json:"format,omitempty"`
	Properties           map[string]any `json:"properties,omitempty"`
	Required             []string       `json:"required,omitempty"`
	Items                any            `json:"items,omitempty"`
	AdditionalProperties any            `json:"additionalProperties,omitempty"`
	OneOf                []any          `json:"oneOf,omitempty"`
	Enum                 []string       `json:"enum,omitempty"`
}

// MarshalSchemaFields converts a list of SchemaFields into a JSON Schema
// object (as json.RawMessage). This is used to populate ToolIR.InputSchema.
func MarshalSchemaFields(fields []SchemaField) (json.RawMessage, error) {
	schema := jsonSchema{
		Type:                 "object",
		Properties:           make(map[string]any),
		AdditionalProperties: false,
	}

	var required []string
	for _, f := range fields {
		prop := fieldToJSONSchema(f)
		schema.Properties[f.Name] = prop
		if f.Required {
			required = append(required, f.Name)
		}
	}
	if len(required) > 0 {
		schema.Required = required
	}

	return json.Marshal(schema)
}

// fieldToJSONSchema converts a single SchemaField to a JSON Schema property.
func fieldToJSONSchema(f SchemaField) map[string]any {
	prop := make(map[string]any)

	if f.Title != "" {
		prop["title"] = f.Title
	}
	if f.Type != "" {
		prop["type"] = f.Type
	}
	if f.Description != "" {
		prop["description"] = f.Description
	}
	if f.Format != "" {
		prop["format"] = f.Format
	}
	if len(f.Enum) > 0 {
		prop["enum"] = f.Enum
	}

	// Nested object
	if f.Type == "object" && len(f.Properties) > 0 {
		props := make(map[string]any)
		var req []string
		for _, sub := range f.Properties {
			props[sub.Name] = fieldToJSONSchema(sub)
			if sub.Required {
				req = append(req, sub.Name)
			}
		}
		prop["properties"] = props
		if len(req) > 0 {
			prop["required"] = req
		}
	}

	// Map type or strict object (additionalProperties)
	if f.AdditionalProperties != nil {
		if b, ok := f.AdditionalProperties.(*bool); ok {
			prop["additionalProperties"] = *b
		} else if b, ok := f.AdditionalProperties.(bool); ok {
			prop["additionalProperties"] = b
		} else if sf, ok := f.AdditionalProperties.(*SchemaField); ok {
			prop["additionalProperties"] = fieldToJSONSchema(*sf)
		} else if sf, ok := f.AdditionalProperties.(SchemaField); ok {
			prop["additionalProperties"] = fieldToJSONSchema(sf)
		}
	}

	// Array type
	if f.Items != nil {
		prop["items"] = fieldToJSONSchema(*f.Items)
	}

	// Oneof
	if len(f.OneOf) > 0 {
		var oneOf []any
		for _, alt := range f.OneOf {
			oneOf = append(oneOf, fieldToJSONSchema(alt))
		}
		prop["oneOf"] = oneOf
	}

	// Validation constraints from buf.validate
	for k, v := range f.Constraints {
		prop[k] = v
	}

	return prop
}
