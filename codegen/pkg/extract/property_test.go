package extract

import (
	"encoding/json"
	"testing"

	"hegel.dev/go/hegel"
)

// ============================================================================
// Property-Based Tests: SchemaField JSON Roundtrip
// ============================================================================

// genSchemaField generates arbitrary SchemaField values using Hegel.
var genSchemaField = hegel.Composite(func(tc hegel.TestCase) SchemaField {
	f := SchemaField{
		Name:        hegel.Draw(tc, hegel.Text().Alphabet("abcdefghijklmnopqrstuvwxyz_").MinSize(1).MaxSize(30)),
		Type:        hegel.Draw(tc, hegel.SampledFrom([]string{"string", "integer", "number", "boolean", "object", "array"})),
		Description: hegel.Draw(tc, hegel.Text().MaxSize(200)),
		Required:    hegel.Draw(tc, hegel.Booleans()),
	}

	// Optionally add format
	if hegel.Draw(tc, hegel.Booleans()) {
		f.Format = hegel.Draw(tc, hegel.SampledFrom([]string{"int64", "uint64", "date-time", "duration", "byte", "email", "uri", "uuid"}))
	}

	// Optionally add title
	if hegel.Draw(tc, hegel.Booleans()) {
		f.Title = hegel.Draw(tc, hegel.Text().MaxSize(50))
	}

	// Optionally add enum values for string type
	if f.Type == "string" && hegel.Draw(tc, hegel.Booleans()) {
		enumCount := hegel.Draw(tc, hegel.Integers(1, 5))
		for range enumCount {
			f.Enum = append(f.Enum, hegel.Draw(tc, hegel.Text().Alphabet("ABCDEFGHIJKLMNOPQRSTUVWXYZ_0123456789").MinSize(1).MaxSize(20)))
		}
	}

	return f
})

func TestProperty_SchemaField_MarshalProducesValidJSON(t *testing.T) {
	hegel.Test(t, func(ht *hegel.T) {
		field := hegel.Draw(ht, genSchemaField)
		fields := []SchemaField{field}

		result, err := MarshalSchemaFields(fields)
		if err != nil {
			ht.Fatalf("MarshalSchemaFields failed: %v", err)
		}

		// Property: output must always be valid JSON
		var parsed map[string]interface{}
		if err := json.Unmarshal(result, &parsed); err != nil {
			ht.Fatalf("produced invalid JSON: %v\nraw: %s", err, result)
		}

		// Property: output must always have "type": "object"
		if parsed["type"] != "object" {
			ht.Fatalf("expected type=object, got %v", parsed["type"])
		}

		// Property: output must always have "properties"
		if parsed["properties"] == nil {
			ht.Fatal("expected properties key")
		}

		// Property: output must have additionalProperties: false
		if parsed["additionalProperties"] != false {
			ht.Fatal("expected additionalProperties=false")
		}
	}, hegel.WithTestCases(200))
}

