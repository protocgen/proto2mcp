package extract

import (
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
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
