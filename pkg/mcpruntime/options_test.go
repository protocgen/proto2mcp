package mcpruntime

import (
	"context"
	"testing"
)

func TestNewConfig_Defaults(t *testing.T) {
	cfg := NewConfig()
	if cfg.toolNamer == nil {
		t.Fatal("expected default toolNamer to be set")
	}
	name := cfg.toolNamer("Svc", "Method")
	if name != "Svc_Method" {
		t.Fatalf("expected 'Svc_Method', got %q", name)
	}
	if len(cfg.middleware) != 0 {
		t.Fatalf("expected no middleware, got %d", len(cfg.middleware))
	}
	if cfg.tenantExtractor != nil {
		t.Fatal("expected no tenant extractor")
	}
}

func TestNewConfig_WithOptions(t *testing.T) {
	mw := ToolInterceptorFunc(func(ctx context.Context, req ToolRequest, next HandlerFunc) (*CallToolResult, error) {
		return next(ctx, req)
	})
	tenantExt := func(ctx context.Context) (string, error) { return "tenant1", nil }
	namer := func(s, m string) string { return s + m }

	cfg := NewConfig(
		WithMiddleware(mw),
		WithTenantExtractor(tenantExt),
		WithToolNamer(namer),
	)

	if len(cfg.middleware) != 1 {
		t.Fatal("expected middleware to be set")
	}

	if cfg.tenantExtractor == nil {
		t.Fatal("expected tenant extractor to be set")
	}

	if cfg.toolNamer("A", "B") != "AB" {
		t.Fatalf("expected custom tool namer, got %q", cfg.toolNamer("A", "B"))
	}
}

func TestWithToolNamer(t *testing.T) {
	namer := func(svc, method string) string {
		return svc + "." + method
	}
	cfg := NewConfig(WithToolNamer(namer))
	got := cfg.toolNamer("Foo", "Bar")
	if got != "Foo.Bar" {
		t.Fatalf("expected 'Foo.Bar', got %q", got)
	}
}

