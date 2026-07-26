package extract

import (
	"encoding/json"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// MaxRecursionDepth is the maximum nesting depth for schema generation.
const MaxRecursionDepth = 6

// boolPtr is a helper to get a pointer to a boolean.
func boolPtr(b bool) *bool { return &b }

// boolFalse is a pre-allocated pointer to false, avoiding repeated
// heap allocations for the common additionalProperties: false case.
var boolFalse = boolPtr(false)

// anyFullName is the fully qualified name for google.protobuf.Any.
const anyFullName = "google.protobuf.Any"

// MessageToSchema converts a proto message to a JSON Schema as json.RawMessage.
// It recursively walks fields, handling nested messages, enums, maps, oneofs,
// and repeated fields. Recurses up to MaxRecursionDepth levels.
func MessageToSchema(msg *protogen.Message) (json.RawMessage, error) {
	fields := messageToSchemaFields(msg, 0)
	return MarshalSchemaFields(fields)
}

// messageToSchemaFields converts a message's fields to SchemaFields.
// depth tracks recursion to prevent infinite loops from circular references.
func messageToSchemaFields(msg *protogen.Message, depth int) []SchemaField {
	if depth >= MaxRecursionDepth {
		return nil
	}

	oneofFields := make(map[int][]string)
	for _, field := range msg.Fields {
		if field.Oneof != nil && !field.Oneof.Desc.IsSynthetic() {
			idx := field.Oneof.Desc.Index()
			oneofFields[idx] = append(oneofFields[idx], string(field.Desc.JSONName()))
		}
	}

	var fields []SchemaField
	for _, field := range msg.Fields {
		sf := protoFieldToSchemaField(field, depth)

		if field.Oneof != nil && !field.Oneof.Desc.IsSynthetic() {
			idx := field.Oneof.Desc.Index()
			var others []string
			for _, name := range oneofFields[idx] {
				if name != string(field.Desc.JSONName()) {
					others = append(others, name)
				}
			}
			if len(others) > 0 {
				hint := " (mutually exclusive with: " + strings.Join(others, ", ") + ")"
				sf.Description = strings.TrimSpace(sf.Description + hint)
			}
			sf.Required = false
		}

		fields = append(fields, sf)
	}
	return fields
}

// schemaResult holds the type-specific fields resolved by the schema helpers.
type schemaResult struct {
	jsonType string
	format   string
	enum     []string
	props    []SchemaField
	addProps any
	items    *SchemaField
	descNote string // extra text to append to the description
}

// protoFieldToSchemaField converts a single proto field descriptor to a SchemaField.
func protoFieldToSchemaField(field *protogen.Field, depth int) SchemaField {
	sf := SchemaField{
		Name: string(field.Desc.JSONName()),
	}

	var descBuilder strings.Builder
	if field.Comments.Leading.String() != "" {
		descBuilder.WriteString(strings.TrimSpace(field.Comments.Leading.String()))
	}

	kind := field.Desc.Kind()

	var r schemaResult
	switch {
	case field.Desc.IsMap():
		r = schemaForMap(field, depth)
	case kind == protoreflect.MessageKind || kind == protoreflect.GroupKind:
		r = schemaForMessage(field, &sf, depth)
	case kind == protoreflect.EnumKind:
		r = schemaForEnum(field)
	default:
		r = schemaForScalar(kind)
	}

	// Append any type-specific description note.
	if r.descNote != "" {
		if descBuilder.Len() > 0 {
			descBuilder.WriteString("\n")
		}
		descBuilder.WriteString(r.descNote)
	}

	sf.Description = descBuilder.String()

	if field.Desc.IsList() && !field.Desc.IsMap() {
		sf.Type = "array"
		sf.Items = &SchemaField{
			Type:                 r.jsonType,
			Format:               r.format,
			Properties:           r.props,
			AdditionalProperties: r.addProps,
			Items:                r.items,
			Enum:                 r.enum,
		}
	} else {
		sf.Type = r.jsonType
		sf.Format = r.format
		sf.Properties = r.props
		sf.AdditionalProperties = r.addProps
		sf.Items = r.items
		sf.Enum = r.enum
	}

	// Wire in buf.validate constraints and required flag.
	constraints := ExtractConstraints(field.Desc)
	if constraints != nil {
		sf.Constraints = constraints
	}
	if IsFieldRequired(field.Desc) {
		sf.Required = true
	}

	return sf
}

// schemaForMap handles proto map fields, returning an object type
// with additionalProperties describing the map value type.
func schemaForMap(field *protogen.Field, depth int) schemaResult {
	r := schemaResult{jsonType: "object"}
	valDesc := field.Message.Desc.Fields().ByNumber(2)
	if valDesc == nil {
		return r
	}
	for _, f := range field.Message.Fields {
		if f.Desc.Number() == valDesc.Number() {
			vsf := protoFieldToSchemaField(f, depth+1)
			r.addProps = &SchemaField{
				Type:                 vsf.Type,
				Format:               vsf.Format,
				Title:                vsf.Title,
				Description:          vsf.Description,
				Properties:           vsf.Properties,
				Items:                vsf.Items,
				AdditionalProperties: vsf.AdditionalProperties,
				Enum:                 vsf.Enum,
				Constraints:          vsf.Constraints,
			}
			break
		}
	}
	return r
}

// schemaForMessage handles proto message fields, including well-known types,
// google.protobuf.Any, and user-defined nested messages.
func schemaForMessage(field *protogen.Field, sf *SchemaField, depth int) schemaResult {
	fullName := field.Message.Desc.FullName()
	sf.Title = string(field.Message.Desc.Name())
	// Note: additionalProperties is set via schemaResult.addProps, NOT directly
	// on sf, to avoid leaking it onto array schemas for repeated message fields.

	if string(fullName) == anyFullName {
		return schemaResult{
			jsonType: "object",
			addProps: boolFalse,
			descNote: "WARNING: google.protobuf.Any cannot be represented as JSON Schema",
		}
	}

	if isWellKnown(fullName) {
		if wk, ok := wellKnownSchema(fullName); ok && wk != nil {
			return schemaResult{
				jsonType: wk.Type,
				format:   wk.Format,
				props:    wk.Properties,
				addProps: wk.AdditionalProperties,
				items:    wk.Items,
				descNote: wk.Description,
			}
		}
	}

	r := schemaResult{jsonType: "object", addProps: boolFalse}
	if depth >= MaxRecursionDepth {
		r.descNote = "...(truncated)..."
	} else {
		r.props = messageToSchemaFields(field.Message, depth+1)
	}
	return r
}

// schemaForEnum handles proto enum fields by extracting all enum value names.
// Special-cases google.protobuf.NullValue to map to JSON null type.
func schemaForEnum(field *protogen.Field) schemaResult {
	// google.protobuf.NullValue is a well-known enum that maps to JSON null.
	if field.Enum.Desc.FullName() == "google.protobuf.NullValue" {
		return schemaResult{jsonType: "null", descNote: "JSON null value"}
	}

	r := schemaResult{jsonType: "string"}
	for _, val := range field.Enum.Values {
		r.enum = append(r.enum, string(val.Desc.Name()))
	}
	return r
}

// schemaForScalar handles scalar proto fields (string, int, bool, etc.).
func schemaForScalar(kind protoreflect.Kind) schemaResult {
	jsonType, format := protoKindToJSONType(kind)
	r := schemaResult{jsonType: jsonType, format: format}
	if format == "int64" || format == "uint64" {
		r.descNote = "(serialized as string for 64-bit precision)"
	}
	return r
}

// protoKindToJSONType maps a protobuf field kind to JSON Schema type.
// CRITICAL: All 64-bit integers (int64, uint64, sint64, fixed64, sfixed64)
// MUST map to "string" because protojson serializes them as strings
// (JSON numbers lose precision beyond 2^53). Add description annotation.
func protoKindToJSONType(kind protoreflect.Kind) (jsonType string, format string) {
	switch kind {
	case protoreflect.DoubleKind, protoreflect.FloatKind:
		return "number", ""
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return "integer", ""
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return "integer", ""
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return "string", "int64"
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return "string", "uint64"
	case protoreflect.BoolKind:
		return "boolean", ""
	case protoreflect.StringKind:
		return "string", ""
	case protoreflect.BytesKind:
		return "string", "byte"
	default:
		return "string", ""
	}
}
