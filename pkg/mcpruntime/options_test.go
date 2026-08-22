package mcpruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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

func TestWithToolRegistry(t *testing.T) {
	reg := NewToolRegistry()
	cfg := NewConfig(WithToolRegistry(reg))
	if cfg.registry != reg {
		t.Fatal("expected registry to be set")
	}

	// Nil registry should be accepted.
	cfg2 := NewConfig(WithToolRegistry(nil))
	if cfg2.registry != nil {
		t.Fatal("expected nil registry")
	}

	// Default should be nil.
	cfg3 := NewConfig()
	if cfg3.registry != nil {
		t.Fatal("expected nil registry by default")
	}
}

func TestWithHeaderAllowlist(t *testing.T) {
	cfg := NewConfig(WithHeaderAllowlist("Allowed-Header", "another-header"))

	allowlist := cfg.HeaderAllowlist()
	if len(allowlist) != 2 {
		t.Fatalf("expected 2 headers in allowlist, got %d", len(allowlist))
	}

	if !allowlist["allowed-header"] {
		t.Errorf("expected 'allowed-header' to be in allowlist (lowercased)")
	}
	if !allowlist["another-header"] {
		t.Errorf("expected 'another-header' to be in allowlist")
	}

	// Default config should have nil allowlist
	defaultCfg := NewConfig()
	if defaultCfg.HeaderAllowlist() != nil {
		t.Errorf("expected default config to have nil allowlist")
	}
}

func TestWithResourceKeyValidator(t *testing.T) {
	validator := func(key, value string) error {
		if value == ".." {
			return context.DeadlineExceeded
		}
		return nil
	}
	c := NewConfig(WithResourceKeyValidator(validator))
	if c.resourceKeyValidator == nil {
		t.Error("expected validator to be set")
	}
}

func TestResourceKeyValidator_Integration(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(ToolDefinition{
		Name:         "TestTool",
		ResourceKeys: []string{"safe_id", "bad_id"},
	}, func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		// Verify bad_id was filtered out
		if _, ok := req.ResourceKeys["bad_id"]; ok {
			t.Error("bad_id should have been filtered by validator")
		}
		if v, ok := req.ResourceKeys["safe_id"]; !ok || v != "abc123" {
			t.Errorf("expected safe_id=abc123, got %q", v)
		}
		return &CallToolResult{Content: json.RawMessage(`{}`)}, nil
	})

	config := NewConfig(
		WithToolRegistry(reg),
		WithResourceKeyValidator(func(key, value string) error {
			if strings.Contains(value, "..") {
				return fmt.Errorf("path traversal")
			}
			return nil
		}),
	)

	handler, _ := reg.Lookup("TestTool")
	wrapped := config.WrapHandler("TestTool", handler)

	_, err := wrapped(context.Background(), ToolRequest{
		ToolName:  "TestTool",
		Arguments: json.RawMessage(`{"safe_id": "abc123", "bad_id": "../../etc/passwd"}`),
	})
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}
}
