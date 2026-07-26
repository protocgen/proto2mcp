package mcpruntime

import (
	"context"
	"net/http"
)

type contextKey string

const tenantKey contextKey = "tenant_id"
const headersKey contextKey = "propagate_headers"

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

// WithHeaders returns a new context carrying HTTP headers to propagate
// to downstream ConnectRPC calls. Typically used in middleware to forward
// authorization tokens, trace IDs, or other request metadata.
func WithHeaders(ctx context.Context, headers http.Header) context.Context {
	return context.WithValue(ctx, headersKey, headers)
}

// HeadersFromContext returns the HTTP headers stored in context, if any.
// Returns nil if no headers were set. Used by generated connect forwarders
// to propagate headers from the MCP context to ConnectRPC requests.
func HeadersFromContext(ctx context.Context) http.Header {
	if val, ok := ctx.Value(headersKey).(http.Header); ok {
		return val
	}
	return nil
}
