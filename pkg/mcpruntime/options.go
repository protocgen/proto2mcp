package mcpruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ToolNamerFunc generates MCP tool names from service and method names.
type ToolNamerFunc func(serviceName, methodName string) string

// DefaultToolNamer returns "ServiceName_MethodName".
func DefaultToolNamer(serviceName, methodName string) string {
	return fmt.Sprintf("%s_%s", serviceName, methodName)
}

// Config holds the configuration for a registered service.
type Config struct {
	middleware      []Middleware
	tenantExtractor TenantExtractorFunc
	toolNamer       ToolNamerFunc
	registry        *ToolRegistry
	headerAllowlist map[string]bool
}

// Option configures a Config.
type Option func(*Config)

// NewConfig creates a Config with defaults and applies options.
func NewConfig(opts ...Option) *Config {
	c := &Config{
		toolNamer: DefaultToolNamer,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithMiddleware adds middleware to the chain.
func WithMiddleware(mw ...Middleware) Option {
	return func(c *Config) {
		c.middleware = append(c.middleware, mw...)
	}
}

// WithTenantExtractor sets the tenant extraction function.
func WithTenantExtractor(fn TenantExtractorFunc) Option {
	return func(c *Config) {
		c.tenantExtractor = fn
	}
}

// WithToolNamer sets a custom tool naming function.
func WithToolNamer(fn ToolNamerFunc) Option {
	return func(c *Config) {
		c.toolNamer = fn
	}
}

// WithToolRegistry configures the registry for ToolDefinition injection.
// When set, WrapHandler populates ToolRequest.Definition before the
// middleware chain runs, enabling interceptors to inspect tool metadata
// (e.g., Annotations with readOnlyHint, destructiveHint).
func WithToolRegistry(r *ToolRegistry) Option {
	return func(c *Config) {
		c.registry = r
	}
}

// WithHeaderAllowlist sets the allowed headers for ConnectRPC forwarding.
// Only headers in this list will be propagated from context to outgoing
// ConnectRPC requests. If not set, a sensible default is used:
// Authorization, X-Request-ID, traceparent, tracestate.
//
// SECURITY: Without an allowlist, all headers from context are forwarded,
// which may leak cookies, internal service mesh headers, or other
// sensitive data to the backend.
func WithHeaderAllowlist(headers ...string) Option {
	return func(c *Config) {
		c.headerAllowlist = make(map[string]bool, len(headers))
		for _, h := range headers {
			c.headerAllowlist[strings.ToLower(h)] = true
		}
	}
}

// HeaderAllowlist returns the configured header allowlist.
// Returns nil if no allowlist is configured (all headers forwarded).
func (c *Config) HeaderAllowlist() map[string]bool {
	return c.headerAllowlist
}

// WrapHandler wraps a handler with the configured middleware chain,
// tenant extraction, and panic recovery. The execution order is:
//
//  1. Panic recovery (outermost)
//  2. Tenant extraction (if configured)
//  3. Definition injection (if registry configured)
//  4. Middleware chain (in registration order)
//  5. Handler (innermost)
func (c *Config) WrapHandler(toolName string, handler HandlerFunc) HandlerFunc {
	// Build middleware chain from innermost to outermost.
	chain := handler
	for i := len(c.middleware) - 1; i >= 0; i-- {
		mw := c.middleware[i]
		next := chain
		chain = func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
			return mw.HandleToolCall(ctx, req, next)
		}
	}

	// Wrap with definition injection and resource key extraction
	// if registry is configured.
	if c.registry != nil {
		inner := chain
		reg := c.registry
		chain = func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
			if def, ok := reg.LookupDefinition(toolName); ok {
				req.Definition = &def
				// Extract resource keys from arguments.
				if len(def.ResourceKeys) > 0 && len(req.Arguments) > 0 {
					var args map[string]json.RawMessage
					if json.Unmarshal(req.Arguments, &args) == nil {
						keys := make(map[string]string, len(def.ResourceKeys))
						for _, k := range def.ResourceKeys {
							if raw, ok := args[k]; ok && string(raw) != "null" {
								var val string
								if json.Unmarshal(raw, &val) == nil {
									keys[k] = val
								}
							}
						}
						if len(keys) > 0 {
							req.ResourceKeys = keys
						}
					}
				}
			}
			return inner(ctx, req)
		}
	}

	// Wrap with tenant extraction if configured.
	if c.tenantExtractor != nil {
		inner := chain
		extractor := c.tenantExtractor
		chain = func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
			tenantID, err := extractor(ctx)
			if err != nil {
				return InternalError("tenant extraction failed"), nil
			}
			if tenantID != "" {
				ctx = WithTenant(ctx, tenantID)
			}
			return inner(ctx, req)
		}
	}

	// Wrap with panic recovery (outermost).
	final := chain
	return func(ctx context.Context, req ToolRequest) (result *CallToolResult, err error) {
		defer func() {
			if r := recover(); r != nil {
				result = InternalError("internal error")
				err = nil
			}
		}()
		return final(ctx, req)
	}
}
