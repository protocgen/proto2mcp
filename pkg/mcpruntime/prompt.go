package mcpruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// PromptArgument describes an input parameter for a prompt.
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// PromptDefinition describes an LLM prompt template exposed by the server.
type PromptDefinition struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// PromptContent represents one piece of content in a prompt message.
// MCP spec defines content as structured objects with a type discriminator.
//
// For text content:
//
//	PromptContent{Type: "text", Text: "Hello, world!"}
//
// For embedded resources (V2):
//
//	PromptContent{Type: "resource", Resource: json.RawMessage(`{...}`)}
type PromptContent struct {
	// Type discriminator: "text", "image", or "resource".
	Type string `json:"type"`
	// Text is the text payload when Type is "text".
	Text string `json:"text,omitempty"`
	// Resource is the embedded resource payload when Type is "resource".
	// V2: structured as json.RawMessage for forward compatibility.
	Resource json.RawMessage `json:"resource,omitempty"`
}

// TextContent is a convenience constructor for text prompt content.
func TextContent(text string) PromptContent {
	return PromptContent{Type: "text", Text: text}
}

// PromptMessage represents a single message in a prompt template result.
type PromptMessage struct {
	Role    string        `json:"role"` // "user", "assistant", or "system"
	Content PromptContent `json:"content"`
}

// GetPromptResult contains the generated messages for an LLM prompt.
type GetPromptResult struct {
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

// PromptHandlerFunc generates the list of messages for a prompt template given arguments.
type PromptHandlerFunc func(ctx context.Context, arguments map[string]string) (*GetPromptResult, error)

// PromptRegistrar is the interface accepted by prompt registration functions.
// This enables mocking in tests, distributed registries, and decorator patterns.
type PromptRegistrar interface {
	// RegisterPrompt adds a prompt template to the registry.
	RegisterPrompt(def PromptDefinition, handler PromptHandlerFunc)
}

// PromptEntry holds a registered prompt definition and its evaluator/handler.
type PromptEntry struct {
	Definition PromptDefinition
	Handler    PromptHandlerFunc
}

// PromptRegistry is the default in-memory implementation of PromptRegistrar.
// It manages registered MCP prompt templates with thread-safe access.
type PromptRegistry struct {
	mu      sync.RWMutex
	prompts map[string]PromptEntry
}

// Verify PromptRegistry implements PromptRegistrar at compile time.
var _ PromptRegistrar = (*PromptRegistry)(nil)

// NewPromptRegistry creates a new empty PromptRegistry.
func NewPromptRegistry() *PromptRegistry {
	return &PromptRegistry{
		prompts: make(map[string]PromptEntry),
	}
}

// RegisterPrompt adds a prompt template to the registry.
func (r *PromptRegistry) RegisterPrompt(def PromptDefinition, handler PromptHandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.prompts[def.Name] = PromptEntry{
		Definition: def,
		Handler:    handler,
	}
}

// Lookup retrieves a prompt handler by template name.
func (r *PromptRegistry) Lookup(name string) (PromptHandlerFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.prompts[name]
	if !ok {
		return nil, false
	}
	return entry.Handler, true
}

// Prompts returns all registered prompt definitions.
func (r *PromptRegistry) Prompts() []PromptDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]PromptDefinition, 0, len(r.prompts))
	for _, entry := range r.prompts {
		// Create a copy of arguments slice
		def := entry.Definition
		if def.Arguments != nil {
			argsCopy := make([]PromptArgument, len(def.Arguments))
			copy(argsCopy, def.Arguments)
			def.Arguments = argsCopy
		}
		defs = append(defs, def)
	}
	return defs
}

// EvaluatePrompt calls the prompt handler with the provided arguments.
// Returns an MCPError with code "NOT_FOUND" if the prompt name is not registered.
func (r *PromptRegistry) EvaluatePrompt(ctx context.Context, name string, arguments map[string]string) (*GetPromptResult, error) {
	handler, ok := r.Lookup(name)
	if !ok {
		return nil, NewMCPError("NOT_FOUND", fmt.Sprintf("prompt not found: %s", name))
	}
	return handler(ctx, arguments)
}