func TestProperty_SchemaField_RequiredFieldsAppearInArray(t *testing.T) {
	hegel.Test(t, func(ht *hegel.T) {
		// Generate 1-5 fields, some required some not
		fieldCount := hegel.Draw(ht, hegel.Integers(1, 5))
		fields := make([]SchemaField, fieldCount)
		expectedRequired := 0
		for i := range fieldCount {
			fields[i] = hegel.Draw(ht, genSchemaField)
			// Ensure unique names
			fields[i].Name = fields[i].Name + "_" + string(rune('a'+i))
			if fields[i].Required {
				expectedRequired++
			}
		}

		result, err := MarshalSchemaFields(fields)
		if err != nil {
			ht.Fatalf("MarshalSchemaFields failed: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(result, &parsed); err != nil {
			ht.Fatalf("invalid JSON: %v", err)
		}

		// Property: required array length matches number of required fields
		reqArray, _ := parsed["required"].([]interface{})
		if len(reqArray) != expectedRequired {
			ht.Fatalf("expected %d required fields, got %d", expectedRequired, len(reqArray))
		}
	}, hegel.WithTestCases(200))
}

func TestProperty_SchemaField_MultipleFieldsAllPresent(t *testing.T) {
	hegel.Test(t, func(ht *hegel.T) {
		fieldCount := hegel.Draw(ht, hegel.Integers(1, 8))
		fields := make([]SchemaField, fieldCount)
		names := make(map[string]bool)
		for i := range fieldCount {
			fields[i] = hegel.Draw(ht, genSchemaField)
			// Ensure unique names
			fields[i].Name = fields[i].Name + "_" + string(rune('a'+i))
			names[fields[i].Name] = true
		}

		result, err := MarshalSchemaFields(fields)
		if err != nil {
			ht.Fatalf("MarshalSchemaFields failed: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(result, &parsed); err != nil {
			ht.Fatalf("invalid JSON: %v", err)
		}

		props := parsed["properties"].(map[string]interface{})

		// Property: every generated field must appear in properties
		for name := range names {
			if props[name] == nil {
				ht.Fatalf("field %q missing from properties", name)
			}
		}

		// Property: properties count matches input count
		if len(props) != fieldCount {
			ht.Fatalf("expected %d properties, got %d", fieldCount, len(props))
		}
	}, hegel.WithTestCases(200))
}

func TestProperty_SchemaField_NestedObjectRoundtrip(t *testing.T) {
	hegel.Test(t, func(ht *hegel.T) {
		// Build a 2-level nested schema
		innerField := hegel.Draw(ht, genSchemaField)
		innerField.Name = "inner_field"

		outer := SchemaField{
			Name:        "outer",
			Type:        "object",
			Description: hegel.Draw(ht, hegel.Text().MaxSize(100)),
			Properties:  []SchemaField{innerField},
		}

		result, err := MarshalSchemaFields([]SchemaField{outer})
		if err != nil {
			ht.Fatalf("MarshalSchemaFields failed: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(result, &parsed); err != nil {
			ht.Fatalf("invalid JSON: %v", err)
		}

		props := parsed["properties"].(map[string]interface{})
		outerProp := props["outer"].(map[string]interface{})

		// Property: nested object must have type "object"
		if outerProp["type"] != "object" {
			ht.Fatalf("expected nested type=object, got %v", outerProp["type"])
		}

		// Property: nested object must have properties
		nestedProps, ok := outerProp["properties"].(map[string]interface{})
		if !ok {
			ht.Fatal("expected nested properties")
		}

		// Property: inner field must be present
		if nestedProps["inner_field"] == nil {
			ht.Fatal("inner_field missing from nested properties")
		}
	}, hegel.WithTestCases(100))
}

func TestProperty_SchemaField_ArrayFieldRoundtrip(t *testing.T) {
	hegel.Test(t, func(ht *hegel.T) {
		itemType := hegel.Draw(ht, hegel.SampledFrom([]string{"string", "integer", "number", "boolean"}))

		field := SchemaField{
			Name: "items_field",
			Type: "array",
			Items: &SchemaField{
				Type: itemType,
			},
		}

		result, err := MarshalSchemaFields([]SchemaField{field})
		if err != nil {
			ht.Fatalf("MarshalSchemaFields failed: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(result, &parsed); err != nil {
			ht.Fatalf("invalid JSON: %v", err)
		}

		props := parsed["properties"].(map[string]interface{})
		arrProp := props["items_field"].(map[string]interface{})

		// Property: array field has type "array"
		if arrProp["type"] != "array" {
			ht.Fatalf("expected type=array, got %v", arrProp["type"])
		}

		// Property: array field has items
		items, ok := arrProp["items"].(map[string]interface{})
		if !ok {
			ht.Fatal("expected items in array field")
		}

		// Property: items type matches what we set
		if items["type"] != itemType {
			ht.Fatalf("expected items.type=%s, got %v", itemType, items["type"])
		}
	}, hegel.WithTestCases(100))
}

func TestProperty_SchemaField_EmptyFieldsProducesMinimalSchema(t *testing.T) {
	hegel.Test(t, func(ht *hegel.T) {
		result, err := MarshalSchemaFields([]SchemaField{})
		if err != nil {
			ht.Fatalf("MarshalSchemaFields failed for empty: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(result, &parsed); err != nil {
			ht.Fatalf("invalid JSON: %v", err)
		}

		// Property: empty produces type=object with empty properties
		if parsed["type"] != "object" {
			ht.Fatal("expected type=object for empty fields")
		}
	})
}
