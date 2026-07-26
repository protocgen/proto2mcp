package mcpruntime

import (
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

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

func FuzzSanitizeErrorMessage(f *testing.F) {
	// Seed corpus
	f.Add("simple error")
	f.Add("failed to connect to /Users/admin/secret/config.yaml")
	f.Add("connection refused at 192.168.1.1:5432")
	f.Add("error at C:\\Users\\admin\\Documents\\secret.key")
	f.Add("timeout connecting to database-host.internal:3306")
	f.Add("")
	f.Add("a]]]long string that is over two hundred characters long and should be truncated by the sanitizer to ensure we never leak excessive information in error messages returned to clients which could be a security concern")

	f.Fuzz(func(t *testing.T, msg string) {
		result := SanitizeErrorMessage(msg)

		// Property: result must never exceed 200 characters
		if len(result) > 200 {
			t.Errorf("sanitized message exceeds 200 chars: got %d", len(result))
		}
	})
}
