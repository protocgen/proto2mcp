package mcpruntime

import (
	"context"
	"encoding/json"
)

// ToolRequest represents an incoming MCP tool call.
//
// MCP 2026-07-28: every request carries _meta with protocol version
// and client capabilities. The Meta field passes this through so
// middleware can inspect it without parsing transport headers.
type ToolRequest struct {
	ToolName  string
	Arguments json.RawMessage
	// Meta carries the MCP _meta object from the request.
	// Required fields: protocol version, client capabilities.
	// Optional: client info.
	// MCP 2026-07-28: present on every request in the stateless protocol.
	// Validation of _meta contents belongs at the transport boundary.
	Meta json.RawMessage `json:"_meta,omitempty"`
	// Definition holds the tool's metadata, populated before the middleware
	// chain runs when a ToolRegistry is configured via WithToolRegistry.
	// Middleware can inspect Annotations (e.g., readOnlyHint, destructiveHint)
	// without a separate lookup. Nil when no ToolRegistry is configured.
	Definition *ToolDefinition `json:"-"`
	// ResourceKeys are extracted resource identifiers from the tool arguments.
	// SECURITY: These values are untrusted agent input — validate format
	// before using in authorization decisions.
	ResourceKeys map[string]string `json:"-"`
}

// ToolDefinition describes an MCP tool for listing.
//
// MCP 2026-07-28: tools/list responses are cacheable. Use CacheTTLMs
// and CacheScope to control how clients and infrastructure cache the
// tool list.
type ToolDefinition struct {
	Name        string
	Description string
	InputSchema json.RawMessage
	// Deprecated: Use Annotations with key "resourceHint" instead.
	// Kept for backward compatibility.
	IsResource bool `json:"-"`
	// Annotations holds tool metadata hints per MCP spec.
	// Common keys: "readOnlyHint" (bool, default false),
	// "destructiveHint" (bool, default true when omitted),
	// "idempotentHint" (bool), "openWorldHint" (bool).
	Annotations map[string]any `json:"annotations,omitempty"`
	// ResourceKeys lists JSON field names to extract as resource identifiers.
	// Set by generated code from proto field annotations (resource_key = true).
	// Used by WrapHandler to populate ToolRequest.ResourceKeys.
	ResourceKeys []string `json:"-"`
	// ResourceURI is the URI template for tools that expose MCP Resources (V2).
	// Template variables use {field_name} syntax, e.g. "patient://{patient_id}".
	// Set by generated code from proto method annotations.
	ResourceURI string `json:"-"`
	// Steps defines macro-tool sub-steps for sequential execution (V3 experimental).
	// When non-empty, the tool is treated as a macro that orchestrates other tools.
	Steps []MacroStep `json:"-"`
}

// MacroStep defines a step in a macro-tool execution (V3 experimental).
type MacroStep struct {
	ToolName  string
	OutputKey string
}

// HandlerFunc is the function signature for tool call handlers.
type HandlerFunc func(ctx context.Context, req ToolRequest) (*CallToolResult, error)

// CallToolResult wraps the MCP tool call result.
// Content is a json.RawMessage to allow pass-through of any content
// structure the MCP SDK expects (text, image, resource embeds).
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

// DiscoveryInterceptorFunc is an adapter for middleware that only filters
// tool visibility without intercepting tool calls. It implements the full
// Middleware interface with a pass-through HandleToolCall.
type DiscoveryInterceptorFunc func(ctx context.Context, tools []ToolDefinition) []ToolDefinition

// HandleToolCall is a pass-through — DiscoveryInterceptorFunc does not intercept calls.
func (f DiscoveryInterceptorFunc) HandleToolCall(ctx context.Context, req ToolRequest, next HandlerFunc) (*CallToolResult, error) {
	return next(ctx, req)
}

// FilterTools delegates to the underlying function.
func (f DiscoveryInterceptorFunc) FilterTools(ctx context.Context, tools []ToolDefinition) []ToolDefinition {
	return f(ctx, tools)
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
