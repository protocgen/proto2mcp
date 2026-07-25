package extract

import (
	"reflect"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestIsWellKnown(t *testing.T) {
	tests := []struct {
		name     string
		fullName string
		want     bool
	}{
		{"Timestamp", "google.protobuf.Timestamp", true},
		{"Duration", "google.protobuf.Duration", true},
		{"StringValue", "google.protobuf.StringValue", true},
		{"BoolValue", "google.protobuf.BoolValue", true},
		{"Int32Value", "google.protobuf.Int32Value", true},
		{"Int64Value", "google.protobuf.Int64Value", true},
		{"UInt32Value", "google.protobuf.UInt32Value", true},
		{"UInt64Value", "google.protobuf.UInt64Value", true},
		{"FloatValue", "google.protobuf.FloatValue", true},
		{"DoubleValue", "google.protobuf.DoubleValue", true},
		{"BytesValue", "google.protobuf.BytesValue", true},
		{"Struct", "google.protobuf.Struct", true},
		{"Value", "google.protobuf.Value", true},
		{"ListValue", "google.protobuf.ListValue", true},
		{"FieldMask", "google.protobuf.FieldMask", true},
		{"Empty", "google.protobuf.Empty", true},
		{"Any", "google.protobuf.Any", true},
		{"Custom", "com.example.MyMessage", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isWellKnown(protoreflect.FullName(tt.fullName)); got != tt.want {
				t.Errorf("isWellKnown() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWellKnownSchema(t *testing.T) {
	tests := []struct {
		name     string
		fullName string
		wantOK   bool
		wantSF   *SchemaField
	}{
		{
			name:     "Timestamp",
			fullName: "google.protobuf.Timestamp",
			wantOK:   true,
			wantSF:   &SchemaField{Type: "string", Format: "date-time", Description: "RFC 3339 timestamp"},
		},
		{
			name:     "Duration",
			fullName: "google.protobuf.Duration",
			wantOK:   true,
			wantSF:   &SchemaField{Type: "string", Format: "duration", Description: "Duration (e.g., \"1.5s\")"},
		},
		{
			name:     "StringValue",
			fullName: "google.protobuf.StringValue",
			wantOK:   true,
			wantSF:   &SchemaField{Type: "string"},
		},
		{
			name:     "BoolValue",
			fullName: "google.protobuf.BoolValue",
			wantOK:   true,
			wantSF:   &SchemaField{Type: "boolean"},
		},
		{
			name:     "Int32Value",
			fullName: "google.protobuf.Int32Value",
			wantOK:   true,
			wantSF:   &SchemaField{Type: "integer"},
		},
		{
			name:     "Int64Value",
			fullName: "google.protobuf.Int64Value",
			wantOK:   true,
			wantSF:   &SchemaField{Type: "string", Format: "int64"},
		},
		{
			name:     "UInt32Value",
			fullName: "google.protobuf.UInt32Value",
			wantOK:   true,
			wantSF:   &SchemaField{Type: "integer"},
		},
		{
			name:     "UInt64Value",
			fullName: "google.protobuf.UInt64Value",
			wantOK:   true,
			wantSF:   &SchemaField{Type: "string", Format: "uint64"},
		},
		{
			name:     "FloatValue",
			fullName: "google.protobuf.FloatValue",
			wantOK:   true,
			wantSF:   &SchemaField{Type: "number"},
		},
		{
			name:     "DoubleValue",
			fullName: "google.protobuf.DoubleValue",
			wantOK:   true,
			wantSF:   &SchemaField{Type: "number"},
		},
		{
			name:     "BytesValue",
			fullName: "google.protobuf.BytesValue",
			wantOK:   true,
			wantSF:   &SchemaField{Type: "string", Format: "byte"},
		},
		{
			name:     "Struct",
			fullName: "google.protobuf.Struct",
			wantOK:   true,
			wantSF:   &SchemaField{Type: "object", AdditionalProperties: &SchemaField{}},
		},
		{
			name:     "Value",
			fullName: "google.protobuf.Value",
			wantOK:   true,
			wantSF:   &SchemaField{},
		},
		{
			name:     "ListValue",
			fullName: "google.protobuf.ListValue",
			wantOK:   true,
			wantSF:   &SchemaField{Type: "array", Items: &SchemaField{}},
		},
		{
			name:     "FieldMask",
			fullName: "google.protobuf.FieldMask",
			wantOK:   true,
			wantSF:   &SchemaField{Type: "string", Description: "Comma-separated field paths"},
		},
		{
			name:     "Empty",
			fullName: "google.protobuf.Empty",
			wantOK:   true,
			wantSF:   &SchemaField{Type: "object"},
		},
		{
			name:     "Any",
			fullName: "google.protobuf.Any",
			wantOK:   false,
			wantSF:   nil,
		},
		{
			name:     "Custom",
			fullName: "com.example.MyMessage",
			wantOK:   false,
			wantSF:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSF, gotOK := wellKnownSchema(protoreflect.FullName(tt.fullName))
			if gotOK != tt.wantOK {
				t.Errorf("wellKnownSchema() gotOK = %v, want %v", gotOK, tt.wantOK)
			}
			if !reflect.DeepEqual(gotSF, tt.wantSF) {
				t.Errorf("wellKnownSchema() gotSF = %v, want %v", gotSF, tt.wantSF)
			}
		})
	}
}
