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

func TestToolRegistry_LookupDefinition(t *testing.T) {
	reg := NewToolRegistry()

	schema := json.RawMessage(`{"type":"object"}`)
	def := ToolDefinition{
		Name:        "GetPatient",
		Description: "Get a patient by ID",
		InputSchema: schema,
		Annotations: map[string]any{
			"readOnlyHint":    true,
			"destructiveHint": false,
		},
	}
	reg.Register(def, func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
	})

	// Lookup existing tool.
	got, ok := reg.LookupDefinition("GetPatient")
	if !ok {
		t.Fatal("expected tool to be found")
	}
	if got.Name != "GetPatient" {
		t.Fatalf("expected name 'GetPatient', got %q", got.Name)
	}
	if got.Description != "Get a patient by ID" {
		t.Fatalf("expected description 'Get a patient by ID', got %q", got.Description)
	}
	if got.Annotations["readOnlyHint"] != true {
		t.Fatalf("expected readOnlyHint=true, got %v", got.Annotations["readOnlyHint"])
	}
	if string(got.InputSchema) != `{"type":"object"}` {
		t.Fatalf("unexpected InputSchema: %s", got.InputSchema)
	}
}

func TestToolRegistry_LookupDefinition_NotFound(t *testing.T) {
	reg := NewToolRegistry()

	got, ok := reg.LookupDefinition("Nonexistent")
	if ok {
		t.Fatal("expected ok=false for missing tool")
	}
	if got.Name != "" {
		t.Fatalf("expected zero ToolDefinition, got name=%q", got.Name)
	}
}

func TestToolRegistry_LookupDefinition_DefensiveCopy(t *testing.T) {
	reg := NewToolRegistry()

	schema := json.RawMessage(`{"type":"object"}`)
	reg.Register(ToolDefinition{
		Name:        "Tool1",
		InputSchema: schema,
	}, func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
	})

	// Get a copy and mutate it.
	got, _ := reg.LookupDefinition("Tool1")
	got.InputSchema[0] = 'X'

	// Original should be unchanged.
	got2, _ := reg.LookupDefinition("Tool1")
	if got2.InputSchema[0] != '{' {
		t.Fatalf("defensive copy failed: internal InputSchema was mutated, got %q", string(got2.InputSchema))
	}
}

func TestToolRegistry_LookupDefinition_Concurrent(t *testing.T) {
	reg := NewToolRegistry()

	// Pre-register some tools.
	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("Tool_%d", i)
		reg.Register(
			ToolDefinition{Name: name, InputSchema: json.RawMessage(`{}`)},
			func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
				return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
			},
		)
	}

	// Concurrent LookupDefinition + Register should not race.
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func(n int) {
			defer func() { done <- struct{}{} }()
			name := fmt.Sprintf("Tool_%d", n)
			reg.LookupDefinition(name)
			// Also register new tools concurrently.
			reg.Register(
				ToolDefinition{Name: fmt.Sprintf("New_%d", n), InputSchema: json.RawMessage(`{}`)},
				func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
					return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
				},
			)
			reg.LookupDefinition(fmt.Sprintf("New_%d", n))
		}(i)
	}
	for i := 0; i < 10; i++ {
		<-done
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

func TestWrapHandler_DefinitionInjection(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(ToolDefinition{
		Name:        "GetPatient",
		Description: "Get a patient",
		InputSchema: json.RawMessage(`{}`),
		Annotations: map[string]any{
			"readOnlyHint":    true,
			"destructiveHint": false,
		},
	}, func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
	})

	var gotDef *ToolDefinition
	cfg := NewConfig(
		WithToolRegistry(reg),
		WithMiddleware(ToolInterceptorFunc(
			func(ctx context.Context, req ToolRequest, next HandlerFunc) (*CallToolResult, error) {
				gotDef = req.Definition
				return next(ctx, req)
			},
		)),
	)

	handler := func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
	}

	wrapped := cfg.WrapHandler("GetPatient", handler)
	_, err := wrapped(context.Background(), ToolRequest{ToolName: "GetPatient"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotDef == nil {
		t.Fatal("expected Definition to be populated in middleware")
	}
	if gotDef.Name != "GetPatient" {
		t.Fatalf("expected Definition.Name='GetPatient', got %q", gotDef.Name)
	}
	if gotDef.Annotations["readOnlyHint"] != true {
		t.Fatalf("expected readOnlyHint=true, got %v", gotDef.Annotations["readOnlyHint"])
	}
}

func TestWrapHandler_DefinitionInjection_NoRegistry(t *testing.T) {
	var gotDef *ToolDefinition
	cfg := NewConfig(
		WithMiddleware(ToolInterceptorFunc(
			func(ctx context.Context, req ToolRequest, next HandlerFunc) (*CallToolResult, error) {
				gotDef = req.Definition
				return next(ctx, req)
			},
		)),
	)

	handler := func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
	}

	wrapped := cfg.WrapHandler("GetPatient", handler)
	_, _ = wrapped(context.Background(), ToolRequest{ToolName: "GetPatient"})
	if gotDef != nil {
		t.Fatal("expected Definition to be nil when no registry configured")
	}
}

