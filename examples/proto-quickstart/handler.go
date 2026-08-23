package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"connectrpc.com/connect"
	todov1 "github.com/protocgen/proto2mcp/examples/proto-quickstart/gen/todo/v1"
)

// todoStore is a thread-safe in-memory todo store.
type todoStore struct {
	mu     sync.RWMutex
	todos  map[int64]*todov1.GetTodoResponse
	nextID atomic.Int64
}

var store = &todoStore{todos: make(map[int64]*todov1.GetTodoResponse)}

// TodoHandler implements todov1.TodoServiceMCPHandler
type TodoHandler struct{}

func (h *TodoHandler) CreateTodo(ctx context.Context, req *todov1.CreateTodoRequest) (*todov1.CreateTodoResponse, error) {
	if req.Title == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("title is required"))
	}

	id := store.nextID.Add(1)
	todo := &todov1.GetTodoResponse{Id: id, Title: req.Title, Done: false}

	store.mu.Lock()
	store.todos[id] = todo
	store.mu.Unlock()

	return &todov1.CreateTodoResponse{
		Id:    id,
		Title: req.Title,
		Done:  false,
	}, nil
}

func (h *TodoHandler) GetTodo(ctx context.Context, req *todov1.GetTodoRequest) (*todov1.GetTodoResponse, error) {
	store.mu.RLock()
	todo, ok := store.todos[req.Id]
	store.mu.RUnlock()

	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("todo %d not found", req.Id))
	}

	return todo, nil
}

func (h *TodoHandler) ListTodos(ctx context.Context, req *todov1.ListTodosRequest) (*todov1.ListTodosResponse, error) {
	store.mu.RLock()
	todos := make([]*todov1.GetTodoResponse, 0, len(store.todos))
	for _, t := range store.todos {
		todos = append(todos, t)
	}
	store.mu.RUnlock()

	return &todov1.ListTodosResponse{Todos: todos}, nil
}

func (h *TodoHandler) DeleteTodo(ctx context.Context, req *todov1.DeleteTodoRequest) (*todov1.DeleteTodoResponse, error) {
	store.mu.Lock()
	_, ok := store.todos[req.Id]
	if ok {
		delete(store.todos, req.Id)
	}
	store.mu.Unlock()

	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("todo %d not found", req.Id))
	}

	return &todov1.DeleteTodoResponse{Status: "deleted"}, nil
}
