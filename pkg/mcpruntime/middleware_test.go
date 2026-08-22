package mcpruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

func TestChainMiddleware_ExecutionOrder(t *testing.T) {
	var order []string

	mw1 := ToolInterceptorFunc(func(ctx context.Context, req ToolRequest, next HandlerFunc) (*CallToolResult, error) {
		order = append(order, "mw1-before")
		result, err := next(ctx, req)
		order = append(order, "mw1-after")
		return result, err
	})

	mw2 := ToolInterceptorFunc(func(ctx context.Context, req ToolRequest, next HandlerFunc) (*CallToolResult, error) {
		order = append(order, "mw2-before")
		result, err := next(ctx, req)
		order = append(order, "mw2-after")
		return result, err
	})

	chain := ChainMiddleware(mw1, mw2)

	handler := func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		order = append(order, "handler")
		return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
	}

	_, err := chain.HandleToolCall(context.Background(), ToolRequest{ToolName: "test"}, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d calls, got %d: %v", len(expected), len(order), order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Fatalf("expected order[%d]=%q, got %q. Full: %v", i, v, order[i], order)
		}
	}
}

func TestChainMiddleware_FilterTools(t *testing.T) {
	// First middleware removes tools with "admin" prefix
	mw1 := DiscoveryInterceptorFunc(func(ctx context.Context, tools []ToolDefinition) []ToolDefinition {
		var filtered []ToolDefinition
		for _, tool := range tools {
			if len(tool.Name) < 5 || tool.Name[:5] != "admin" {
				filtered = append(filtered, tool)
			}
		}
		return filtered
	})

	// Second middleware removes tools with "internal" prefix
	mw2 := DiscoveryInterceptorFunc(func(ctx context.Context, tools []ToolDefinition) []ToolDefinition {
		var filtered []ToolDefinition
		for _, tool := range tools {
			if len(tool.Name) < 8 || tool.Name[:8] != "internal" {
				filtered = append(filtered, tool)
			}
		}
		return filtered
	})

	chain := ChainMiddleware(mw1, mw2)

	tools := []ToolDefinition{
		{Name: "getPatient"},
		{Name: "admin_deleteAll"},
		{Name: "internal_debug"},
		{Name: "listPatients"},
	}

	filtered := chain.FilterTools(context.Background(), tools)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 tools after filtering, got %d: %v", len(filtered), filtered)
	}
	if filtered[0].Name != "getPatient" || filtered[1].Name != "listPatients" {
		t.Fatalf("unexpected filtered tools: %v", filtered)
	}
}

func TestToolInterceptorFunc_PassthroughFilter(t *testing.T) {
	mw := ToolInterceptorFunc(func(ctx context.Context, req ToolRequest, next HandlerFunc) (*CallToolResult, error) {
		return next(ctx, req)
	})

	tools := []ToolDefinition{
		{Name: "a"}, {Name: "b"}, {Name: "c"},
	}
	filtered := mw.FilterTools(context.Background(), tools)
	if len(filtered) != 3 {
		t.Fatalf("ToolInterceptorFunc.FilterTools should pass through all tools, got %d", len(filtered))
	}
}

func TestDiscoveryInterceptorFunc_PassthroughCall(t *testing.T) {
	called := false
	mw := DiscoveryInterceptorFunc(func(ctx context.Context, tools []ToolDefinition) []ToolDefinition {
		return tools
	})

	handler := func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		called = true
		return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
	}

	_, err := mw.HandleToolCall(context.Background(), ToolRequest{ToolName: "test"}, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("DiscoveryInterceptorFunc.HandleToolCall should pass through to handler")
	}
}

func TestChainMiddleware_Empty(t *testing.T) {
	chain := ChainMiddleware()
	called := false

	handler := func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		called = true
		return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
	}

	_, err := chain.HandleToolCall(context.Background(), ToolRequest{ToolName: "test"}, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("empty chain should call handler directly")
	}
}

