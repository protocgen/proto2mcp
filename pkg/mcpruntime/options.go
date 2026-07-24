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
	Middleware      []Middleware
	TenantExtractor TenantExtractorFunc
	ToolNamer       ToolNamerFunc
}

// Option configures a Config.
type Option func(*Config)

// NewConfig creates a Config with defaults and applies options.
func NewConfig(opts ...Option) *Config {
	c := &Config{
		ToolNamer: DefaultToolNamer,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithMiddleware adds middleware to the chain.
func WithMiddleware(mw ...Middleware) Option {
	return func(c *Config) {
		c.Middleware = append(c.Middleware, mw...)
	}
}

// WithTenantExtractor sets the tenant extraction function.
func WithTenantExtractor(fn TenantExtractorFunc) Option {
	return func(c *Config) {
		c.TenantExtractor = fn
	}
}

// WithToolNamer sets a custom tool naming function.
func WithToolNamer(fn ToolNamerFunc) Option {
	return func(c *Config) {
		c.ToolNamer = fn
	}
}

// WrapHandler wraps a handler with the configured middleware chain.
func (c *Config) WrapHandler(toolName string, handler HandlerFunc) HandlerFunc {
	if len(c.Middleware) == 0 {
		return handler
	}
	chain := ChainMiddleware(c.Middleware...)
	return func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
		return chain.HandleToolCall(ctx, req, handler)
	}
}
