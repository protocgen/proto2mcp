package mcpruntime

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestPromptRegistry_RegisterAndEvaluate(t *testing.T) {
	registry := NewPromptRegistry()

	def := PromptDefinition{
		Name:        "greet_user",
		Description: "Greets a user by name",
		Arguments: []PromptArgument{
			{Name: "name", Description: "The user's name", Required: true},
		},
	}

	handler := func(ctx context.Context, arguments map[string]string) (*GetPromptResult, error) {
		name := arguments["name"]
		if name == "" {
			name = "Guest"
		}
		return &GetPromptResult{
			Description: "Greeting messages",
			Messages: []PromptMessage{
				{Role: "system", Content: TextContent("You are a helpful greeting bot.")},
				{Role: "user", Content: TextContent("Hello, my name is " + name)},
			},
		}, nil
	}

	registry.RegisterPrompt(def, handler)

	// Test List Prompts
	prompts := registry.Prompts()
	if len(prompts) != 1 {
		t.Fatalf("expected 1 prompt, got %d", len(prompts))
	}
	if prompts[0].Name != "greet_user" {
		t.Errorf("expected name 'greet_user', got %q", prompts[0].Name)
	}

	// Test Lookup
	_, ok := registry.Lookup("greet_user")
	if !ok {
		t.Fatal("expected to find greet_user prompt")
	}

	// Test Evaluation
	res, err := registry.EvaluatePrompt(context.Background(), "greet_user", map[string]string{"name": "Alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(res.Messages))
	}
	if res.Messages[1].Content.Text != "Hello, my name is Alice" {
		t.Errorf("expected content text 'Hello, my name is Alice', got %q", res.Messages[1].Content.Text)
	}
	if res.Messages[1].Content.Type != "text" {
		t.Errorf("expected content type 'text', got %q", res.Messages[1].Content.Type)
	}
}

func TestPromptRegistry_EvaluateNotFound(t *testing.T) {
	registry := NewPromptRegistry()

	_, err := registry.EvaluatePrompt(context.Background(), "non_existent", nil)
	if err == nil {
		t.Fatal("expected error for non-existent prompt")
	}

	// Should be an MCPError with NOT_FOUND code
	var mcpErr *MCPError
	if !errors.As(err, &mcpErr) {
		t.Fatalf("expected MCPError, got %T: %v", err, err)
	}
	if mcpErr.Code != "NOT_FOUND" {
		t.Errorf("expected code NOT_FOUND, got %q", mcpErr.Code)
	}
}

func TestPromptRegistry_OverwriteBehavior(t *testing.T) {
	registry := NewPromptRegistry()

	// Register first handler
	registry.RegisterPrompt(PromptDefinition{
		Name:        "my_prompt",
		Description: "version 1",
	}, func(ctx context.Context, args map[string]string) (*GetPromptResult, error) {
		return &GetPromptResult{Messages: []PromptMessage{
			{Role: "user", Content: TextContent("v1")},
		}}, nil
	})

	// Overwrite with second handler
	registry.RegisterPrompt(PromptDefinition{
		Name:        "my_prompt",
		Description: "version 2",
	}, func(ctx context.Context, args map[string]string) (*GetPromptResult, error) {
		return &GetPromptResult{Messages: []PromptMessage{
			{Role: "user", Content: TextContent("v2")},
		}}, nil
	})

	// List should show only 1 prompt with the latest description
	prompts := registry.Prompts()
	if len(prompts) != 1 {
		t.Fatalf("expected 1 prompt after overwrite, got %d", len(prompts))
	}
	if prompts[0].Description != "version 2" {
		t.Errorf("expected description 'version 2', got %q", prompts[0].Description)
	}

	// Evaluate should use the second handler
	res, err := registry.EvaluatePrompt(context.Background(), "my_prompt", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Messages[0].Content.Text != "v2" {
		t.Errorf("expected v2 content, got %q", res.Messages[0].Content.Text)
	}
}

func TestPromptRegistry_Concurrency(t *testing.T) {
	registry := NewPromptRegistry()
	var wg sync.WaitGroup

	// Pre-register a prompt for concurrent evaluation
	registry.RegisterPrompt(PromptDefinition{Name: "shared"}, func(ctx context.Context, args map[string]string) (*GetPromptResult, error) {
		return &GetPromptResult{Messages: []PromptMessage{
			{Role: "user", Content: TextContent("ok")},
		}}, nil
	})

	// Concurrent writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			registry.RegisterPrompt(PromptDefinition{
				Name: string(rune(id + '0')),
			}, func(ctx context.Context, args map[string]string) (*GetPromptResult, error) {
				return &GetPromptResult{}, nil
			})
		}(i)
	}

	// Concurrent readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = registry.Prompts()
		}()
	}

	// Concurrent evaluators
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = registry.EvaluatePrompt(context.Background(), "shared", nil)
		}()
	}

	wg.Wait()
}

func TestPromptRegistrar_Interface(t *testing.T) {
	// Verify the interface can be used for dependency injection
	var registrar PromptRegistrar = NewPromptRegistry()
	registrar.RegisterPrompt(PromptDefinition{Name: "test"}, func(ctx context.Context, args map[string]string) (*GetPromptResult, error) {
		return &GetPromptResult{}, nil
	})
}
