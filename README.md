# proto2mcp

> Generate type-safe MCP (Model Context Protocol) servers from Protobuf service definitions.

## Why?

Large Language Models (LLMs) are incredibly capable, but integrating them safely and reliably into existing infrastructure requires robust tooling. The Model Context Protocol (MCP) provides a standardized way for LLMs to invoke tools, but writing and maintaining the schemas, error handling, and routing for these tools manually is repetitive and error-prone.

Meanwhile, your existing backend systems likely already have well-defined APIs using Protocol Buffers and gRPC/ConnectRPC. These definitions contain exactly the structural information that an LLM needs to use your tools safely, including input schemas, descriptions, and validation rules.

`proto2mcp` bridges this gap. By adding a few lightweight annotations to your `.proto` files, one `protoc` command generates a fully-typed, ready-to-run MCP server. It handles JSON Schema generation, tool registration, middleware, metrics, and error mapping automatically, letting you focus on the business logic instead of the boilerplate.

## Quick Start (2 minutes)

### 1. Annotate your proto
```protobuf
import "protocgen/mcp/v1/options.proto";

service PatientService {
  // Get a patient by their unique identifier.
  rpc GetPatient(GetPatientRequest) returns (GetPatientResponse) {
    option (protocgen.mcp.v1.method) = {
      description: "Look up a patient record by ID"
    };
  }
}
```

### 2. Generate
```bash
buf generate
```

### 3. Use
```go
registry := mcpruntime.NewToolRegistry()
RegisterPatientServiceMCP(registry, myHandler)
```

## Features

- **Type-safe**: Generated handler interfaces match your proto definitions exactly.
- **Zero boilerplate**: Registration, schema generation, error mapping — all generated for you.
- **`buf.validate` → JSON Schema**: Validation constraints are automatically surfaced to LLMs to prevent bad inputs.
- **Well-known types**: Timestamp, Duration, and wrappers are mapped correctly to JSON schema types.
- **LLM Linter**: Warns you at generation time about patterns that confuse LLMs (like streaming, `Any` types, or overly complex schemas).
- **ConnectRPC bridge**: Optional forwarding to directly connect MCP calls to existing ConnectRPC backends without rewriting handlers.
- **Middleware**: Intercept requests with composable middleware for Auth, logging, and tenant isolation.
- **OTel metrics**: Built-in OpenTelemetry metrics (`mcp_tool_calls_total` and `mcp_tool_call_duration_seconds`).

## Architecture

```
.proto files
    │
    ▼
protoc-gen-proto2mcp (extract → emit)
    │
    ▼
service.pb.mcp.go
    ├── ServiceMCPHandler interface
    ├── RegisterServiceMCP()
    └── ServiceForwardToConnect() (opt-in)
```

## Installation

Install the protoc plugin:

```bash
go install github.com/protocgen/proto2mcp/cmd/protoc-gen-proto2mcp@latest
```

And add the runtime dependency to your Go module:

```bash
go get github.com/protocgen/proto2mcp/pkg/mcpruntime
```

## Configuration (proto annotations)

Use the `protocgen.mcp.v1` options to customize how tools are generated.

- **Method-level (`method`)**: Override tool name, description, skip generation, or mark as read-only.
- **Service-level (`service`)**: Add tool name prefixes or service-wide descriptions.
- **File-level (`file`)**: Skip entire files or define MCP Prompt orchestrations.

## Middleware

You can wrap tools with middleware at registration time. Middleware can inspect the request, modify context, or handle errors globally.

```go
func loggingMiddleware(next mcpruntime.HandlerFunc) mcpruntime.HandlerFunc {
    return func(ctx context.Context, req mcpruntime.ToolRequest) (any, error) {
        log.Printf("Calling tool %s", req.Name)
        return next(ctx, req)
    }
}

RegisterPatientServiceMCP(registry, handler, mcpruntime.WithMiddleware(loggingMiddleware))
```

## ConnectRPC Integration

If you already have a running ConnectRPC backend, `proto2mcp` can generate a forwarder that skips the handler implementation entirely, bridging the MCP request directly to a Connect client. This is completely opt-in and lets you expose existing internal APIs to LLMs without rewriting them.

## LLM Linter

The plugin includes an LLM Linter that analyzes your Protobuf definitions. It will emit warnings if you use patterns that LLMs typically struggle with, such as:
- Bi-directional or client streaming
- `google.protobuf.Any` fields (LLMs can't dynamically construct these)
- Extremely large, deeply nested request messages.

## Metrics

The `mcpruntime` package provides built-in support for OpenTelemetry metrics. You can track tool usage, errors, and latencies.

```go
metrics, err := mcpruntime.NewMetrics(otel.Meter("mcp"))
// Then pass it via options during registration
RegisterPatientServiceMCP(registry, handler, mcpruntime.WithMetrics(metrics))
```

## API Reference

See the [godoc](https://pkg.go.dev/github.com/protocgen/proto2mcp) for full package documentation.

## Contributing

We welcome contributions! Please open an issue or submit a pull request.

## License

MIT
