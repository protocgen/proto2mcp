package mcpruntime

import (
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func newErrorResult(msg string) *CallToolResult {
	// A simple text representation of the error as a JSON string,
	// standardizing per MCP typical string returns for errors, though
	// MCP protocol might dictate a specific content array structure.
	b, _ := json.Marshal(map[string]interface{}{
		"error": msg,
	})
	return &CallToolResult{
		Content: b,
		IsError: true,
	}
}

// MapConnectError maps a ConnectRPC error to an MCP CallToolResult.
// Maps codes: InvalidArgument->InvalidParams description, NotFound->clear message,
// PermissionDenied->unauthorized message, etc.
// NEVER leaks internal stack traces or sensitive backend details.
func MapConnectError(err error) *CallToolResult {
	if err == nil {
		return nil
	}

	code := connect.CodeOf(err)
	switch code {
	case connect.CodeInvalidArgument:
		return InvalidParamsError(fmt.Errorf("invalid argument provided"))
	case connect.CodeNotFound:
		return newErrorResult("resource not found")
	case connect.CodePermissionDenied:
		return newErrorResult("permission denied")
	case connect.CodeUnauthenticated:
		return newErrorResult("unauthenticated access")
	default:
		// Default to a sanitized internal error to avoid leaking details.
		return InternalError("an internal error occurred while processing the request")
	}
}

// InvalidParamsError creates a CallToolResult for invalid input parameters.
func InvalidParamsError(err error) *CallToolResult {
	return newErrorResult(fmt.Sprintf("InvalidParams: %v", err))
}

// InternalError creates a CallToolResult for internal errors (sanitized, no stack traces).
func InternalError(msg string) *CallToolResult {
	return newErrorResult(fmt.Sprintf("InternalError: %s", msg))
}

// UnmarshalToolInput unmarshals MCP tool arguments into a proto message.
// Uses protojson with strict mode.
func UnmarshalToolInput(req ToolRequest, msg proto.Message) error {
	opts := protojson.UnmarshalOptions{
		DiscardUnknown: false, // Strict mode
	}
	return opts.Unmarshal(req.Arguments, msg)
}

// MarshalToolResult marshals a proto response message into an MCP CallToolResult.
func MarshalToolResult(msg proto.Message) (*CallToolResult, error) {
	opts := protojson.MarshalOptions{
		EmitUnpopulated: true,
		UseProtoNames:   true,
	}
	b, err := opts.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return &CallToolResult{
		Content: json.RawMessage(b),
		IsError: false,
	}, nil
}
