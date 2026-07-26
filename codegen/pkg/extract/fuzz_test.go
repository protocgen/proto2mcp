package extract

import (
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// FuzzValidateToolName verifies that ValidateToolName never panics
// on arbitrary string input and always returns valid Warning structs.
func FuzzValidateToolName(f *testing.F) {
	f.Add("")
	f.Add("GetPatient")
	f.Add("PatientService_GetPatient")
	f.Add("a")
	f.Add("with-hyphens-and_underscores")
	f.Add("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa") // 67 chars
	f.Add("invalid!@#$%^&*()")
	f.Add("名前") // multibyte
	f.Add("\x00\xff\xfe")

	f.Fuzz(func(t *testing.T, name string) {
		warnings := ValidateToolName(name)
		for _, w := range warnings {
			if w.Severity != WarnError && w.Severity != WarnWarning && w.Severity != WarnInfo {
				t.Errorf("invalid severity %d for name %q", w.Severity, name)
			}
			if w.Message == "" {
				t.Errorf("empty warning message for name %q", name)
			}
		}
	})
}

// FuzzProtoKindToJSONType verifies that protoKindToJSONType never panics
// and always returns a string type for valid protoreflect.Kind values.
func FuzzProtoKindToJSONType(f *testing.F) {
	// Seed with all valid Kind values (1-18).
	for i := 1; i <= 18; i++ {
		f.Add(i)
	}
	// And some edge cases.
	f.Add(0)
	f.Add(19)
	f.Add(255)
	f.Add(-1)

	f.Fuzz(func(t *testing.T, kindInt int) {
		kind := protoreflect.Kind(kindInt)
		jsonType, format := protoKindToJSONType(kind)
		// For valid kinds (1-18), we expect a non-empty type.
		if kindInt >= 1 && kindInt <= 18 {
			if jsonType == "" {
				t.Errorf("empty jsonType for valid kind %d", kindInt)
			}
		}
		// Format is optional but should never cause issues.
		_ = format
	})
}

// FuzzWellKnownSchema verifies that wellKnownSchema never panics
// on arbitrary fully-qualified names.
func FuzzWellKnownSchema(f *testing.F) {
	f.Add("google.protobuf.Timestamp")
	f.Add("google.protobuf.Duration")
	f.Add("google.protobuf.StringValue")
	f.Add("google.protobuf.Any")
	f.Add("google.protobuf.NullValue")
	f.Add("google.protobuf.FieldMask")
	f.Add("com.example.CustomMessage")
	f.Add("")
	f.Add("google.protobuf.NonExistent")

	f.Fuzz(func(t *testing.T, fullName string) {
		fn := protoreflect.FullName(fullName)
		known := isWellKnown(fn)
		schema, ok := wellKnownSchema(fn)

		// If it's well-known and has a schema, the schema should have a type.
		if known && ok && schema != nil {
			if schema.Type == "" {
				t.Errorf("well-known type %q has empty schema type", fullName)
			}
		}
		// If it's not well-known, wellKnownSchema should return false.
		if !known && ok {
			t.Errorf("non-well-known type %q returned ok=true from wellKnownSchema", fullName)
		}
	})
}
