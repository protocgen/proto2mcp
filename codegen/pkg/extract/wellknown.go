package extract

import "google.golang.org/protobuf/reflect/protoreflect"

// isWellKnown checks if a message is a google.protobuf well-known type.
func isWellKnown(fullName protoreflect.FullName) bool {
	switch fullName {
	case "google.protobuf.Timestamp",
		"google.protobuf.Duration",
		"google.protobuf.StringValue",
		"google.protobuf.BoolValue",
		"google.protobuf.Int32Value",
		"google.protobuf.Int64Value",
		"google.protobuf.UInt32Value",
		"google.protobuf.UInt64Value",
		"google.protobuf.FloatValue",
		"google.protobuf.DoubleValue",
		"google.protobuf.BytesValue",
		"google.protobuf.Struct",
		"google.protobuf.Value",
		"google.protobuf.ListValue",
		"google.protobuf.FieldMask",
		"google.protobuf.Empty",
		"google.protobuf.Any",
		"google.protobuf.NullValue":
		return true
	default:
		return false
	}
}

// wellKnownSchema returns the SchemaField for a well-known type.
// Returns the schema and true if it's a well-known type, or nil/false otherwise.
func wellKnownSchema(fullName protoreflect.FullName) (*SchemaField, bool) {
	switch fullName {
	case "google.protobuf.Timestamp":
		return &SchemaField{Type: "string", Format: "date-time", Description: "RFC 3339 timestamp"}, true
	case "google.protobuf.Duration":
		return &SchemaField{Type: "string", Format: "duration", Description: "Duration (e.g., \"1.5s\")"}, true
	case "google.protobuf.StringValue":
		return &SchemaField{Type: "string"}, true
	case "google.protobuf.BoolValue":
		return &SchemaField{Type: "boolean"}, true
	case "google.protobuf.Int32Value":
		return &SchemaField{Type: "integer"}, true
	case "google.protobuf.Int64Value":
		return &SchemaField{Type: "string", Format: "int64"}, true
	case "google.protobuf.UInt32Value":
		return &SchemaField{Type: "integer"}, true
	case "google.protobuf.UInt64Value":
		return &SchemaField{Type: "string", Format: "uint64"}, true
	case "google.protobuf.FloatValue":
		return &SchemaField{Type: "number"}, true
	case "google.protobuf.DoubleValue":
		return &SchemaField{Type: "number"}, true
	case "google.protobuf.BytesValue":
		return &SchemaField{Type: "string", Format: "byte"}, true
	case "google.protobuf.Struct":
		return &SchemaField{Type: "object", AdditionalProperties: &SchemaField{}}, true
	case "google.protobuf.Value":
		return &SchemaField{}, true
	case "google.protobuf.ListValue":
		return &SchemaField{Type: "array", Items: &SchemaField{}}, true
	case "google.protobuf.FieldMask":
		return &SchemaField{Type: "string", Description: "Comma-separated field paths"}, true
	case "google.protobuf.Empty":
		return &SchemaField{Type: "object"}, true
	case "google.protobuf.Any":
		return nil, false
	case "google.protobuf.NullValue":
		return &SchemaField{Type: "null", Description: "JSON null value"}, true
	}
	return nil, false
}
