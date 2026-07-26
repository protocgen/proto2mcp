package mcpruntime

import (
	"context"
	"testing"
)

func TestTenantRoundtrip(t *testing.T) {
	ctx := context.Background()

	// Initially empty.
	if got := TenantFromContext(ctx); got != "" {
		t.Fatalf("expected empty tenant, got %q", got)
	}

	// Set and retrieve empty string
	ctxEmpty := WithTenant(ctx, "")
	if got := TenantFromContext(ctxEmpty); got != "" {
		t.Fatalf("expected empty tenant, got %q", got)
	}

	// Set and retrieve.
	ctxAbc := WithTenant(ctx, "tenant-abc")
	if got := TenantFromContext(ctxAbc); got != "tenant-abc" {
		t.Fatalf("expected 'tenant-abc', got %q", got)
	}

	// Parent unchanged.
	if got := TenantFromContext(ctx); got != "" {
		t.Fatalf("expected empty tenant for parent context, got %q", got)
	}
}