func TestWrapHandler_DefinitionInjection_UnknownTool(t *testing.T) {
	reg := NewToolRegistry()

	var gotDef *ToolDefinition
	cfg := NewConfig(
		WithToolRegistry(reg),
		WithMiddleware(ToolInterceptorFunc(
			func(ctx context.Context, req ToolRequest, next HandlerFunc) (*CallToolResult, error) {
				gotDef = req.Definition
				return next(ctx, req)
			},
		)),
	)

	handler := func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
	}

	wrapped := cfg.WrapHandler("NonexistentTool", handler)
	_, err := wrapped(context.Background(), ToolRequest{ToolName: "NonexistentTool"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotDef != nil {
		t.Fatal("expected Definition to be nil for unknown tool")
	}
}

func TestWrapHandler_ResourceKeyExtraction(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(ToolDefinition{
		Name:         "GetPatient",
		InputSchema:  json.RawMessage(`{}`),
		ResourceKeys: []string{"patient_id", "org_id"},
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

	handler := func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
	}

	wrapped := cfg.WrapHandler("GetPatient", handler)
	args := json.RawMessage(`{"patient_id":"P-123","org_id":"O-456","extra":"ignored"}`)
	_, err := wrapped(context.Background(), ToolRequest{
		ToolName:  "GetPatient",
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotKeys == nil {
		t.Fatal("expected ResourceKeys to be populated")
	}
	if gotKeys["patient_id"] != "P-123" {
		t.Fatalf("expected patient_id='P-123', got %q", gotKeys["patient_id"])
	}
	if gotKeys["org_id"] != "O-456" {
		t.Fatalf("expected org_id='O-456', got %q", gotKeys["org_id"])
	}
	if _, ok := gotKeys["extra"]; ok {
		t.Fatal("unexpected 'extra' in ResourceKeys")
	}
}

func TestWrapHandler_ResourceKeyExtraction_NoKeys(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(ToolDefinition{
		Name:        "ListPatients",
		InputSchema: json.RawMessage(`{}`),
		// No ResourceKeys configured.
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

	handler := func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
	}

	wrapped := cfg.WrapHandler("ListPatients", handler)
	_, _ = wrapped(context.Background(), ToolRequest{
		ToolName:  "ListPatients",
		Arguments: json.RawMessage(`{"page_size":10}`),
	})
	if gotKeys != nil {
		t.Fatal("expected ResourceKeys to be nil when no resource keys configured")
	}
}

func TestWrapHandler_ResourceKeyExtraction_MalformedJSON(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(ToolDefinition{
		Name:         "GetPatient",
		InputSchema:  json.RawMessage(`{}`),
		ResourceKeys: []string{"patient_id"},
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

	handler := func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
	}

	wrapped := cfg.WrapHandler("GetPatient", handler)
	// Malformed JSON — extraction should silently skip.
	_, err := wrapped(context.Background(), ToolRequest{
		ToolName:  "GetPatient",
		Arguments: json.RawMessage(`not valid json`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotKeys != nil {
		t.Fatal("expected ResourceKeys to be nil for malformed JSON")
	}
}

func TestWrapHandler_ResourceKeyExtraction_EmptyArgs(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(ToolDefinition{
		Name:         "GetPatient",
		InputSchema:  json.RawMessage(`{}`),
		ResourceKeys: []string{"patient_id"},
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

	handler := func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
	}

	wrapped := cfg.WrapHandler("GetPatient", handler)

	// Nil args.
	_, _ = wrapped(context.Background(), ToolRequest{ToolName: "GetPatient"})
	if gotKeys != nil {
		t.Fatal("expected nil ResourceKeys for nil Arguments")
	}

	// Empty bytes.
	_, _ = wrapped(context.Background(), ToolRequest{
		ToolName:  "GetPatient",
		Arguments: json.RawMessage(``),
	})
	if gotKeys != nil {
		t.Fatal("expected nil ResourceKeys for empty Arguments")
	}
}

func TestWrapHandler_ResourceKeyExtraction_NonStringValues(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(ToolDefinition{
		Name:         "GetPatient",
		InputSchema:  json.RawMessage(`{}`),
		ResourceKeys: []string{"int_field", "bool_field", "null_field", "obj_field", "str_field"},
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

	handler := func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
	}

	wrapped := cfg.WrapHandler("GetPatient", handler)
	args := json.RawMessage(`{"int_field":42,"bool_field":true,"null_field":null,"obj_field":{"nested":1},"str_field":"ok"}`)
	_, err := wrapped(context.Background(), ToolRequest{
		ToolName:  "GetPatient",
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only str_field should be extracted — non-string values silently skipped.
	if gotKeys == nil {
		t.Fatal("expected ResourceKeys to be populated")
	}
	if gotKeys["str_field"] != "ok" {
		t.Fatalf("expected str_field='ok', got %q", gotKeys["str_field"])
	}
	if _, ok := gotKeys["int_field"]; ok {
		t.Fatal("int_field should not be in ResourceKeys")
	}
	if _, ok := gotKeys["bool_field"]; ok {
		t.Fatal("bool_field should not be in ResourceKeys")
	}
	if _, ok := gotKeys["null_field"]; ok {
		t.Fatal("null_field should not be in ResourceKeys")
	}
	if _, ok := gotKeys["obj_field"]; ok {
		t.Fatal("obj_field should not be in ResourceKeys")
	}
}

func TestWrapHandler_ResourceKeyExtraction_MissingKeyInArgs(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(ToolDefinition{
		Name:         "GetPatient",
		InputSchema:  json.RawMessage(`{}`),
		ResourceKeys: []string{"patient_id", "org_id"},
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

	handler := func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
	}

	wrapped := cfg.WrapHandler("GetPatient", handler)
	// Only patient_id present, org_id missing — should extract partial.
	_, err := wrapped(context.Background(), ToolRequest{
		ToolName:  "GetPatient",
		Arguments: json.RawMessage(`{"patient_id":"P-123"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotKeys == nil {
		t.Fatal("expected ResourceKeys to be populated")
	}
	if gotKeys["patient_id"] != "P-123" {
		t.Fatalf("expected patient_id='P-123', got %q", gotKeys["patient_id"])
	}
	if _, ok := gotKeys["org_id"]; ok {
		t.Fatal("missing org_id should not be in ResourceKeys")
	}
}

func TestToolRegistry_LookupDefinition_AnnotationsDefensiveCopy(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(ToolDefinition{
		Name: "Tool1",
		Annotations: map[string]any{
			"readOnlyHint": true,
		},
	}, func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: json.RawMessage(`{}`), IsError: false}, nil
	})

	// Get a copy and mutate the Annotations map.
	got, _ := reg.LookupDefinition("Tool1")
	got.Annotations["readOnlyHint"] = false
	got.Annotations["injected"] = "evil"

	// Internal state should be unchanged.
	got2, _ := reg.LookupDefinition("Tool1")
	if got2.Annotations["readOnlyHint"] != true {
		t.Fatal("defensive copy failed: readOnlyHint was mutated")
	}
	if _, ok := got2.Annotations["injected"]; ok {
		t.Fatal("defensive copy failed: injected key appeared in internal state")
	}
}

func TestFilteredTools_NoMiddleware(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(ToolDefinition{Name: "tool1"}, func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return nil, nil
	})
	reg.Register(ToolDefinition{Name: "tool2"}, func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return nil, nil
	})

	tools := reg.FilteredTools(context.Background())
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
}

func TestFilteredTools_WithDiscoveryInterceptor(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(ToolDefinition{Name: "GetPatient"}, func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return nil, nil
	})
	reg.Register(ToolDefinition{Name: "DeletePatient"}, func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return nil, nil
	})
	reg.Register(ToolDefinition{Name: "ListPatients"}, func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return nil, nil
	})

	// Filter that only allows read tools.
	readOnlyFilter := DiscoveryInterceptorFunc(func(ctx context.Context, tools []ToolDefinition) []ToolDefinition {
		var filtered []ToolDefinition
		for _, t := range tools {
			if t.Name != "DeletePatient" {
				filtered = append(filtered, t)
			}
		}
		return filtered
	})

	tools := reg.FilteredTools(context.Background(), readOnlyFilter)
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	for _, tool := range tools {
		if tool.Name == "DeletePatient" {
			t.Error("DeletePatient should have been filtered out")
		}
	}
}

func TestFilteredTools_ChainedMiddleware(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(ToolDefinition{Name: "A"}, func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return nil, nil
	})
	reg.Register(ToolDefinition{Name: "B"}, func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return nil, nil
	})
	reg.Register(ToolDefinition{Name: "C"}, func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return nil, nil
	})

	// First filter removes A.
	filterA := DiscoveryInterceptorFunc(func(ctx context.Context, tools []ToolDefinition) []ToolDefinition {
		var filtered []ToolDefinition
		for _, t := range tools {
			if t.Name != "A" {
				filtered = append(filtered, t)
			}
		}
		return filtered
	})
	// Second filter removes B.
	filterB := DiscoveryInterceptorFunc(func(ctx context.Context, tools []ToolDefinition) []ToolDefinition {
		var filtered []ToolDefinition
		for _, t := range tools {
			if t.Name != "B" {
				filtered = append(filtered, t)
			}
		}
		return filtered
	})

	tools := reg.FilteredTools(context.Background(), filterA, filterB)
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0].Name != "C" {
		t.Errorf("expected tool C, got %s", tools[0].Name)
	}
}

func TestFilteredTools_DefensiveCopy(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(ToolDefinition{
		Name:        "tool1",
		Annotations: map[string]any{"readOnlyHint": true},
	}, func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return nil, nil
	})

	// Get filtered tools and mutate annotations.
	tools := reg.FilteredTools(context.Background())
	tools[0].Annotations["injected"] = true

	// Verify original is unchanged.
	tools2 := reg.FilteredTools(context.Background())
	if _, ok := tools2[0].Annotations["injected"]; ok {
		t.Error("FilteredTools should return defensive copies")
	}
}
