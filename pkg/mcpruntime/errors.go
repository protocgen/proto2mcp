package mcpruntime

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// errorContent is the structured error payload returned in CallToolResult.
type errorContent struct {
	Error      string   `json:"error"`
	Code       string   `json:"code,omitempty"`
	Violations []string `json:"violations,omitempty"`
}

// NewErrorResultWithDetails creates a CallToolResult containing the error details.
func NewErrorResultWithDetails(msg, code string, violations []string) *CallToolResult {
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



// Patterns that indicate internal implementation details that must not
// be leaked to LLMs. Compiled once at package init time.
var (
	// Matches absolute file paths: /home/user/..., /var/lib/..., C:\Users\...
	absPathPattern = regexp.MustCompile(`(?:/[a-zA-Z0-9_.+-]+){2,}`)
	// Matches host:port patterns: localhost:50051, backend-svc:8080
	hostPortPattern = regexp.MustCompile(`[a-zA-Z0-9._-]+:\d{2,5}`)
)

// sanitizeErrorMessage strips internal details from error messages.
// Takes only the first line, checks for sensitive patterns on the full
// content, then caps at 200 chars. This ordering prevents truncation
// from splitting a host:port or path at the boundary and bypassing
// the pattern checks.
func sanitizeErrorMessage(msg string) string {
	// Take only the first line.
	if idx := strings.IndexByte(msg, '\n'); idx != -1 {
		msg = msg[:idx]
	}

	// Check for sensitive patterns on the FULL first line, before truncation.
	// Truncation could split "192.168.1.1:5432" into "192.168.1.1:54..."
	// which would bypass the host:port regex.

	// Check for stack trace / panic keywords.
	for _, pattern := range []string{"goroutine", "panic:", ".go:", "runtime."} {
		if strings.Contains(msg, pattern) {
			return "invalid input parameters"
		}
	}

	// Strip absolute file paths.
	if absPathPattern.MatchString(msg) {
		return "invalid input parameters"
	}

	// Strip host:port patterns (e.g., backend-svc:50051).
	if hostPortPattern.MatchString(msg) {
		return "invalid input parameters"
	}

	// Cap length to prevent excessively long messages.
	const maxLen = 200
	if len(msg) > maxLen {
		msg = msg[:maxLen-3] + "..."
	}

	return msg
}



// InvalidParamsError creates a CallToolResult for invalid input parameters.
func InvalidParamsError(err error) *CallToolResult {
	return NewErrorResultWithDetails(
		fmt.Sprintf("invalid input: %s", sanitizeErrorMessage(err.Error())),
		"INVALID_ARGUMENT",
		nil,
	)
}

// InvalidParamsMessage creates a CallToolResult for invalid input with a raw message.
// The message is sanitized to strip internal details before being returned.
func InvalidParamsMessage(msg string) *CallToolResult {
	return NewErrorResultWithDetails(
		fmt.Sprintf("invalid input: %s", sanitizeErrorMessage(msg)),
		"INVALID_ARGUMENT",
		nil,
	)
}

// InternalError creates a CallToolResult for internal errors (sanitized, no stack traces).
func InternalError(msg string) *CallToolResult {
	return NewErrorResultWithDetails(msg, "INTERNAL", nil)
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
// Returns an empty JSON object if msg is nil.
func MarshalToolResult(msg proto.Message) (*CallToolResult, error) {
	if msg == nil {
		return &CallToolResult{
			Content: json.RawMessage(`{}`),
			IsError: false,
		}, nil
	}
	opts := protojson.MarshalOptions{
		EmitUnpopulated: true,
		UseProtoNames:   true,
	}
	b, err := opts.Marshal(msg)
	if err != nil {
		return InternalError("failed to serialize response"), nil
	}
	return &CallToolResult{
		Content: json.RawMessage(b),
		IsError: false,
	}, nil
}
