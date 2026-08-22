package mcpruntime

import (
	"encoding/json"
	"sync"
)

// Registry is the interface accepted by generated RegisterXxxMCP functions.
// This enables mocking in tests, distributed registries, and decorator patterns.
type Registry interface {
	// Register adds a tool to the registry.
	Register(def ToolDefinition, handler HandlerFunc)
}

// ToolEntry holds a registered tool's definition and handler.
type ToolEntry struct {
	Definition ToolDefinition
	Handler    HandlerFunc
}

// ToolRegistry is the default in-memory implementation of Registry.
// It manages registered MCP tools with thread-safe access.
type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]ToolEntry
}

// Verify ToolRegistry implements Registry at compile time.
var _ Registry = (*ToolRegistry)(nil)

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
func (r *ToolRegistry) Lookup(toolName string) (HandlerFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.tools[toolName]
	if !ok {
		return nil, false
	}
	return entry.Handler, true
}

// LookupDefinition returns a tool's definition by name.
// The returned definition is a defensive copy — callers may freely
// mutate the returned InputSchema, Annotations, or ResourceKeys
// without affecting the registry.
// Returns the zero ToolDefinition and false if the tool is not found.
func (r *ToolRegistry) LookupDefinition(toolName string) (ToolDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.tools[toolName]
	if !ok {
		return ToolDefinition{}, false
	}
	def := entry.Definition
	if def.InputSchema != nil {
		schemaCopy := make(json.RawMessage, len(def.InputSchema))
		copy(schemaCopy, def.InputSchema)
		def.InputSchema = schemaCopy
	}
	if def.Annotations != nil {
		annCopy := make(map[string]any, len(def.Annotations))
		for k, v := range def.Annotations {
			annCopy[k] = v
		}
		def.Annotations = annCopy
	}
	if def.ResourceKeys != nil {
		rkCopy := make([]string, len(def.ResourceKeys))
		copy(rkCopy, def.ResourceKeys)
		def.ResourceKeys = rkCopy
	}
	return def, true
}

// Tools returns all registered tool definitions.
// InputSchema fields are defensively copied to prevent callers from
// mutating the registry's internal state.
func (r *ToolRegistry) Tools() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]ToolDefinition, 0, len(r.tools))
	for _, entry := range r.tools {
		def := entry.Definition
		// Defensive copy of json.RawMessage to prevent data races
		// from callers mutating the returned schema bytes.
		if def.InputSchema != nil {
			schemaCopy := make(json.RawMessage, len(def.InputSchema))
			copy(schemaCopy, def.InputSchema)
			def.InputSchema = schemaCopy
		}
		if def.Annotations != nil {
			annCopy := make(map[string]any, len(def.Annotations))
			for k, v := range def.Annotations {
				annCopy[k] = v
			}
			def.Annotations = annCopy
		}
		if def.ResourceKeys != nil {
			rkCopy := make([]string, len(def.ResourceKeys))
			copy(rkCopy, def.ResourceKeys)
			def.ResourceKeys = rkCopy
		}
		defs = append(defs, def)
	}
	return defs
}

// RegisterMacro registers a macro-tool that composes sub-tools.
// V3 seam — V1 implementation stores it like any other tool.
func (r *ToolRegistry) RegisterMacro(def ToolDefinition, handler HandlerFunc) {
	r.Register(def, handler)
}
