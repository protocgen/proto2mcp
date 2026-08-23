# ConnectRPC Bridge Example

Forward MCP tool calls to existing ConnectRPC backends — zero business logic rewrite.

## Architecture

```
MCP Client (Claude) → MCP Server → ForwardToConnect → ConnectRPC Backend
```

## Step 1: Define your proto with MCP annotations

```protobuf
syntax = "proto3";

package example.patient.v1;

import "protocgen/mcp/v1/options.proto";

service PatientService {
  option (protocgen.mcp.v1.service_mcp) = {};

  rpc GetPatient(GetPatientRequest) returns (GetPatientResponse) {
    option (protocgen.mcp.v1.method_mcp) = {
      description: "Get a patient by ID"
    };
  }
}
```

## Step 2: Generate

Running `buf generate` produces:
- Standard Connect client interface: `PatientServiceServiceClient`
- MCP handler interface: `PatientServiceMCPHandler`
- Bridge function: `PatientServiceForwardToConnect(client) -> handler`

## Step 3: Wire it up

```go
// Create a ConnectRPC client to your existing backend
client := patientv1connect.NewPatientServiceClient(
    http.DefaultClient,
    "http://your-connect-backend:8080",
)

// Bridge: reuse your backend as MCP tools
handler := PatientServiceForwardToConnect(client)

// Register with the MCP runtime
registry := mcpruntime.NewToolRegistry()
RegisterPatientServiceMCP(registry, handler)
```

## Step 4: Add middleware

You can easily add middleware on either the MCP side or the Connect client side. For example, using ConnectRPC interceptors to add authentication or logging when forwarding requests:

```go
interceptor := connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
    return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
        // Add authentication, logging, or rate limiting
        return next(ctx, req)
    }
})

client := patientv1connect.NewPatientServiceClient(
    http.DefaultClient,
    "http://your-connect-backend:8080",
    connect.WithInterceptors(interceptor),
)
```

## Error Mapping

The bridge uses `connectbridge.ErrorMapper` to automatically translate ConnectRPC error codes into MCP-friendly responses:

- `connect.CodeInvalidArgument` -> `INVALID_ARGUMENT`
- `connect.CodeNotFound` -> `NOT_FOUND`
- `connect.CodePermissionDenied` -> `PERMISSION_DENIED`
- `connect.CodeResourceExhausted` -> `RESOURCE_EXHAUSTED`

**Verbose Errors & Validation**
By default, the mapper extracts field-level validation details from `buf.validate.Violations` attached to invalid argument errors. This gives the LLM actionable feedback to self-correct (e.g., `"field 'user_id': value must be at least 3 characters"`).

For production use, you can configure the mapper with `VerboseErrors: false` to sanitize responses and prevent schema leakage.

## Header Forwarding

The generated forwarder propagates context headers securely from the MCP side to the ConnectRPC backend. It uses `mcpruntime.HeadersFromContext(ctx)` and applies an allowlist via `mcpruntime.FilterHeaders(headers, mcpruntime.DefaultHeaderAllowlist)`.

This ensures that only safe, permitted headers (like auth tokens or trace IDs) are injected into `connectReq.Header()` and forwarded to your backend.
