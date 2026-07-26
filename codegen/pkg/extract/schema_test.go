package extract

import (
	"encoding/json"
	"strings"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestProtoKindToJSONType(t *testing.T) {
	tests := []struct {
		name       string
		kind       protoreflect.Kind
		wantType   string
		wantFormat string
	}{
		{"Double", protoreflect.DoubleKind, "number", ""},
		{"Float", protoreflect.FloatKind, "number", ""},
		{"Int32", protoreflect.Int32Kind, "integer", ""},
		{"Sint32", protoreflect.Sint32Kind, "integer", ""},
		{"Sfixed32", protoreflect.Sfixed32Kind, "integer", ""},
		{"Uint32", protoreflect.Uint32Kind, "integer", ""},
		{"Fixed32", protoreflect.Fixed32Kind, "integer", ""},
		{"Int64", protoreflect.Int64Kind, "string", "int64"},
		{"Sint64", protoreflect.Sint64Kind, "string", "int64"},
		{"Sfixed64", protoreflect.Sfixed64Kind, "string", "int64"},
		{"Uint64", protoreflect.Uint64Kind, "string", "uint64"},
		{"Fixed64", protoreflect.Fixed64Kind, "string", "uint64"},
		{"Bool", protoreflect.BoolKind, "boolean", ""},
		{"String", protoreflect.StringKind, "string", ""},
		{"Bytes", protoreflect.BytesKind, "string", "byte"},
		{"Message", protoreflect.MessageKind, "string", ""}, // Default fallback in this func
		{"Enum", protoreflect.EnumKind, "string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotFormat := protoKindToJSONType(tt.kind)
			if gotType != tt.wantType {
				t.Errorf("protoKindToJSONType() gotType = %v, want %v", gotType, tt.wantType)
			}
			if gotFormat != tt.wantFormat {
				t.Errorf("protoKindToJSONType() gotFormat = %v, want %v", gotFormat, tt.wantFormat)
			}
		})
	}
}

func strPtr(s string) *string { return &s }
func typePtr(t descriptorpb.FieldDescriptorProto_Type) *descriptorpb.FieldDescriptorProto_Type { return &t }
func labelPtr(l descriptorpb.FieldDescriptorProto_Label) *descriptorpb.FieldDescriptorProto_Label { return &l }

