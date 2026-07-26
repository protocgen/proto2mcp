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

// SanitizeErrorMessage strips internal details from error messages.
// Takes only the first line, caps at 200 chars, and removes anything
// that looks like a stack trace, internal file path, or host:port.
func SanitizeErrorMessage(msg string) string {
	// Take only the first line.
	if idx := strings.IndexByte(msg, '\n'); idx != -1 {
		msg = msg[:idx]
	}

	// Cap length to prevent excessively long messages.
	const maxLen = 200
	if len(msg) > maxLen {
		msg = msg[:maxLen] + "..."
	}

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

	return msg
}



// InvalidParamsError creates a CallToolResult for invalid input parameters.
func InvalidParamsError(err error) *CallToolResult {
	return NewErrorResultWithDetails(
		fmt.Sprintf("invalid input: %s", SanitizeErrorMessage(err.Error())),
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
