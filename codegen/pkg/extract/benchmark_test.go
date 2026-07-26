package extract

import (
	"testing"
)

func BenchmarkMarshalSchemaFields_Small(b *testing.B) {
	b.ReportAllocs()
	fields := []SchemaField{
		{Name: "id", Type: "string", Description: "Patient identifier"},
		{Name: "active", Type: "boolean", Description: "Whether the patient is active"},
		{Name: "age", Type: "integer", Description: "Patient age in years"},
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = MarshalSchemaFields(fields)
	}
}

func BenchmarkMarshalSchemaFields_Medium(b *testing.B) {
	b.ReportAllocs()
	fields := make([]SchemaField, 10)
	for i := range fields {
		fields[i] = SchemaField{
			Name:        "field_" + string(rune('a'+i)),
			Type:        "string",
			Description: "A test field for benchmarking purposes",
			Required:    i%2 == 0,
		}
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = MarshalSchemaFields(fields)
	}
}

func BenchmarkMarshalSchemaFields_Large(b *testing.B) {
	b.ReportAllocs()
	fields := make([]SchemaField, 25)
	types := []string{"string", "integer", "number", "boolean", "string"}
	for i := range fields {
		fields[i] = SchemaField{
			Name:        "field_" + string(rune('a'+i)),
			Type:        types[i%len(types)],
			Description: "A test field for large schema benchmarking",
			Required:    i < 5,
		}
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = MarshalSchemaFields(fields)
	}
}

func BenchmarkMarshalSchemaFields_Nested(b *testing.B) {
	b.ReportAllocs()
	fields := []SchemaField{
		{Name: "id", Type: "string"},
		{
			Name: "address",
			Type: "object",
			Properties: []SchemaField{
				{Name: "street", Type: "string"},
				{Name: "city", Type: "string"},
				{
					Name: "geo",
					Type: "object",
					Properties: []SchemaField{
						{Name: "lat", Type: "number"},
						{Name: "lng", Type: "number"},
					},
				},
			},
		},
		{
			Name: "tags",
			Type: "array",
			Items: &SchemaField{
				Type: "string",
			},
		},
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = MarshalSchemaFields(fields)
	}
}
