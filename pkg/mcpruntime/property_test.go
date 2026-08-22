package mcpruntime

import (
	"context"
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

// ============================================================================
// Property-Based Tests: Resource Key Extraction
// ============================================================================

func TestProperty_ResourceKeyExtraction_NeverPanics(t *testing.T) {
	hegel.Test(t, func(ht *hegel.T) {
		// Generate random resource key names.
		numKeys := hegel.Draw(ht, hegel.Integers(0, 5))
		keys := make([]string, numKeys)
		for i := range numKeys {
			keys[i] = hegel.Draw(ht, hegel.Text().Alphabet("abcdefghijklmnopqrstuvwxyz_").MinSize(1).MaxSize(15))
		}

		reg := NewToolRegistry()
		reg.Register(ToolDefinition{
			Name:         "TestTool",
			InputSchema:  json.RawMessage(`{}`),
			ResourceKeys: keys,
		}, func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
			return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
		})

		cfg := NewConfig(WithToolRegistry(reg))
		handler := cfg.WrapHandler("TestTool", func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
			return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
		})

		// Generate random argument payloads.
		argType := hegel.Draw(ht, hegel.Integers(0, 4))
		var args json.RawMessage
		switch argType {
		case 0:
			args = nil
		case 1:
			args = json.RawMessage(`{}`)
		case 2:
			// Random garbage.
			args = json.RawMessage(hegel.Draw(ht, hegel.Text().MaxSize(50)))
		case 3:
			// Valid JSON object with random string values.
			fields := make(map[string]string)
			numFields := hegel.Draw(ht, hegel.Integers(0, 5))
			for range numFields {
				k := hegel.Draw(ht, hegel.Text().Alphabet("abcdefghijklmnopqrstuvwxyz_").MinSize(1).MaxSize(10))
				v := hegel.Draw(ht, hegel.Text().MaxSize(20))
				fields[k] = v
			}
			b, _ := json.Marshal(fields)
			args = b
		case 4:
			// Valid JSON with mixed types.
			mixed := map[string]any{
				"str":  "hello",
				"num":  42,
				"bool": true,
				"null": nil,
			}
			b, _ := json.Marshal(mixed)
			args = b
		}

		// Property: WrapHandler NEVER panics regardless of input.
		result, err := handler(context.Background(), ToolRequest{
			ToolName:  "TestTool",
			Arguments: args,
		})

		// Property: no error is returned.
		if err != nil {
			ht.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			ht.Fatal("expected non-nil result")
		}
	}, hegel.WithTestCases(500))
}

func TestProperty_ResourceKeyExtraction_SubsetInvariant(t *testing.T) {
	hegel.Test(t, func(ht *hegel.T) {
		// Generate configured resource key names.
		numKeys := hegel.Draw(ht, hegel.Integers(1, 5))
		configuredKeys := make([]string, numKeys)
		keySet := make(map[string]bool, numKeys)
		for i := range numKeys {
			k := hegel.Draw(ht, hegel.Text().Alphabet("abcdefghijklmnopqrstuvwxyz").MinSize(1).MaxSize(10))
			configuredKeys[i] = k
			keySet[k] = true
		}

		reg := NewToolRegistry()
		reg.Register(ToolDefinition{
			Name:         "TestTool",
			InputSchema:  json.RawMessage(`{}`),
			ResourceKeys: configuredKeys,
		}, func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
			return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
		})

		var gotKeys map[string]string
		cfg := NewConfig(
			WithToolRegistry(reg),
			WithMiddleware(ToolInterceptorFunc(
				func(ctx context.Context, req ToolRequest, next HandlerFunc) (*CallToolResult, error) {
					gotKeys = req.ResourceKeys
					return next(ctx, req)
				},
			)),
		)

		// Generate args with a mix of configured and extra keys.
		fields := make(map[string]string)
		numFields := hegel.Draw(ht, hegel.Integers(1, 8))
		for range numFields {
			k := hegel.Draw(ht, hegel.Text().Alphabet("abcdefghijklmnopqrstuvwxyz").MinSize(1).MaxSize(10))
			v := hegel.Draw(ht, hegel.Text().MaxSize(20))
			fields[k] = v
		}
		argsBytes, _ := json.Marshal(fields)

		handler := cfg.WrapHandler("TestTool", func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
			return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
		})
		_, _ = handler(context.Background(), ToolRequest{
			ToolName:  "TestTool",
			Arguments: argsBytes,
		})

		// Property: every extracted key is in the configured set.
		for k := range gotKeys {
			if !keySet[k] {
				ht.Fatalf("extracted key %q not in configured set %v", k, configuredKeys)
			}
		}

		// Property: every extracted value matches the original args.
		for k, v := range gotKeys {
			if fields[k] != v {
				ht.Fatalf("key %q: extracted %q but args had %q", k, v, fields[k])
			}
		}
	}, hegel.WithTestCases(500))
}
