package mcpruntime

import (
	"context"
)

type contextKey string

const tenantKey contextKey = "tenant_id"

// TenantExtractorFunc is a function that extracts tenant identity from context.
type TenantExtractorFunc func(ctx context.Context) (string, error)

// TenantFromContext extracts the tenant ID from context. Returns empty string if not set.
func TenantFromContext(ctx context.Context) string {
	if val, ok := ctx.Value(tenantKey).(string); ok {
		return val
	}
	return ""
}

// WithTenant returns a new context with the tenant ID set.
func WithTenant(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantKey, tenantID)
}
