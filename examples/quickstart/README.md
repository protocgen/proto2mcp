# proto2mcp Quickstart

A running MCP server in 2 minutes. Zero external dependencies beyond proto2mcp.

## Run

```bash
go run .
```

## Try it

```bash
# List available tools
curl -s localhost:8080 -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | jq '.result.tools[].name'

# Create a todo
curl -s localhost:8080 -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"CreateTodo","arguments":{"title":"Buy milk"}}}' | jq

# List all todos
curl -s localhost:8080 -d '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"ListTodos","arguments":{}}}' | jq

# Get a specific todo
curl -s localhost:8080 -d '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"GetTodo","arguments":{"id":1}}}' | jq

# Delete a todo
curl -s localhost:8080 -d '{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"DeleteTodo","arguments":{"id":1}}}' | jq
```

## Use with Claude Desktop

Add to your Claude Desktop config:
- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Linux**: `~/.config/claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "todo": {
      "url": "http://localhost:8080"
    }
  }
}
```

Restart Claude Desktop. Ask Claude: "Create a todo to buy groceries, then list all my todos."

## What this demonstrates

- **Tool registration** with `mcpruntime.ToolRegistry`
- **JSON Schema** for tool input validation
- **Annotations** (`readOnlyHint`, `destructiveHint`) for LLM behavior hints
- **Error handling** with `InvalidParamsError`, `InternalError`
- **MCP transport** — minimal JSON-RPC 2.0 adapter in ~80 lines of `net/http`

## Architecture

```
main.go          — 15 lines: wire registry + start server
handler.go       — CRUD handlers (your business logic)
transport.go     — JSON-RPC 2.0 ↔ mcpruntime adapter (reference impl)
```

The transport adapter (`transport.go`) is a **reference implementation**. For production, consider:
- [mcp-go](https://github.com/mark3labs/mcp-go) for full MCP features (SSE, sessions)
- Your own framework with OTel instrumentation

## Next steps

Once this works, explore:
- **Proto-driven codegen**: Define tools in `.proto` files → `buf generate` → type-safe handlers
- **Middleware**: Add auth, logging, rate limiting via `mcpruntime.WithMiddleware`
- **Security**: See [docs/authorization.md](../../docs/authorization.md)
