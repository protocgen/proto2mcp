package mcpruntime

import (
	"context"
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

// PromptMessage represents a system, user, or assistant message returned in a prompt template.
type PromptMessage struct {
	Role    string `json:"role"` // "user", "assistant", or "system"
	Content string `json:"content"`
}

// GetPromptResult contains the generated messages for an LLM prompt.
type GetPromptResult struct {
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

// PromptHandlerFunc generates the list of messages for a prompt template given arguments.
type PromptHandlerFunc func(ctx context.Context, arguments map[string]string) (*GetPromptResult, error)

// PromptEntry holds a registered prompt definition and its evaluator/handler.
type PromptEntry struct {
	Definition PromptDefinition
	Handler    PromptHandlerFunc
}

// PromptRegistry manages registered MCP prompt templates.
type PromptRegistry struct {
	mu      sync.RWMutex
	prompts map[string]PromptEntry
}

// NewPromptRegistry creates a new empty PromptRegistry.
func NewPromptRegistry() *PromptRegistry {
	return &PromptRegistry{
		prompts: make(map[string]PromptEntry),
	}
}

// Register adds a prompt template to the registry.
func (r *PromptRegistry) Register(def PromptDefinition, handler PromptHandlerFunc) {
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
func (r *PromptRegistry) EvaluatePrompt(ctx context.Context, name string, arguments map[string]string) (*GetPromptResult, error) {
	handler, ok := r.Lookup(name)
	if !ok {
		return nil, fmt.Errorf("prompt not found: %s", name)
	}
	return handler(ctx, arguments)
}
