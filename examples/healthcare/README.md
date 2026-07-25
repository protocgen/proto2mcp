# Healthcare Service MCP Example

This example demonstrates how to build a Model Context Protocol (MCP) server from a Protobuf definition using `proto2mcp`.

## Overview

The `proto/healthcare/v1/service.proto` file defines a `PatientService` with three RPCs:
1. `GetPatient`
2. `ListPatients`
3. `CreateAppointment`

It uses the `protocgen.mcp.v1` options to customize tool descriptions, and `buf.validate` to specify validation rules for inputs (e.g., UUID validation for patient IDs).

## Running the Example

In this example, we provide a mock of the generated code inside `main.go` to show how you would wire up the real generated code. 

To run the example, execute the following command:

```bash
go run main.go
```

You should see output similar to this:

```
Registered 3 tools
  - get_patient: Look up a patient record by ID
  - list_patients: List multiple patients, optionally filtering by name or status
  - create_appointment: Schedule a new appointment for an existing patient
MCP server initialized successfully. Ready to receive requests.
```

## How it Works

1. **Protobuf Definition**: You write your gRPC/Protobuf service, adding MCP annotations where desired.
2. **Code Generation**: You run `buf generate` or `protoc` with `protoc-gen-proto2mcp` to generate the Go code.
3. **Implementation**: You implement the generated handler interface in Go.
4. **Registration**: You use the generated `Register<Service>MCP` function to register your implementation with the `mcpruntime.ToolRegistry`.
5. **Transport**: You bind the registry to an MCP transport (like stdio or SSE) to serve it to LLM clients.
