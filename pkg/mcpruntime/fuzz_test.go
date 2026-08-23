package mcpruntime

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	"google.golang.org/protobuf/types/known/structpb"
)

// FuzzUnmarshalToolInput verifies that UnmarshalToolInput never panics
// regardless of the byte content in ToolRequest.Arguments.
func FuzzUnmarshalToolInput(f *testing.F) {
	// Seed corpus
	f.Add([]byte(`{"key": "value"}`))
	f.Add([]byte(`{"nested": {"a": 1, "b": true}}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`not json`))
	f.Add([]byte(`{"key": null}`))
	f.Add([]byte(`[]`))
	f.Add([]byte(``))
	f.Add([]byte(`{"a": "xxxxxxxxxx"}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		dest := &structpb.Struct{}
		req := ToolRequest{Arguments: data}

		// Must never panic. Error is acceptable for invalid input.
		_ = UnmarshalToolInput(req, dest)
	})
}

// FuzzSanitizeErrorMessage verifies that sanitizeErrorMessage always
// returns a string of at most 200 characters, regardless of input.
func FuzzSanitizeErrorMessage(f *testing.F) {
	// Seed corpus
	f.Add("simple error")
	f.Add("failed to connect to /Users/admin/secret/config.yaml")
	f.Add("connection refused at 192.168.1.1:5432")
	f.Add("error at C:\\Users\\admin\\Documents\\secret.key")
	f.Add("timeout connecting to database-host.internal:3306")
	f.Add("")
	f.Add("a]]]long string that is over two hundred characters long and should be truncated by the sanitizer to ensure we never leak excessive information in error messages returned to clients which could be a security concern")
	f.Add("goroutine 1 [running]: runtime.panic")
	f.Add(string(make([]byte, 1000)))
	f.Add("line1\nline2\ngoroutine")
	f.Add("\x00\xff\xfe")

	f.Fuzz(func(t *testing.T, msg string) {
		result := sanitizeErrorMessage(msg)

		// Property: result must never exceed 200 characters
		if len(result) > 200 {
			t.Errorf("sanitized message exceeds 200 chars: got %d", len(result))
		}
		// Should never contain newlines
		if strings.Contains(result, "\n") {
			t.Errorf("result contains newline")
		}
	})
}

// FuzzTruncateUTF8 ensures truncation never breaks UTF-8 runes.
func FuzzTruncateUTF8(f *testing.F) {
	f.Add("hello", 3)
	f.Add("日本語テスト", 6)
	f.Add("emoji 🎉🎊", 10)
	f.Add("", 0)
	f.Add("abc", 100)

	f.Fuzz(func(t *testing.T, s string, maxBytes int) {
		if maxBytes < 0 {
			return // skip negative
		}
		result := truncateUTF8(s, maxBytes)
		if len(result) > maxBytes {
			t.Errorf("result %d bytes exceeds max %d", len(result), maxBytes)
		}
		if !utf8.ValidString(result) {
			t.Errorf("result is not valid UTF-8: %q", result)
		}
	})
}

// FuzzResourceKeyExtraction verifies that resource key extraction
// never panics regardless of the Arguments content.
func FuzzResourceKeyExtraction(f *testing.F) {
	// Seed corpus with various edge cases.
	f.Add([]byte(`{"patient_id":"P-123"}`))
	f.Add([]byte(`{"patient_id":42}`))
	f.Add([]byte(`{"patient_id":null}`))
	f.Add([]byte(`{"patient_id":true}`))
	f.Add([]byte(`{"patient_id":{"nested":"obj"}}`))
	f.Add([]byte(`{"patient_id":["array"]}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`[]`))
	f.Add([]byte(`not json`))
	f.Add([]byte(``))
	f.Add([]byte(`{"patient_id":"","org_id":"O-1"}`))
	f.Add([]byte{0xff})

	f.Fuzz(func(t *testing.T, data []byte) {
		reg := NewToolRegistry()
		reg.Register(ToolDefinition{
			Name:         "FuzzTool",
			InputSchema:  []byte(`{}`),
			ResourceKeys: []string{"patient_id", "org_id"},
		}, func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
			return &CallToolResult{Content: []byte(`{}`), IsError: false}, nil
		})

		cfg := NewConfig(WithToolRegistry(reg))
		handler := cfg.WrapHandler("FuzzTool", func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
			return &CallToolResult{Content: []byte(`{}`), IsError: false}, nil
		})

		// Must never panic.
		_, _ = handler(context.Background(), ToolRequest{
			ToolName:  "FuzzTool",
			Arguments: data,
		})
	})
}
