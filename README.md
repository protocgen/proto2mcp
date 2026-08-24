[![CI](https://github.com/protocgen/proto2mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/protocgen/proto2mcp/actions/workflows/ci.yml)

# proto2mcp

> Generate type-safe MCP (Model Context Protocol) servers from Protobuf service definitions.
>
> Targets **MCP 2026-07-28** (stateless protocol).

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

### Runnable Examples

| Example | What it shows | How to run |
|---------|--------------|------------|
| [`examples/quickstart/`](examples/quickstart/) | Manual tool registration, zero deps | `cd examples/quickstart && go run .` |
| [`examples/proto-quickstart/`](examples/proto-quickstart/) | Proto codegen (generated code pre-committed) | `cd examples/proto-quickstart && go run .` |

Both implement the same TodoService — compare them to see what codegen gives you.

See also [`examples/connectrpc-bridge/`](examples/connectrpc-bridge/) for forwarding MCP calls to existing ConnectRPC backends.

## Features

- **Type-safe**: Generated handler interfaces match your proto definitions exactly.
- **Zero boilerplate**: Registration, schema generation, error mapping — all generated for you.
- **`buf.validate` → JSON Schema**: Validation constraints are automatically surfaced to LLMs to prevent bad inputs.
- **Well-known types**: Timestamp, Duration, and wrappers are mapped correctly to JSON schema types.
- **LLM Linter**: Warns you at generation time about patterns that confuse LLMs (like streaming, `Any` types, or overly complex schemas).
- **ConnectRPC bridge**: Optional forwarding to directly connect MCP calls to existing ConnectRPC backends without rewriting handlers.
- **Middleware**: Intercept requests with composable middleware for Auth, logging, and tenant isolation.
- **OTel metrics**: Built-in OpenTelemetry metrics (`mcp_tool_calls_total` and `mcp_tool_call_duration_seconds`).
- **Prompt templates**: Define LLM prompt templates in proto with explicit arguments, generate handler interfaces.
- **Resource URIs**: Annotate methods with URI templates for MCP Resource exposure.
- **Tool filtering**: `FilteredTools(ctx)` ensures agents only see tools they're authorized to use.
- **Rate limiting**: Built-in per-tenant token bucket rate limiter middleware.
- **Security hardening**: Header allowlisting, resource key validation, bounded metrics cardinality.
- **Macro tools (experimental)**: Compose tools into sequential macro workflows via proto annotations.

## Architecture

```text
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
go install github.com/protocgen/proto2mcp/codegen/cmd/protoc-gen-proto2mcp@latest
```

Or use the pre-built Docker image (no Go required):

```bash
docker pull ghcr.io/protocgen/proto2mcp:latest

# Use as a protoc plugin via docker
docker run --rm -v $(pwd):/workspace ghcr.io/protocgen/proto2mcp:latest
```

Then add the runtime dependency to your Go module:

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
// loggingInterceptor implements mcpruntime.ToolInterceptor.
type loggingInterceptor struct{}

func (l *loggingInterceptor) HandleToolCall(ctx context.Context, req mcpruntime.ToolRequest, next mcpruntime.HandlerFunc) (*mcpruntime.CallToolResult, error) {
    log.Printf("Calling tool %s", req.ToolName)
    return next(ctx, req)
}

RegisterPatientServiceMCP(registry, handler, mcpruntime.WithMiddleware(&loggingInterceptor{}))
```

### Middleware Metadata

- `ToolRequest.Definition` — access tool metadata (annotations, schema) in middleware
- `ToolRequest.ResourceKeys` — auto-extracted resource identifiers for ABAC

The `resource_key` proto annotation lets you mark fields that contain resource IDs. The generated code automatically extracts these into `req.ResourceKeys`.

Example authorization middleware using ResourceKeys:

```go
// authzInterceptor checks resource access before tool execution.
type authzInterceptor struct {
    policy PolicyEngine
}

func (a *authzInterceptor) HandleToolCall(ctx context.Context, req mcpruntime.ToolRequest, next mcpruntime.HandlerFunc) (*mcpruntime.CallToolResult, error) {
    // Resource keys are automatically extracted from arguments
    // for fields annotated with resource_key in your proto.
    if patientID, ok := req.ResourceKeys["patient_id"]; ok {
        if !a.policy.CanAccess(ctx, "patient", patientID) {
            return mcpruntime.InternalError("access denied"), nil
        }
    }
    return next(ctx, req)
}
```

```protobuf
import "protocgen/mcp/v1/options.proto";

message GetPatientRequest {
  string patient_id = 1 [(protocgen.mcp.v1.field) = { resource_key: true }];
}
```

## ConnectRPC Integration

If you already have a running ConnectRPC backend, `proto2mcp` can generate a forwarder that skips the handler implementation entirely, bridging the MCP request directly to a Connect client. This is completely opt-in and lets you expose existing internal APIs to LLMs without rewriting them.

## Prompt Templates

Define LLM prompt templates at the file level in your proto files:

```protobuf
import "protocgen/mcp/v1/options.proto";

option (protocgen.mcp.v1.file) = {
  prompts: [{
    name: "OnboardPatient"
    description: "Guide through patient onboarding"
    arguments: [
      { name: "patient_name", description: "Patient full name", required: true },
      { name: "insurance_id", description: "Insurance ID" }
    ]
    tools: ["PatientService_CreatePatient", "BillingService_CreateAccount"]
  }]
};
```

Generates a handler interface and registration function:

```go
// Implement the generated interface
type myPromptHandler struct{}

func (h *myPromptHandler) HandleOnboardPatient(ctx context.Context, args map[string]string) (*mcpruntime.GetPromptResult, error) {
    return &mcpruntime.GetPromptResult{
        Messages: []mcpruntime.PromptMessage{{
            Role: "user",
            Content: mcpruntime.TextContent(fmt.Sprintf("Onboard patient %s", args["patient_name"])),
        }},
    }, nil
}

// Register
promptRegistry := mcpruntime.NewPromptRegistry()
RegisterPatientPrompts(promptRegistry, &myPromptHandler{})
```

## Macro Tools (Experimental)

Compose tools into sequential workflows using proto annotations:

```protobuf
rpc OnboardPatient(OnboardReq) returns (OnboardResp) {
  option (protocgen.mcp.v1.method) = {
    macro: {
      steps: [
        { tool: "CreatePatient", output_key: "patient" },
        { tool: "CreateBilling", output_key: "billing" }
      ]
    }
  };
}
```

The generated code calls `RegisterMacro` with step definitions. Use the internal `SequentialExecutor` to run them:

```go
import "github.com/protocgen/proto2mcp/pkg/mcpruntime/internal/macro"

executor := &macro.SequentialExecutor{
    Lookup: func(name string) (macro.HandlerFunc, bool) {
        h, ok := registry.Lookup(name)
        // adapt HandlerFunc types...
        return adapted, ok
    },
}
```

> **Note:** Macro APIs are experimental and may change.

## LLM Linter

The plugin includes an LLM Linter that analyzes your Protobuf definitions. It will emit warnings if you use patterns that LLMs typically struggle with, such as:
- Bi-directional or client streaming
- `google.protobuf.Any` fields (LLMs can't dynamically construct these)
- Extremely large, deeply nested request messages.

## Metrics

The `mcpruntime` package provides built-in support for OpenTelemetry metrics. You can track tool usage, errors, and latencies.

```go
// Bounded cardinality (recommended for production)
tools := registry.Tools()
toolNames := make([]string, len(tools))
for i, t := range tools {
    toolNames[i] = t.Name
}
metrics, err := mcpruntime.NewBoundedMetrics(otel.Meter("mcp"), toolNames)
if err != nil {
    log.Fatal(err)
}

// Record in your middleware or handler
metrics.RecordToolCall(ctx, toolName, tenantID, "success", duration)
```

## Security

proto2mcp operates at the **tool authorization layer** — controlling which tools an authenticated agent can call and with what arguments. It does not handle transport-level authentication (OAuth 2.1, HTTP sessions); use your MCP server framework for that.

For a comprehensive guide covering tool allowlists, JWT/RBAC, macaroon/biscuit capability tokens, and a production security checklist, see **[docs/authorization.md](docs/authorization.md)**.

### Tool Filtering

Use `FilteredTools` to ensure agents only see tools they're authorized for:

```go
// Agents only see permitted tools in tools/list
tools := registry.FilteredTools(ctx, authzMiddleware)
```

### Header Allowlist

Generated ConnectRPC forwarders filter headers through `DefaultHeaderAllowlist` (Authorization, X-Request-ID, traceparent, tracestate). Customize with `WithHeaderAllowlist`.

### Rate Limiting

```go
rl := mcpruntime.NewRateLimiter(10.0, 20) // 10 calls/sec, burst 20
RegisterServiceMCP(registry, handler, mcpruntime.WithMiddleware(rl))
```

### Resource Key Validation

```go
mcpruntime.WithResourceKeyValidator(func(key, value string) error {
    if strings.Contains(value, "..") {
        return fmt.Errorf("invalid resource key")
    }
    return nil
})
```

### Error Verbosity

Control validation error detail level for production vs development:

```go
// Production: generic errors, no schema leakage
mapper := &connectbridge.ErrorMapper{VerboseErrors: false}

// Development: field-level details for LLM self-correction
mapper := &connectbridge.ErrorMapper{VerboseErrors: true}
```

## API Reference

See the [godoc](https://pkg.go.dev/github.com/protocgen/proto2mcp) for full package documentation.

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for setup instructions and guidelines. See [CHANGELOG.md](CHANGELOG.md) for release history.

Quick start:
```bash
nix develop    # Enter dev shell with all tools
make test      # Run full test suite
make hygiene   # Run formatting checks
```

## License

Apache 2.0
