package mcpruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

// BenchmarkUnmarshalToolInput measures the cost of deserializing JSON
// tool arguments into a protobuf Struct.
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

// BenchmarkMarshalToolResult measures the cost of serializing a
// protobuf Struct into JSON for MCP tool results.
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

// BenchmarkSanitizeErrorMessage measures the cost of sanitizing error
// messages with sub-benchmarks for clean, path, host:port, and long inputs.
func BenchmarkSanitizeErrorMessage(b *testing.B) {
	b.Run("clean", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = sanitizeErrorMessage("connection timeout")
		}
	})

	b.Run("with_path", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = sanitizeErrorMessage("failed to read /Users/admin/secrets/config.yaml: permission denied")
		}
	})

	b.Run("with_host_port", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = sanitizeErrorMessage("connection refused at 192.168.1.100:5432")
		}
	})

	b.Run("long_message", func(b *testing.B) {
		b.ReportAllocs()
		long := "error: " + string(make([]byte, 500))
		for b.Loop() {
			_ = sanitizeErrorMessage(long)
		}
	})
}

func BenchmarkWrapHandler(b *testing.B) {
	config := NewConfig(
		WithMiddleware(ToolInterceptorFunc(func(ctx context.Context, req ToolRequest, next HandlerFunc) (*CallToolResult, error) {
			return next(ctx, req)
		})),
	)
	handler := config.WrapHandler("TestTool", func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: json.RawMessage(`{}`)}, nil
	})
	req := ToolRequest{ToolName: "TestTool", Arguments: json.RawMessage(`{}`)}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handler(ctx, req)
	}
}

func BenchmarkWrapHandler_WithRegistry(b *testing.B) {
	reg := NewToolRegistry()
	reg.Register(ToolDefinition{
		Name:         "TestTool",
		InputSchema:  json.RawMessage(`{}`),
		ResourceKeys: []string{"id"},
	}, func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: json.RawMessage(`{}`)}, nil
	})
	config := NewConfig(WithToolRegistry(reg))
	handler := config.WrapHandler("TestTool", func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: json.RawMessage(`{}`)}, nil
	})
	req := ToolRequest{
		ToolName:  "TestTool",
		Arguments: json.RawMessage(`{"id": "abc123"}`),
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handler(ctx, req)
	}
}

func BenchmarkFilteredTools(b *testing.B) {
	reg := NewToolRegistry()
	for i := 0; i < 50; i++ {
		name := fmt.Sprintf("Tool_%d", i)
		reg.Register(ToolDefinition{
			Name:        name,
			InputSchema: json.RawMessage(`{}`),
		}, func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
			return &CallToolResult{Content: json.RawMessage(`{}`)}, nil
		})
	}
	filter := DiscoveryInterceptorFunc(func(ctx context.Context, tools []ToolDefinition) []ToolDefinition {
		if len(tools) > 25 {
			return tools[:25]
		}
		return tools
	})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reg.FilteredTools(ctx, filter)
	}
}

func BenchmarkRateLimiter(b *testing.B) {
	rl := NewRateLimiter(1000000, 1000000)
	ctx := WithTenant(context.Background(), "tenant-1")
	handler := func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: json.RawMessage(`{}`)}, nil
	}
	req := ToolRequest{ToolName: "TestTool"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = rl.HandleToolCall(ctx, req, handler)
	}
}

func BenchmarkToolRegistry_Lookup(b *testing.B) {
	reg := NewToolRegistry()
	for i := 0; i < 100; i++ {
		name := fmt.Sprintf("Service_Tool_%d", i)
		reg.Register(ToolDefinition{Name: name, InputSchema: json.RawMessage(`{}`)},
			func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
				return &CallToolResult{Content: json.RawMessage(`{}`)}, nil
			})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reg.Lookup("Service_Tool_50")
	}
}
