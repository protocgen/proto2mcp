package main

import (
	"fmt"
	"log"
	"net/http"

	todov1 "github.com/protocgen/proto2mcp/examples/proto-quickstart/gen/todo/v1"
	"github.com/protocgen/proto2mcp/pkg/mcpruntime"
)

func main() {
	registry := mcpruntime.NewToolRegistry()

	// Register the generated MCP handler
	todov1.RegisterTodoServiceMCP(registry, &TodoHandler{})

	fmt.Println("MCP server listening on http://localhost:8080")
	fmt.Println("")
	fmt.Println("Try it:")
	fmt.Println("  curl -s localhost:8080 -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/list\"}' | jq")
	fmt.Println("")
	fmt.Println("Claude Desktop config (~/.config/claude/claude_desktop_config.json):")
	fmt.Println(`  {"mcpServers": {"todo": {"url": "http://localhost:8080"}}}`)

	log.Fatal(http.ListenAndServe(":8080", NewMCPHandler(registry)))
}
