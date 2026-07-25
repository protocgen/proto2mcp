package extract

import (
	"encoding/json"
	"testing"
)

func TestMarshalSchemaFields(t *testing.T) {
	tests := []struct {
		name    string
		fields  []SchemaField
		want    string
		wantErr bool
	}{
		{
			name:   "empty fields",
			fields: []SchemaField{},
			want:   `{"type":"object"}`,
		},
		{
			name: "simple object",
			fields: []SchemaField{
				{Name: "str_field", Type: "string", Description: "A string"},
				{Name: "int_field", Type: "integer"},
				{Name: "bool_field", Type: "boolean", Required: true},
			},
			want: `{"type":"object","properties":{"bool_field":{"type":"boolean"},"int_field":{"type":"integer"},"str_field":{"description":"A string","type":"string"}},"required":["bool_field"]}`,
		},
		{
			name: "nested object",
			fields: []SchemaField{
				{
					Name: "nested",
					Type: "object",
					Properties: []SchemaField{
						{Name: "inner_str", Type: "string", Required: true},
					},
					Required: true,
				},
			},
			want: `{"type":"object","properties":{"nested":{"properties":{"inner_str":{"type":"string"}},"required":["inner_str"],"type":"object"}},"required":["nested"]}`,
		},
		{
			name: "array field",
			fields: []SchemaField{
				{
					Name: "arr",
					Type: "array",
					Items: &SchemaField{
						Type: "string",
					},
				},
			},
			want: `{"type":"object","properties":{"arr":{"items":{"type":"string"},"type":"array"}}}`,
		},
		{
			name: "map field",
			fields: []SchemaField{
				{
					Name: "map_field",
					Type: "object",
					AdditionalProperties: &SchemaField{
						Type: "integer",
					},
				},
			},
			want: `{"type":"object","properties":{"map_field":{"additionalProperties":{"type":"integer"},"type":"object"}}}`,
		},
		{
			name: "oneof field",
			fields: []SchemaField{
				{
					Name: "choice",
					OneOf: []SchemaField{
						{Type: "string"},
						{Type: "integer"},
					},
				},
			},
			want: `{"type":"object","properties":{"choice":{"oneOf":[{"type":"string"},{"type":"integer"}]}}}`,
		},
		{
			name: "constraints",
			fields: []SchemaField{
				{
					Name: "constrained_str",
					Type: "string",
					Constraints: map[string]any{
						"minLength": 5,
						"pattern":   "^[a-z]+$",
					},
				},
			},
			want: `{"type":"object","properties":{"constrained_str":{"minLength":5,"pattern":"^[a-z]+$","type":"string"}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MarshalSchemaFields(tt.fields)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalSchemaFields() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// verify it's valid JSON
				var js map[string]any
				if err := json.Unmarshal(got, &js); err != nil {
					t.Fatalf("MarshalSchemaFields() returned invalid JSON: %v, output: %s", err, string(got))
				}

				// to compare, we can use string comparison if we marshal standard ways, 
				// or just use json.RawMessage comparison, or compare as map
				var wantMap map[string]any
				if err := json.Unmarshal([]byte(tt.want), &wantMap); err != nil {
					t.Fatalf("invalid want JSON: %v", err)
				}
				
				// doing simple string compare could fail due to map ordering,
				// so let's unmarshal and remarshal both, or use map comparison
				gotBytes, _ := json.Marshal(js)
				wantBytes, _ := json.Marshal(wantMap)
				if string(gotBytes) != string(wantBytes) {
					t.Errorf("MarshalSchemaFields() = %s, want %s", string(gotBytes), string(wantBytes))
				}
			}
		})
	}
}
