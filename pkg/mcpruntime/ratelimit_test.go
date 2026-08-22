package mcpruntime

import (
	"context"
	"encoding/json"
	"testing"
)

func TestRateLimiter_AllowsWithinBurst(t *testing.T) {
	rl := NewRateLimiter(10, 5) // 10/sec, burst 5
	ctx := WithTenant(context.Background(), "tenant-1")

	called := 0
	handler := func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		called++
		return &CallToolResult{Content: json.RawMessage(`{}`)}, nil
	}

	for i := 0; i < 5; i++ {
		result, err := rl.HandleToolCall(ctx, ToolRequest{ToolName: "test"}, handler)
		if err != nil {
			t.Fatalf("call %d failed: %v", i, err)
		}
		if result.IsError {
			t.Fatalf("call %d rate limited unexpectedly", i)
		}
	}
	if called != 5 {
		t.Errorf("expected 5 calls, got %d", called)
	}
}

func TestRateLimiter_RejectsOverBurst(t *testing.T) {
	rl := NewRateLimiter(0.1, 2) // very slow refill, burst 2
	ctx := WithTenant(context.Background(), "tenant-1")

	handler := func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: json.RawMessage(`{}`)}, nil
	}

	// First 2 should succeed
	for i := 0; i < 2; i++ {
		result, _ := rl.HandleToolCall(ctx, ToolRequest{ToolName: "test"}, handler)
		if result.IsError {
			t.Fatalf("call %d should not be rate limited", i)
		}
	}

	// 3rd should be rate limited
	result, _ := rl.HandleToolCall(ctx, ToolRequest{ToolName: "test"}, handler)
	if !result.IsError {
		t.Error("expected rate limit error")
	}
}

func TestRateLimiter_SeparatesTenants(t *testing.T) {
	rl := NewRateLimiter(0.1, 1) // burst 1

	handler := func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return &CallToolResult{Content: json.RawMessage(`{}`)}, nil
	}

	// Tenant 1 uses their one token
	ctx1 := WithTenant(context.Background(), "t1")
	result, _ := rl.HandleToolCall(ctx1, ToolRequest{ToolName: "test"}, handler)
	if result.IsError {
		t.Error("t1 first call should succeed")
	}

	// Tenant 2 should still have their own token
	ctx2 := WithTenant(context.Background(), "t2")
	result, _ = rl.HandleToolCall(ctx2, ToolRequest{ToolName: "test"}, handler)
	if result.IsError {
		t.Error("t2 first call should succeed (separate bucket)")
	}
}

func TestRateLimiter_ImplementsMiddleware(t *testing.T) {
	var _ Middleware = (*RateLimiter)(nil)
}