func TestMessageToSchema(t *testing.T) {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("test.proto"),
		Package: strPtr("testpkg"),
		Options: &descriptorpb.FileOptions{
			GoPackage: strPtr("github.com/test/testpkg"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("SimpleMessage"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     strPtr("str_field"),
						Number:   func(i int32) *int32 { return &i }(1),
						Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING),
						JsonName: strPtr("strField"),
					},
					{
						Name:     strPtr("int_field"),
						Number:   func(i int32) *int32 { return &i }(2),
						Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_INT32),
						JsonName: strPtr("intField"),
					},
					{
						Name:     strPtr("int64_field"),
						Number:   func(i int32) *int32 { return &i }(3),
						Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_INT64),
						JsonName: strPtr("int64Field"),
					},
					{
						Name:     strPtr("bool_field"),
						Number:   func(i int32) *int32 { return &i }(4),
						Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_REPEATED),
						Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_BOOL),
						JsonName: strPtr("boolField"),
					},
				},
			},
			{
				Name: strPtr("NestedMessage"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     strPtr("nested"),
						Number:   func(i int32) *int32 { return &i }(1),
						Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
						TypeName: strPtr(".testpkg.SimpleMessage"),
						JsonName: strPtr("nested"),
					},
				},
			},
			{
				Name: strPtr("DeepMessage1"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     strPtr("next"),
						Number:   func(i int32) *int32 { return &i }(1),
						Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
						TypeName: strPtr(".testpkg.DeepMessage2"),
						JsonName: strPtr("next"),
					},
				},
			},
			{
				Name: strPtr("DeepMessage2"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     strPtr("next"),
						Number:   func(i int32) *int32 { return &i }(1),
						Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
						TypeName: strPtr(".testpkg.DeepMessage3"),
						JsonName: strPtr("next"),
					},
				},
			},
			{
				Name: strPtr("DeepMessage3"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     strPtr("next"),
						Number:   func(i int32) *int32 { return &i }(1),
						Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
						TypeName: strPtr(".testpkg.DeepMessage1"),
						JsonName: strPtr("next"),
					},
				},
			},
			{
				Name: strPtr("MapMessage"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     strPtr("map_field"),
						Number:   func(i int32) *int32 { return &i }(1),
						Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_REPEATED),
						Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
						TypeName: strPtr(".testpkg.MapMessage.MapFieldEntry"),
						JsonName: strPtr("mapField"),
					},
				},
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name: strPtr("MapFieldEntry"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:   strPtr("key"),
								Number: func(i int32) *int32 { return &i }(1),
								Label:  labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
								Type:   typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING),
							},
							{
								Name:   strPtr("value"),
								Number: func(i int32) *int32 { return &i }(2),
								Label:  labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
								Type:   typePtr(descriptorpb.FieldDescriptorProto_TYPE_INT32),
							},
						},
						Options: &descriptorpb.MessageOptions{
							MapEntry: func(b bool) *bool { return &b }(true),
						},
					},
				},
			},
		},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: strPtr("TestEnum"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: strPtr("UNKNOWN"), Number: func(i int32) *int32 { return &i }(0)},
					{Name: strPtr("STARTED"), Number: func(i int32) *int32 { return &i }(1)},
				},
			},
		},
	}

	fd.MessageType[0].Field = append(fd.MessageType[0].Field, &descriptorpb.FieldDescriptorProto{
		Name:     strPtr("enum_field"),
		Number:   func(i int32) *int32 { return &i }(5),
		Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
		Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_ENUM),
		TypeName: strPtr(".testpkg.TestEnum"),
		JsonName: strPtr("enumField"),
	})
	
	// Add Oneof
	fd.MessageType[0].OneofDecl = []*descriptorpb.OneofDescriptorProto{
		{Name: strPtr("test_oneof")},
	}
	fd.MessageType[0].Field = append(fd.MessageType[0].Field, &descriptorpb.FieldDescriptorProto{
		Name:     strPtr("oneof_str"),
		Number:   func(i int32) *int32 { return &i }(6),
		Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
		Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING),
		JsonName: strPtr("oneofStr"),
		OneofIndex: func(i int32) *int32 { return &i }(0),
	}, &descriptorpb.FieldDescriptorProto{
		Name:     strPtr("oneof_int"),
		Number:   func(i int32) *int32 { return &i }(7),
		Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
		Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_INT32),
		JsonName: strPtr("oneofInt"),
		OneofIndex: func(i int32) *int32 { return &i }(0),
	})

	plugin := buildTestPlugin(t, fd)

	file := plugin.FilesByPath["test.proto"]
	
	t.Run("SimpleMessage", func(t *testing.T) {
		msg := file.Messages[0]
		schemaBytes, err := MessageToSchema(msg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var schema map[string]interface{}
		if err := json.Unmarshal(schemaBytes, &schema); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}

		if schema["type"] != "object" {
			t.Error("expected object type")
		}
		props, ok := schema["properties"].(map[string]interface{})
		if !ok {
			t.Fatal("expected properties")
		}
		// Check string field
		if sf, ok := props["strField"].(map[string]interface{}); !ok || sf["type"] != "string" {
			t.Error("expected strField with type string")
		}
		// Check int32 field
		if sf, ok := props["intField"].(map[string]interface{}); !ok || sf["type"] != "integer" {
			t.Error("expected intField with type integer")
		}
		// Check int64 field: should be string with format int64
		if sf, ok := props["int64Field"].(map[string]interface{}); !ok || sf["type"] != "string" || sf["format"] != "int64" {
			t.Error("expected int64Field with type string and format int64")
		}
		// Check repeated bool field: should be array
		if sf, ok := props["boolField"].(map[string]interface{}); !ok || sf["type"] != "array" {
			t.Error("expected boolField as array type")
		}
		// Check enum field
		if sf, ok := props["enumField"].(map[string]interface{}); !ok || sf["type"] != "string" {
			t.Error("expected enumField with type string")
		}
		// Check additionalProperties: false
		if schema["additionalProperties"] != false {
			t.Error("expected additionalProperties: false on top-level schema")
		}

		// Check oneof fields: oneofStr and oneofInt should be present
		oneofStr, ok := props["oneofStr"].(map[string]interface{})
		if !ok {
			t.Fatal("expected oneofStr in properties")
		}
		if oneofStr["type"] != "string" {
			t.Errorf("expected oneofStr type=string, got %v", oneofStr["type"])
		}

		oneofInt, ok := props["oneofInt"].(map[string]interface{})
		if !ok {
			t.Fatal("expected oneofInt in properties")
		}
		if oneofInt["type"] != "integer" {
			t.Errorf("expected oneofInt type=integer, got %v", oneofInt["type"])
		}

		// Oneof fields should have mutual exclusivity hint in description
		oneofStrDesc, _ := oneofStr["description"].(string)
		if !strings.Contains(oneofStrDesc, "mutually exclusive") {
			t.Errorf("expected oneofStr description to contain mutual exclusivity hint, got %q", oneofStrDesc)
		}
		oneofIntDesc, _ := oneofInt["description"].(string)
		if !strings.Contains(oneofIntDesc, "mutually exclusive") {
			t.Errorf("expected oneofInt description to contain mutual exclusivity hint, got %q", oneofIntDesc)
		}

		// Oneof fields should NOT be in the required array
		required, _ := schema["required"].([]interface{})
		for _, r := range required {
			if r == "oneofStr" || r == "oneofInt" {
				t.Errorf("oneof field %q should not be in required array", r)
			}
		}
	})

	t.Run("NestedMessage", func(t *testing.T) {
		msg := file.Messages[1]
		schemaBytes, err := MessageToSchema(msg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var schema map[string]interface{}
		if err := json.Unmarshal(schemaBytes, &schema); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		props := schema["properties"].(map[string]interface{})
		nested, ok := props["nested"].(map[string]interface{})
		if !ok {
			t.Fatal("expected nested field")
		}
		if nested["type"] != "object" {
			t.Error("expected nested to be object type")
		}
	})

	t.Run("DeepMessage1 (recursive nesting)", func(t *testing.T) {
		msg := file.Messages[2]
		schemaBytes, err := MessageToSchema(msg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// With MaxRecursionDepth=6 and 3 cycling types, we expect deeply nested but valid JSON.
		var schema map[string]interface{}
		if err := json.Unmarshal(schemaBytes, &schema); err != nil {
			t.Fatalf("invalid JSON from recursive schema: %v", err)
		}
		if schema["type"] != "object" {
			t.Error("expected top-level object")
		}
		// Verify at least one level of nesting exists
		if props, ok := schema["properties"].(map[string]interface{}); ok {
			if _, hasNext := props["next"]; !hasNext {
				t.Error("expected 'next' field in recursive message")
			}
		}
	})

	t.Run("MapMessage", func(t *testing.T) {
		msg := file.Messages[5] // index 5 is MapMessage
		schemaBytes, err := MessageToSchema(msg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var schema map[string]interface{}
		if err := json.Unmarshal(schemaBytes, &schema); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		props := schema["properties"].(map[string]interface{})
		mapField, ok := props["mapField"].(map[string]interface{})
		if !ok {
			t.Fatal("expected mapField")
		}
		if mapField["type"] != "object" {
			t.Error("expected mapField to be object type")
		}
		if mapField["additionalProperties"] == nil {
			t.Error("expected additionalProperties on map field")
		}
	})
}
