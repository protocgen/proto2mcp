package mcpruntime

import (
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

func BenchmarkUnmarshalToolInput(b *testing.B) {
	b.ReportAllocs()
	input := []byte(`{"patient_id": "P-12345", "include_history": true, "max_results": 100}`)
	req := ToolRequest{Arguments: input}

	b.ResetTimer()
	for b.Loop() {
		dest := &structpb.Struct{}
		_ = UnmarshalToolInput(req, dest)
	}
}

func BenchmarkMarshalToolResult(b *testing.B) {
	b.ReportAllocs()
	msg, _ := structpb.NewStruct(map[string]interface{}{
		"patient_id":  "P-12345",
		"name":        "John Doe",
		"age":         float64(42),
		"active":      true,
		"medications": []interface{}{"aspirin", "metformin"},
	})

	b.ResetTimer()
	for b.Loop() {
		_, _ = MarshalToolResult(msg)
	}
}

func BenchmarkSanitizeErrorMessage(b *testing.B) {
	b.Run("clean", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = SanitizeErrorMessage("connection timeout")
		}
	})

	b.Run("with_path", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = SanitizeErrorMessage("failed to read /Users/admin/secrets/config.yaml: permission denied")
		}
	})

	b.Run("with_host_port", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = SanitizeErrorMessage("connection refused at 192.168.1.100:5432")
		}
	})

	b.Run("long_message", func(b *testing.B) {
		b.ReportAllocs()
		long := "error: " + string(make([]byte, 500))
		for b.Loop() {
			_ = SanitizeErrorMessage(long)
		}
	})
}
