package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/protocgen/proto2mcp/pkg/mcpruntime"
)

// Todo represents a todo item.
type Todo struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

// todoStore is a thread-safe in-memory todo store.
type todoStore struct {
	mu     sync.RWMutex
	todos  map[int64]*Todo
	nextID atomic.Int64
}

var store = &todoStore{todos: make(map[int64]*Todo)}

func registerTodoTools(registry *mcpruntime.ToolRegistry) {
	registry.Register(mcpruntime.ToolDefinition{
		Name:        "CreateTodo",
		Description: "Create a new todo item",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"title": {"type": "string", "description": "The todo title"}
			},
			"required": ["title"]
		}`),
		Annotations: map[string]any{"destructiveHint": false},
	}, handleCreateTodo)

	registry.Register(mcpruntime.ToolDefinition{
		Name:        "GetTodo",
		Description: "Get a todo by ID",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {"type": "integer", "description": "The todo ID"}
			},
			"required": ["id"]
		}`),
		Annotations: map[string]any{"readOnlyHint": true},
	}, handleGetTodo)

	registry.Register(mcpruntime.ToolDefinition{
		Name:        "ListTodos",
		Description: "List all todo items",
		InputSchema: json.RawMessage(`{"type": "object", "properties": {}}`),
		Annotations: map[string]any{"readOnlyHint": true},
	}, handleListTodos)

	registry.Register(mcpruntime.ToolDefinition{
		Name:        "DeleteTodo",
		Description: "Delete a todo by ID",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {"type": "integer", "description": "The todo ID"}
			},
			"required": ["id"]
		}`),
		Annotations: map[string]any{"destructiveHint": true},
	}, handleDeleteTodo)
}

func handleCreateTodo(_ context.Context, req mcpruntime.ToolRequest) (*mcpruntime.CallToolResult, error) {
	var input struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(req.Arguments, &input); err != nil {
		return mcpruntime.InvalidParamsError(err), nil
	}
	if input.Title == "" {
		return mcpruntime.InvalidParamsMessage("title is required"), nil
	}

	id := store.nextID.Add(1)
	todo := &Todo{ID: id, Title: input.Title}

	store.mu.Lock()
	store.todos[id] = todo
	store.mu.Unlock()

	return jsonResult(todo)
}

func handleGetTodo(_ context.Context, req mcpruntime.ToolRequest) (*mcpruntime.CallToolResult, error) {
	var input struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(req.Arguments, &input); err != nil {
		return mcpruntime.InvalidParamsError(err), nil
	}

	store.mu.RLock()
	todo, ok := store.todos[input.ID]
	store.mu.RUnlock()

	if !ok {
		return mcpruntime.InternalError(fmt.Sprintf("todo %d not found", input.ID)), nil
	}
	return jsonResult(todo)
}

func handleListTodos(_ context.Context, _ mcpruntime.ToolRequest) (*mcpruntime.CallToolResult, error) {
	store.mu.RLock()
	todos := make([]*Todo, 0, len(store.todos))
	for _, t := range store.todos {
		todos = append(todos, t)
	}
	store.mu.RUnlock()

	return jsonResult(todos)
}

func handleDeleteTodo(_ context.Context, req mcpruntime.ToolRequest) (*mcpruntime.CallToolResult, error) {
	var input struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(req.Arguments, &input); err != nil {
		return mcpruntime.InvalidParamsError(err), nil
	}

	store.mu.Lock()
	_, ok := store.todos[input.ID]
	if ok {
		delete(store.todos, input.ID)
	}
	store.mu.Unlock()

	if !ok {
		return mcpruntime.InternalError(fmt.Sprintf("todo %d not found", input.ID)), nil
	}
	return jsonResult(map[string]string{"status": "deleted"})
}

func jsonResult(v any) (*mcpruntime.CallToolResult, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return mcpruntime.InternalError("failed to serialize response"), nil
	}
	return &mcpruntime.CallToolResult{Content: b}, nil
}
