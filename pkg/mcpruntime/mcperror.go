package mcpruntime

import "fmt"

// MCPError is a structured error type for MCP tool failures.
// Middleware can inspect errors via errors.As instead of JSON parsing.
//
//	var mcpErr *mcpruntime.MCPError
//	if errors.As(err, &mcpErr) {
//	    log.Printf("tool error: code=%s msg=%s", mcpErr.Code, mcpErr.Message)
//	}
type MCPError struct {
	// Code is a machine-readable error code (e.g., "INVALID_ARGUMENT", "NOT_FOUND").
	Code string
	// Message is a human/LLM-readable error description.
	Message string
	// Violations contains field-level validation errors, if any.
	Violations []string
}

// Error implements the error interface.
func (e *MCPError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return e.Message
}

// NewMCPError creates a new MCPError with the given code and message.
func NewMCPError(code, message string) *MCPError {
	return &MCPError{Code: code, Message: message}
}

// NewMCPErrorWithViolations creates a new MCPError with field-level violations.
func NewMCPErrorWithViolations(code, message string, violations []string) *MCPError {
	return &MCPError{Code: code, Message: message, Violations: violations}
}

// ToCallToolResult converts an MCPError to a CallToolResult for MCP transport.
func (e *MCPError) ToCallToolResult() *CallToolResult {
	return NewErrorResultWithDetails(e.Message, e.Code, e.Violations)
}
