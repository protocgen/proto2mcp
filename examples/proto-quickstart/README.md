# Proto-driven MCP Quickstart

This example shows the **REAL** proto2mcp workflow. While the `quickstart` example shows how to use the runtime by manually constructing `ToolDefinitions`, this example demonstrates the actual code generation value proposition.

## Workflow

1. Write your `.proto` file with MCP annotations:
   See `proto/todo/v1/todo.proto` where we annotate the service and methods.
2. Run `buf generate` to generate the Go code and MCP handler interface.
3. Implement the generated `TodoServiceMCPHandler` interface (see `handler.go`).
4. Wire it up with the registry and transport (see `main.go`).

## Running the Server

```bash
go run .
```

## Testing

```bash
# List tools
curl -s localhost:8080 -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | jq

# Call CreateTodo
curl -s localhost:8080 -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"TodoService_CreateTodo","arguments":{"title":"Buy milk"}}}' | jq

# Call ListTodos
curl -s localhost:8080 -d '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"TodoService_ListTodos","arguments":{}}}' | jq
```
