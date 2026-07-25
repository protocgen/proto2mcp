package extract

import (
	"encoding/json"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// MaxRecursionDepth is the maximum nesting depth for schema generation.
const MaxRecursionDepth = 6

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

	var fields []SchemaField
	for _, field := range msg.Fields {
		fields = append(fields, protoFieldToSchemaField(field, depth))
	}
	return fields
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

	var jsonType, format string
	var enum []string
	var props []SchemaField
	var addProps *SchemaField
	var items *SchemaField

	kind := field.Desc.Kind()

	if field.Desc.IsMap() {
		jsonType = "object"
		valField := field.Message.Fields[1] // Value field in map entry
		vsf := protoFieldToSchemaField(valField, depth+1)
		addProps = &SchemaField{
			Type:                 vsf.Type,
			Format:               vsf.Format,
			Description:          vsf.Description,
			Properties:           vsf.Properties,
			Items:                vsf.Items,
			AdditionalProperties: vsf.AdditionalProperties,
			Enum:                 vsf.Enum,
		}
	} else if kind == protoreflect.MessageKind || kind == protoreflect.GroupKind {
		fullName := field.Message.Desc.FullName()
		if isWellKnown(fullName) {
			if wk, ok := wellKnownSchema(fullName); ok && wk != nil {
				jsonType = wk.Type
				format = wk.Format
				props = wk.Properties
				addProps = wk.AdditionalProperties
				items = wk.Items
				if wk.Description != "" {
					if descBuilder.Len() > 0 {
						descBuilder.WriteString("\n")
					}
					descBuilder.WriteString(wk.Description)
				}
			}
		} else {
			jsonType = "object"
			if depth >= MaxRecursionDepth {
				if descBuilder.Len() > 0 {
					descBuilder.WriteString("\n")
				}
				descBuilder.WriteString("...(truncated)...")
			} else {
				props = messageToSchemaFields(field.Message, depth+1)
			}
		}
	} else if kind == protoreflect.EnumKind {
		jsonType = "string"
		for _, val := range field.Enum.Values {
			enum = append(enum, string(val.Desc.Name()))
		}
	} else {
		jsonType, format = protoKindToJSONType(kind)
		if format == "int64" || format == "uint64" {
			note := " (serialized as string for 64-bit precision)"
			if descBuilder.Len() > 0 {
				descBuilder.WriteString(note)
			} else {
				descBuilder.WriteString(strings.TrimSpace(note))
			}
		}
	}

	sf.Description = descBuilder.String()

	if field.Desc.IsList() && !field.Desc.IsMap() {
		sf.Type = "array"
		sf.Items = &SchemaField{
			Type:                 jsonType,
			Format:               format,
			Properties:           props,
			AdditionalProperties: addProps,
			Items:                items,
			Enum:                 enum,
		}
	} else {
		sf.Type = jsonType
		sf.Format = format
		sf.Properties = props
		sf.AdditionalProperties = addProps
		sf.Items = items
		sf.Enum = enum
	}

	// Make sure fields that are part of a oneof are just listed as fields.
	// proto3 optional fields are structurally identical to oneofs with a single field.
	return sf
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
