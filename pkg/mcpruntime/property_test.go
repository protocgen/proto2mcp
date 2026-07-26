package mcpruntime

import (
	"encoding/json"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
	"hegel.dev/go/hegel"
)

// ============================================================================
// Property-Based Tests: MarshalToolResult / UnmarshalToolInput Roundtrip
// ============================================================================

func TestProperty_MarshalUnmarshal_Roundtrip(t *testing.T) {
	hegel.Test(t, func(ht *hegel.T) {
		// Generate arbitrary key-value pairs for a Struct
		numFields := hegel.Draw(ht, hegel.Integers(1, 10))
		fields := make(map[string]interface{})
		for range numFields {
			key := hegel.Draw(ht, hegel.Text().Alphabet("abcdefghijklmnopqrstuvwxyz_").MinSize(1).MaxSize(20))
			// Generate a value — keep it to types that survive JSON roundtrip
			valType := hegel.Draw(ht, hegel.Integers(0, 3))
			switch valType {
			case 0:
				fields[key] = hegel.Draw(ht, hegel.Text().MaxSize(100))
			case 1:
				fields[key] = float64(hegel.Draw(ht, hegel.Integers(-1000, 1000)))
			case 2:
				fields[key] = hegel.Draw(ht, hegel.Booleans())
			case 3:
				fields[key] = nil
			}
		}

		// Build proto Struct
		original, err := structpb.NewStruct(fields)
		if err != nil {
			// Some generated keys may conflict — skip
			ht.Assume(false)
			return
		}

		// Marshal
		result, err := MarshalToolResult(original)
		if err != nil {
			ht.Fatalf("MarshalToolResult failed: %v", err)
		}

		// Property: result is never an error
		if result.IsError {
			ht.Fatal("expected IsError=false")
		}

		// Property: content is valid JSON
		var parsed map[string]interface{}
		if err := json.Unmarshal(result.Content, &parsed); err != nil {
			ht.Fatalf("invalid JSON from MarshalToolResult: %v", err)
		}

		// Unmarshal back
		roundtripped := &structpb.Struct{}
		req := ToolRequest{Arguments: result.Content}
		if err := UnmarshalToolInput(req, roundtripped); err != nil {
			ht.Fatalf("UnmarshalToolInput failed: %v", err)
		}

		// Property: roundtripped struct has same number of fields
		if len(roundtripped.Fields) != len(original.Fields) {
			ht.Fatalf("field count mismatch: original=%d, roundtripped=%d",
				len(original.Fields), len(roundtripped.Fields))
		}

		// Property: all original keys exist in roundtripped
		for key := range original.Fields {
			if roundtripped.Fields[key] == nil {
				ht.Fatalf("key %q lost in roundtrip", key)
			}
		}
	}, hegel.WithTestCases(200))
}

func TestProperty_MarshalToolResult_NeverPanics(t *testing.T) {
	hegel.Test(t, func(ht *hegel.T) {
		// Build a struct with edge-case values
		fields := make(map[string]interface{})

		// Add 0-5 fields with various types
		numFields := hegel.Draw(ht, hegel.Integers(0, 5))
		for range numFields {
			key := hegel.Draw(ht, hegel.Text().Alphabet("abcdefghijklmnopqrstuvwxyz").MinSize(1).MaxSize(10))
			fields[key] = hegel.Draw(ht, hegel.Text().MaxSize(50))
		}

		msg, err := structpb.NewStruct(fields)
		if err != nil {
			ht.Assume(false)
			return
		}

		// Property: MarshalToolResult never panics on valid input
		result, err := MarshalToolResult(msg)
		if err != nil {
			ht.Fatalf("unexpected error: %v", err)
		}
		if len(result.Content) == 0 {
			ht.Fatal("expected non-empty content")
		}
	}, hegel.WithTestCases(300))
}

func TestProperty_UnmarshalToolInput_InvalidJSON_AlwaysErrors(t *testing.T) {
	hegel.Test(t, func(ht *hegel.T) {
		// Generate random bytes that are NOT valid JSON
		garbage := hegel.Draw(ht, hegel.Text().Alphabet("abcdefg{[!@#$%^&*").MinSize(1).MaxSize(50))

		dest := &structpb.Struct{}
		req := ToolRequest{Arguments: []byte(garbage)}
		err := UnmarshalToolInput(req, dest)

		// Property: invalid JSON always produces an error
		if err == nil {
			// Unless the garbage happens to be valid JSON for a Struct
			var check map[string]interface{}
			if json.Unmarshal([]byte(garbage), &check) != nil {
				ht.Fatalf("expected error for invalid JSON: %q", garbage)
			}
		}
	}, hegel.WithTestCases(200))
}