func TestChainMiddleware_ShortCircuit(t *testing.T) {
	authMw := ToolInterceptorFunc(func(ctx context.Context, req ToolRequest, next HandlerFunc) (*CallToolResult, error) {
		// Short-circuit: reject without calling next
		return &CallToolResult{
			Content: json.RawMessage(`{"error":"unauthorized"}`),
			IsError: true,
		}, nil
	})

	handlerCalled := false
	handler := func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		handlerCalled = true
		return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
	}

	chain := ChainMiddleware(authMw)
	result, err := chain.HandleToolCall(context.Background(), ToolRequest{ToolName: "test"}, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result from short-circuit")
	}
	if handlerCalled {
		t.Fatal("handler should NOT have been called after auth short-circuit")
	}
}

func TestWrapHandler(t *testing.T) {
	var intercepted bool
	mw := ToolInterceptorFunc(func(ctx context.Context, req ToolRequest, next HandlerFunc) (*CallToolResult, error) {
		intercepted = true
		return next(ctx, req)
	})

	cfg := NewConfig(WithMiddleware(mw))
	handler := func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: json.RawMessage(`{"ok":true}`), IsError: false}, nil
	}

	wrapped := cfg.WrapHandler("test", handler)
	result, err := wrapped(context.Background(), ToolRequest{ToolName: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !intercepted {
		t.Fatal("middleware was not invoked")
	}
	if result.IsError {
		t.Fatal("expected non-error result")
	}
}

func TestWrapHandler_NoMiddleware(t *testing.T) {
	cfg := NewConfig()
	called := false
	handler := func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		called = true
		return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
	}

	wrapped := cfg.WrapHandler("test", handler)
	_, err := wrapped(context.Background(), ToolRequest{ToolName: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("handler should be called directly with no middleware")
	}
}

func TestDefaultToolNamer(t *testing.T) {
	tests := []struct {
		service, method, want string
	}{
		{"PatientService", "GetPatient", "PatientService_GetPatient"},
		{"BillingService", "CreateInvoice", "BillingService_CreateInvoice"},
		{"Svc", "M", "Svc_M"},
	}
	for _, tt := range tests {
		got := DefaultToolNamer(tt.service, tt.method)
		if got != tt.want {
			t.Errorf("DefaultToolNamer(%q, %q) = %q, want %q", tt.service, tt.method, got, tt.want)
		}
	}
}

func TestToolRegistry(t *testing.T) {
	reg := NewToolRegistry()

	handler := func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
	}

	def := ToolDefinition{Name: "GetPatient", Description: "Get a patient", InputSchema: json.RawMessage(`{}`)}
	reg.Register(def, handler)

	// Lookup existing tool.
	h, ok := reg.Lookup("GetPatient")
	if !ok {
		t.Fatal("expected tool to be found")
	}
	if h == nil {
		t.Fatal("expected non-nil handler")
	}

	// Lookup missing tool.
	_, ok = reg.Lookup("Nonexistent")
	if ok {
		t.Fatal("expected ok=false for missing tool")
	}

	// List all tools.
	tools := reg.Tools()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0].Name != "GetPatient" {
		t.Fatalf("expected tool name 'GetPatient', got %q", tools[0].Name)
	}
}

func TestToolRegistry_RegisterMacro(t *testing.T) {
	reg := NewToolRegistry()

	handler := func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
	}

	def := ToolDefinition{Name: "OnboardPatient", Description: "Macro: onboard patient"}
	reg.RegisterMacro(def, handler)

	// Should be findable via Lookup.
	h, ok := reg.Lookup("OnboardPatient")
	if !ok {
		t.Fatal("expected macro tool to be found")
	}
	if h == nil {
		t.Fatal("expected non-nil handler")
	}

	// Should appear in Tools list.
	tools := reg.Tools()
	if len(tools) != 1 || tools[0].Name != "OnboardPatient" {
		t.Fatalf("unexpected tools: %v", tools)
	}
}

