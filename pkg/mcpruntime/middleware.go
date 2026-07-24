package mcpruntime

import (
	"context"
	"encoding/json"
)

// ToolRequest represents an incoming MCP tool call.
type ToolRequest struct {
	ToolName  string
	Arguments json.RawMessage
}

// ToolDefinition describes an MCP tool for listing.
type ToolDefinition struct {
	Name        string
	Description string
	InputSchema json.RawMessage
	IsResource  bool // V2: true if this is an MCP Resource
}

// HandlerFunc is the function signature for tool call handlers.
type HandlerFunc func(ctx context.Context, req ToolRequest) (*CallToolResult, error)

// CallToolResult wraps the MCP tool call result.
type CallToolResult struct {
	Content json.RawMessage
	IsError bool
}

// Middleware intercepts tool calls for auth, logging, metrics, etc.
type Middleware interface {
	// HandleToolCall wraps a tool invocation. Implementations should call next to proceed.
	HandleToolCall(ctx context.Context, req ToolRequest, next HandlerFunc) (*CallToolResult, error)
	// FilterTools filters the list of tools visible to the caller.
	// V1 default: returns all tools unchanged.
	// V2: per-tenant filtering based on context.
	FilterTools(ctx context.Context, tools []ToolDefinition) []ToolDefinition
}

// MiddlewareFunc is an adapter for simple middleware that only wraps HandleToolCall.
type MiddlewareFunc func(ctx context.Context, req ToolRequest, next HandlerFunc) (*CallToolResult, error)

// HandleToolCall delegates to the underlying function.
func (f MiddlewareFunc) HandleToolCall(ctx context.Context, req ToolRequest, next HandlerFunc) (*CallToolResult, error) {
	return f(ctx, req, next)
}

// FilterTools is a pass-through for MiddlewareFunc.
func (f MiddlewareFunc) FilterTools(ctx context.Context, tools []ToolDefinition) []ToolDefinition {
	return tools
}

// ChainMiddleware chains multiple middleware in order (first middleware is outermost).
func ChainMiddleware(middlewares ...Middleware) Middleware {
	return &middlewareChain{middlewares: middlewares}
}

type middlewareChain struct {
	middlewares []Middleware
}

func (c *middlewareChain) HandleToolCall(ctx context.Context, req ToolRequest, next HandlerFunc) (*CallToolResult, error) {
	// Build the chain from innermost to outermost
	chain := next
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		mw := c.middlewares[i]
		nextMw := chain
		chain = func(ctx context.Context, req ToolRequest) (*CallToolResult, error) {
			return mw.HandleToolCall(ctx, req, nextMw)
		}
	}
	return chain(ctx, req)
}

func (c *middlewareChain) FilterTools(ctx context.Context, tools []ToolDefinition) []ToolDefinition {
	filtered := tools
	for _, mw := range c.middlewares {
		filtered = mw.FilterTools(ctx, filtered)
	}
	return filtered
}
