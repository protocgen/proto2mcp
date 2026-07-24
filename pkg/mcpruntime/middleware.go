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

// ToolInterceptor intercepts tool call invocations for authorization,
// logging, metrics, tenant injection, etc.
//
// Implementations should call next to proceed with the tool call,
// or return early to short-circuit (e.g., for auth failures).
type ToolInterceptor interface {
	// HandleToolCall wraps a tool invocation. Call next to proceed.
	HandleToolCall(ctx context.Context, req ToolRequest, next HandlerFunc) (*CallToolResult, error)
}

// DiscoveryInterceptor filters the list of tools visible to the caller.
// This is a separate concern from tool invocation — it controls what
// tools appear in the MCP tools/list response.
//
// V1: Default implementation returns all tools unchanged.
// V2: Per-tenant filtering based on context (e.g., role-based tool visibility).
type DiscoveryInterceptor interface {
	// FilterTools filters tools visible to the caller. Return a subset
	// of the input slice to hide tools from the current context.
	FilterTools(ctx context.Context, tools []ToolDefinition) []ToolDefinition
}

// Middleware combines both ToolInterceptor and DiscoveryInterceptor.
// Implement this interface when your middleware needs to control both
// tool execution and tool visibility (e.g., a full AuthZ middleware).
//
// For middleware that only wraps tool calls (logging, metrics, tracing),
// use ToolInterceptorFunc instead.
type Middleware interface {
	ToolInterceptor
	DiscoveryInterceptor
}

// ToolInterceptorFunc is an adapter for simple middleware that only wraps
// tool calls without filtering tool visibility. It implements the full
// Middleware interface with a pass-through FilterTools.
type ToolInterceptorFunc func(ctx context.Context, req ToolRequest, next HandlerFunc) (*CallToolResult, error)

// HandleToolCall delegates to the underlying function.
func (f ToolInterceptorFunc) HandleToolCall(ctx context.Context, req ToolRequest, next HandlerFunc) (*CallToolResult, error) {
	return f(ctx, req, next)
}

// FilterTools is a pass-through — ToolInterceptorFunc does not filter tools.
func (f ToolInterceptorFunc) FilterTools(_ context.Context, tools []ToolDefinition) []ToolDefinition {
	return tools
}

// ChainMiddleware chains multiple middleware in order (first middleware is outermost).
// Each middleware's HandleToolCall wraps the next, and FilterTools is applied
// sequentially across all middleware.
func ChainMiddleware(middlewares ...Middleware) Middleware {
	return &middlewareChain{middlewares: middlewares}
}

type middlewareChain struct {
	middlewares []Middleware
}

func (c *middlewareChain) HandleToolCall(ctx context.Context, req ToolRequest, next HandlerFunc) (*CallToolResult, error) {
	// Build the chain from innermost to outermost.
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
