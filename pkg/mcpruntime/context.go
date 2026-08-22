package mcpruntime

import (
	"context"
	"net/http"
	"strings"
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
//
// SECURITY: Only include headers you explicitly intend to propagate.
// Do NOT pass raw incoming request headers — filter to an allowlist
// first (e.g., Authorization, X-Request-ID, traceparent). The generated
// connect forwarder blindly copies all headers from context to the
// outgoing ConnectRPC request.
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

// DefaultHeaderAllowlist is the default set of headers propagated to
// ConnectRPC backends when no explicit allowlist is configured.
var DefaultHeaderAllowlist = map[string]bool{
	"authorization": true,
	"x-request-id":  true,
	"traceparent":   true,
	"tracestate":    true,
}

// FilterHeaders returns a new http.Header containing only the allowed headers.
// If allowlist is nil, returns the input headers unchanged.
func FilterHeaders(headers http.Header, allowlist map[string]bool) http.Header {
	if allowlist == nil {
		return headers
	}
	filtered := make(http.Header)
	for k, v := range headers {
		if allowlist[strings.ToLower(k)] {
			filtered[k] = v
		}
	}
	return filtered
}
