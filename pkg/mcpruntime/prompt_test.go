package mcpruntime

import (
	"context"
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
				{Role: "system", Content: "You are a helpful greeting bot."},
				{Role: "user", Content: "Hello, my name is " + name},
			},
		}, nil
	}

	registry.Register(def, handler)

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
	if res.Messages[1].Content != "Hello, my name is Alice" {
		t.Errorf("expected content 'Hello, my name is Alice', got %q", res.Messages[1].Content)
	}

	// Test Evaluation for missing prompt
	_, err = registry.EvaluatePrompt(context.Background(), "non_existent", nil)
	if err == nil {
		t.Error("expected error for non-existent prompt")
	}
}

func TestPromptRegistry_Concurrency(t *testing.T) {
	registry := NewPromptRegistry()
	var wg sync.WaitGroup

	// Run concurrent writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			registry.Register(PromptDefinition{
				Name: string(rune(id + '0')),
			}, func(ctx context.Context, args map[string]string) (*GetPromptResult, error) {
				return &GetPromptResult{}, nil
			})
		}(i)
	}

	// Run concurrent readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = registry.Prompts()
		}()
	}

	wg.Wait()
}
