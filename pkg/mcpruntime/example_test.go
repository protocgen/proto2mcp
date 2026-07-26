package mcpruntime_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/protocgen/proto2mcp/pkg/mcpruntime"
)

// ExampleNewToolRegistry demonstrates creating a registry and registering tools.
func ExampleNewToolRegistry() {
	registry := mcpruntime.NewToolRegistry()

	// Register a tool (typically done by generated code).
	registry.Register(mcpruntime.ToolDefinition{
		Name:        "PatientService_GetPatient",
		Description: "Look up a patient record by ID",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"patient_id":{"type":"string"}}}`),
	}, func(ctx context.Context, req mcpruntime.ToolRequest) (*mcpruntime.CallToolResult, error) {
		return &mcpruntime.CallToolResult{
			Content: json.RawMessage(`{"name":"Jane Doe"}`),
		}, nil
	})

	for _, tool := range registry.Tools() {
		fmt.Printf("%s: %s\n", tool.Name, tool.Description)
	}
	// Output:
	// PatientService_GetPatient: Look up a patient record by ID
}

// ExampleWithTenant demonstrates injecting and reading tenant context.
func ExampleWithTenant() {
	ctx := mcpruntime.WithTenant(context.Background(), "tenant-42")
	fmt.Println(mcpruntime.TenantFromContext(ctx))
	// Output:
	// tenant-42
}

// ExampleWithHeaders demonstrates injecting headers for connect forwarding.
func ExampleWithHeaders() {
	headers := http.Header{}
	headers.Set("Authorization", "Bearer tok_abc123")
	headers.Set("X-Request-ID", "req-789")

	ctx := mcpruntime.WithHeaders(context.Background(), headers)

	// In generated connect forwarder, headers are read and propagated:
	h := mcpruntime.HeadersFromContext(ctx)
	fmt.Println(h.Get("Authorization"))
	// Output:
	// Bearer tok_abc123
}
