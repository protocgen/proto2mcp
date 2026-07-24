package mcpruntime

import (
	"encoding/json"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// errorContent is the structured error payload returned in CallToolResult.
type errorContent struct {
	Error      string   `json:"error"`
	Code       string   `json:"code,omitempty"`
	Violations []string `json:"violations,omitempty"`
}

func newErrorResult(msg string) *CallToolResult {
	return newErrorResultWithDetails(msg, "", nil)
}

func newErrorResultWithDetails(msg, code string, violations []string) *CallToolResult {
	ec := errorContent{
		Error:      msg,
		Code:       code,
		Violations: violations,
	}
	b, err := json.Marshal(ec)
	if err != nil {
		// Fallback: if marshal fails (should never happen for this struct),
		// return a minimal valid JSON error.
		b = []byte(fmt.Sprintf(`{"error":%q}`, msg))
	}
	return &CallToolResult{
		Content: b,
		IsError: true,
	}
}

// MapConnectError maps a ConnectRPC error to an MCP CallToolResult.
//
// For InvalidArgument errors, it extracts field-level validation details
// from the error's detail messages (e.g., protovalidate violations) so
// that LLMs can self-correct. This is critical for agent reliability —
// a generic "invalid argument" message gives the LLM no signal about
// what to fix.
//
// Error messages are sanitized to never leak internal stack traces,
// backend addresses, or sensitive implementation details.
func MapConnectError(err error) *CallToolResult {
	if err == nil {
		return nil
	}

	connectErr, ok := asConnectError(err)
	if !ok {
		// Not a connect error — return sanitized generic message.
		return InternalError("an internal error occurred while processing the request")
	}

	switch connectErr.Code() {
	case connect.CodeInvalidArgument:
		return mapInvalidArgument(connectErr)
	case connect.CodeNotFound:
		return newErrorResultWithDetails("resource not found", "NOT_FOUND", nil)
	case connect.CodePermissionDenied:
		return newErrorResultWithDetails("permission denied", "PERMISSION_DENIED", nil)
	case connect.CodeUnauthenticated:
		return newErrorResultWithDetails("unauthenticated: valid credentials required", "UNAUTHENTICATED", nil)
	case connect.CodeAlreadyExists:
		return newErrorResultWithDetails("resource already exists", "ALREADY_EXISTS", nil)
	case connect.CodeFailedPrecondition:
		return newErrorResultWithDetails("operation precondition not met", "FAILED_PRECONDITION", nil)
	case connect.CodeResourceExhausted:
		return newErrorResultWithDetails("rate limit exceeded, try again later", "RESOURCE_EXHAUSTED", nil)
	default:
		return InternalError("an internal error occurred while processing the request")
	}
}

// mapInvalidArgument extracts field-level validation details from a
// ConnectRPC InvalidArgument error. It iterates over error details to
// find protovalidate Violations messages and formats them as actionable
// feedback for the LLM.
//
// Output format: "field 'user_id': value must be at least 3 characters"
func mapInvalidArgument(connectErr *connect.Error) *CallToolResult {
	var violations []string

	// Extract violation details from the connect error.
	// protovalidate attaches these as error details using the
	// buf.validate.Violations message type.
	for _, detail := range connectErr.Details() {
		// Try to extract the detail as a proto message.
		msg, err := detail.Value()
		if err != nil {
			continue
		}

		// Use protojson to get a structured representation we can parse
		// without importing buf.validate directly (avoiding the dependency).
		jsonBytes, err := protojson.Marshal(msg)
		if err != nil {
			continue
		}

		// Parse the violations from the JSON representation.
		parsed := parseViolationsFromJSON(jsonBytes)
		violations = append(violations, parsed...)
	}

	if len(violations) == 0 {
		// No structured details found — use the connect error message
		// but sanitize it (take only the first line, cap length).
		sanitized := sanitizeErrorMessage(connectErr.Message())
		return newErrorResultWithDetails(
			fmt.Sprintf("invalid input: %s", sanitized),
			"INVALID_ARGUMENT",
			nil,
		)
	}

	return newErrorResultWithDetails(
		fmt.Sprintf("invalid input: %d validation error(s)", len(violations)),
		"INVALID_ARGUMENT",
		violations,
	)
}

// parseViolationsFromJSON extracts human-readable violation strings from
// a protovalidate Violations JSON payload. Handles both the standard
// buf.validate.Violations format and similar structures.
func parseViolationsFromJSON(data []byte) []string {
	// Structure matches buf.validate.Violations
	var payload struct {
		Violations []struct {
			FieldPath   string `json:"fieldPath"`
			ConstraintID string `json:"constraintId"`
			Message     string `json:"message"`
		} `json:"violations"`
	}

	if err := json.Unmarshal(data, &payload); err != nil {
		return nil
	}

	var results []string
	for _, v := range payload.Violations {
		if v.FieldPath != "" && v.Message != "" {
			results = append(results, fmt.Sprintf("field '%s': %s", v.FieldPath, v.Message))
		} else if v.Message != "" {
			results = append(results, v.Message)
		}
	}
	return results
}

// sanitizeErrorMessage strips internal details from error messages.
// Takes only the first line, caps at 200 chars, and removes anything
// that looks like a stack trace or internal path.
func sanitizeErrorMessage(msg string) string {
	// Take only the first line.
	if idx := strings.IndexByte(msg, '\n'); idx != -1 {
		msg = msg[:idx]
	}

	// Cap length to prevent excessively long messages.
	const maxLen = 200
	if len(msg) > maxLen {
		msg = msg[:maxLen] + "..."
	}

	// Remove anything that looks like internal details.
	// Strip file paths (e.g., /home/user/service/...)
	// Strip port numbers (e.g., :50051)
	for _, pattern := range []string{"goroutine", "panic:", ".go:", "runtime."} {
		if strings.Contains(msg, pattern) {
			return "invalid input parameters"
		}
	}

	return msg
}

// asConnectError attempts to extract a *connect.Error from the given error.
func asConnectError(err error) (*connect.Error, bool) {
	var connectErr *connect.Error
	if ok := connect.IsNotModifiedError(err); ok {
		// Not a connect error in the usual sense.
		return nil, false
	}
	// connect.CodeOf returns CodeUnknown for non-connect errors,
	// but we need the full *connect.Error to access Details().
	connectErr = new(connect.Error)
	if errors, ok := err.(*connect.Error); ok {
		connectErr = errors
		return connectErr, true
	}
	return nil, false
}

// InvalidParamsError creates a CallToolResult for invalid input parameters.
func InvalidParamsError(err error) *CallToolResult {
	return newErrorResultWithDetails(
		fmt.Sprintf("invalid input: %s", sanitizeErrorMessage(err.Error())),
		"INVALID_ARGUMENT",
		nil,
	)
}

// InternalError creates a CallToolResult for internal errors (sanitized, no stack traces).
func InternalError(msg string) *CallToolResult {
	return newErrorResultWithDetails(msg, "INTERNAL", nil)
}

// UnmarshalToolInput unmarshals MCP tool arguments into a proto message.
// Uses protojson with strict mode — unknown fields cause an error rather
// than being silently dropped. This is critical for LLM interactions:
// if the LLM sends a hallucinated field name, we want to catch it.
func UnmarshalToolInput(req ToolRequest, msg proto.Message) error {
	opts := protojson.UnmarshalOptions{
		DiscardUnknown: false, // Strict mode: reject hallucinated fields
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
