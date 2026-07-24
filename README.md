# proto2mcp

> **Turn your gRPC fleet into secure, tenant-aware LLM tools in under two minutes.**

![Status Badge](https://img.shields.io/badge/status-experimental-orange) <!-- placeholder -->

`proto2mcp` is a `protoc`/`buf` plugin that generates Model Context Protocol (MCP) servers directly from your Protobuf service definitions. It bridges the gap between your existing backend microservices (often built with gRPC/Connect) and LLM-powered agents.

## The "Before/After"

**Before `proto2mcp`**: Manual MCP tool registration (verbose, error-prone)
```go
// 50+ lines of manual schema definition, validation, and mapping
mcpServer.RegisterTool("get_user", "Get user by ID", json.RawMessage(`{
  "type": "object",
  "properties": {
    "id": { "type": "string" }
  },
  "required": ["id"]
}`), func(ctx context.Context, args json.RawMessage) (any, error) {
    // manual decoding and validation
    // manual client call
    // manual encoding
})
```

**After `proto2mcp`**: Zero boilerplate
```go
// Just 3 lines after `buf generate`
mcpSrv := mcp.NewServer("my-mcp", "1.0.0")
mcpgen.RegisterUserServiceMCP(mcpSrv, userClient)
mcpSrv.Serve()
```

## Quick Start

1. Add the plugin to your `buf.gen.yaml`:

```yaml
version: v2
plugins:
  - plugin: go
    out: gen/go
    opt: paths=source_relative
  - plugin: buf.build/protocgen/proto2mcp
    out: gen/go
    opt: paths=source_relative
```

2. Run generation:
```bash
buf generate
```

3. Register in your Go code:
```go
package main

import (
	"log"

	"github.com/modelcontextprotocol/go-sdk/server"
	"github.com/protocgen/proto2mcp/pkg/mcpruntime"
	myappv1 "github.com/myorg/myrepo/gen/go/myapp/v1"
)

func main() {
	// 1. Initialize your existing gRPC/Connect client
	client := myappv1.NewUserServiceClient(...)

	// 2. Create an MCP server
	s := server.NewMCPServer("User API", "1.0.0")

	// 3. Register the generated tools (with optional middleware)
	myappv1.ForwardUserServiceToConnect(s, client,
		mcpruntime.WithMiddleware(myAuthMiddleware),
	)

	// 4. Serve
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
```

## Feature Highlights

* **Zero-config**: Automatically infers tool schemas from Protobuf messages.
* **ConnectRPC forwarding**: Generates lightweight proxy code to forward MCP requests to your gRPC/Connect endpoints.
* **Multitenancy**: Transparently injects context/tenant headers (via interceptors).
* **`buf.validate` integration**: Propagates validation constraints (`minLength`, `pattern`, etc.) to LLM tool definitions, giving the model hints to prevent hallucinations.
* **LLM Linter**: Emits warnings at compile time if your API design is "LLM-hostile" (e.g., using `google.protobuf.Any`, streaming, or undocumented fields).

## Architecture

`proto2mcp` uses a two-phase compilation architecture:

1. **Phase 1: Extract** (`pkg/extract`) - Converts Protobuf descriptors into a normalized Intermediate Representation (IR). This phase is isolated and reusable.
2. **Phase 2: Emit** (`pkg/emit`) - Generates the target code (Go) from the IR. 

This separation allows the `extract` package to be used independently by AI API Gateways for dynamic runtime tool extraction (V3).

## Roadmap

| Phase | Features | Status |
| :--- | :--- | :--- |
| **V1** | Go Code Generation, ConnectRPC integration, LLM Linter | 🚧 In Progress |
| **V2** | Typescript Generator, Python Generator, MCP Resources | 📅 Planned |
| **V3** | API Gateway integration, Dynamic runtime extraction | 📅 Planned |

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## License

Apache 2.0. See [LICENSE](LICENSE) for details.

---
Part of the [protocgen](https://github.com/protocgen) organization. Sister project to [proto2type](https://github.com/protocgen/proto2type).
