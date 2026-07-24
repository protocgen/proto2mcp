package mcpruntime

import (
	"sync"
)

// ToolEntry holds a registered tool's definition and handler.
type ToolEntry struct {
	Definition ToolDefinition
	Handler    HandlerFunc
}

// ToolRegistry manages registered MCP tools.
// V1: Tools are registered here, then added to the MCP server.
// V3: Macro-tools use Lookup to compose sub-tools.
type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]ToolEntry
}

// NewToolRegistry creates a new empty registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]ToolEntry),
	}
}

// Register adds a tool to the registry.
func (r *ToolRegistry) Register(def ToolDefinition, handler HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[def.Name] = ToolEntry{
		Definition: def,
		Handler:    handler,
	}
}

// Lookup returns a tool's handler by name. Returns nil, false if not found.
// V3: Used by macro-tools to compose sub-tool calls.
func (r *ToolRegistry) Lookup(toolName string) (HandlerFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.tools[toolName]
	if !ok {
		return nil, false
	}
	return entry.Handler, true
}

// Tools returns all registered tool definitions.
func (r *ToolRegistry) Tools() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]ToolDefinition, 0, len(r.tools))
	for _, entry := range r.tools {
		defs = append(defs, entry.Definition)
	}
	return defs
}

// RegisterMacro registers a macro-tool that composes sub-tools.
// V3 seam — V1 implementation stores it like any other tool.
func (r *ToolRegistry) RegisterMacro(def ToolDefinition, handler HandlerFunc) {
	r.Register(def, handler)
}