func TestToolRegistry_ConcurrentAccess(t *testing.T) {
	reg := NewToolRegistry()

	// Concurrent registrations should not panic.
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func(n int) {
			defer func() { done <- struct{}{} }()
			name := fmt.Sprintf("Tool_%d", n)
			reg.Register(
				ToolDefinition{Name: name},
				func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
					return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
				},
			)
			reg.Lookup(name)
			reg.Tools()
		}(i)
	}
	for i := 0; i < 10; i++ {
		<-done
	}

	tools := reg.Tools()
	if len(tools) != 10 {
		t.Fatalf("expected 10 tools after concurrent registration, got %d", len(tools))
	}
}

func TestWrapHandler_TenantExtraction(t *testing.T) {
	// C1: Verify that tenantExtractor is actually invoked and context is populated.
	var gotTenant string
	cfg := NewConfig(WithTenantExtractor(func(ctx context.Context) (string, error) {
		return "tenant-42", nil
	}))

	handler := func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		gotTenant = TenantFromContext(ctx)
		return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
	}

	wrapped := cfg.WrapHandler("test", handler)
	result, err := wrapped(context.Background(), ToolRequest{ToolName: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected non-error result")
	}
	if gotTenant != "tenant-42" {
		t.Fatalf("TenantFromContext = %q, want %q", gotTenant, "tenant-42")
	}
}

func TestWrapHandler_TenantExtractionError(t *testing.T) {
	cfg := NewConfig(WithTenantExtractor(func(ctx context.Context) (string, error) {
		return "", fmt.Errorf("auth failed")
	}))

	handlerCalled := false
	handler := func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		handlerCalled = true
		return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
	}

	wrapped := cfg.WrapHandler("test", handler)
	result, err := wrapped(context.Background(), ToolRequest{ToolName: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result when tenant extraction fails")
	}
	if handlerCalled {
		t.Fatal("handler should NOT be called when tenant extraction fails")
	}
}

func TestWrapHandler_PanicRecovery(t *testing.T) {
	// C2: Verify that panics in handlers are recovered and returned as InternalError.
	cfg := NewConfig()
	handler := func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		panic("something went terribly wrong")
	}

	wrapped := cfg.WrapHandler("test", handler)
	result, err := wrapped(context.Background(), ToolRequest{ToolName: "test"})
	if err != nil {
		t.Fatalf("unexpected error (panic should be recovered): %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result from recovered panic")
	}
}

func TestWrapHandler_PanicInMiddleware(t *testing.T) {
	mw := ToolInterceptorFunc(func(ctx context.Context, req ToolRequest, next HandlerFunc) (*CallToolResult, error) {
		panic("middleware panic")
	})

	cfg := NewConfig(WithMiddleware(mw))
	handler := func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
	}

	wrapped := cfg.WrapHandler("test", handler)
	result, err := wrapped(context.Background(), ToolRequest{ToolName: "test"})
	if err != nil {
		t.Fatalf("unexpected error (panic should be recovered): %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result from recovered middleware panic")
	}
}

func TestWrapHandler_TenantWithMiddleware(t *testing.T) {
	// Verify tenant is available inside middleware.
	var tenantInMW string
	mw := ToolInterceptorFunc(func(ctx context.Context, req ToolRequest, next HandlerFunc) (*CallToolResult, error) {
		tenantInMW = TenantFromContext(ctx)
		return next(ctx, req)
	})

	cfg := NewConfig(
		WithTenantExtractor(func(ctx context.Context) (string, error) {
			return "tenant-99", nil
		}),
		WithMiddleware(mw),
	)

	handler := func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
	}

	wrapped := cfg.WrapHandler("test", handler)
	_, err := wrapped(context.Background(), ToolRequest{ToolName: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tenantInMW != "tenant-99" {
		t.Fatalf("tenant in middleware = %q, want %q", tenantInMW, "tenant-99")
	}
}

func TestMarshalToolResult_NilMessage(t *testing.T) {
	// M5: Verify nil message doesn't panic.
	result, err := MarshalToolResult(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("nil message should return non-error result")
	}
	if string(result.Content) != `{}` {
		t.Fatalf("expected empty JSON object, got %s", result.Content)
	}
}
