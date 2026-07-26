package mcpruntime

import (
	"context"
	"fmt"
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

// WrapHandler wraps a handler with the configured middleware chain,
// tenant extraction, and panic recovery. The execution order is:
//
//  1. Panic recovery (outermost)
//  2. Tenant extraction (if configured)
//  3. Middleware chain (in registration order)
//  4. Handler (innermost)
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
