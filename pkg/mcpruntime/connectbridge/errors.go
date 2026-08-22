package connectbridge

import (
	"encoding/json"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/protocgen/proto2mcp/pkg/mcpruntime"
	"google.golang.org/protobuf/encoding/protojson"
)

// ErrorMapper maps ConnectRPC errors to MCP CallToolResults.
type ErrorMapper struct {
	// VerboseErrors controls whether field-level validation details
	// are included in error responses. Set to true during development
	// for LLM self-correction, false in production to prevent schema leakage.
	VerboseErrors bool
}

// MapConnectError maps a ConnectRPC error using default settings (verbose errors enabled).
// For production, use ErrorMapper{VerboseErrors: false}.MapConnectError instead.
func MapConnectError(err error) *mcpruntime.CallToolResult {
	return (&ErrorMapper{VerboseErrors: true}).MapConnectError(err)
}

// MapConnectError maps a ConnectRPC error to an MCP-compatible CallToolResult.
func (e *ErrorMapper) MapConnectError(err error) *mcpruntime.CallToolResult {
	if err == nil {
		return nil
	}

	connectErr, ok := asConnectError(err)
	if !ok {
		// Not a connect error — return sanitized generic message.
		return mcpruntime.InternalError("an internal error occurred while processing the request")
	}

	switch connectErr.Code() {
	case connect.CodeInvalidArgument:
		if e.VerboseErrors {
			return mapInvalidArgument(connectErr)
		}
		return mcpruntime.NewErrorResultWithDetails("invalid input", "INVALID_ARGUMENT", nil)
	case connect.CodeNotFound:
		return mcpruntime.NewErrorResultWithDetails("resource not found", "NOT_FOUND", nil)
	case connect.CodePermissionDenied:
		return mcpruntime.NewErrorResultWithDetails("permission denied", "PERMISSION_DENIED", nil)
	case connect.CodeUnauthenticated:
		return mcpruntime.NewErrorResultWithDetails("unauthenticated: valid credentials required", "UNAUTHENTICATED", nil)
	case connect.CodeAlreadyExists:
		return mcpruntime.NewErrorResultWithDetails("resource already exists", "ALREADY_EXISTS", nil)
	case connect.CodeFailedPrecondition:
		return mcpruntime.NewErrorResultWithDetails("operation precondition not met", "FAILED_PRECONDITION", nil)
	case connect.CodeResourceExhausted:
		return mcpruntime.NewErrorResultWithDetails("rate limit exceeded, try again later", "RESOURCE_EXHAUSTED", nil)
	case connect.CodeCanceled:
		return mcpruntime.NewErrorResultWithDetails("Request was canceled", "CANCELED", nil)
	case connect.CodeUnknown:
		return mcpruntime.InternalError("an internal error occurred while processing the request")
	case connect.CodeDeadlineExceeded:
		return mcpruntime.NewErrorResultWithDetails("Request timed out", "DEADLINE_EXCEEDED", nil)
	case connect.CodeAborted:
		return mcpruntime.NewErrorResultWithDetails("Operation was aborted, please retry", "ABORTED", nil)
	case connect.CodeUnimplemented:
		return mcpruntime.NewErrorResultWithDetails("This operation is not supported", "UNIMPLEMENTED", nil)
	case connect.CodeUnavailable:
		return mcpruntime.NewErrorResultWithDetails("Service temporarily unavailable, please retry", "UNAVAILABLE", nil)
	case connect.CodeDataLoss:
		return mcpruntime.NewErrorResultWithDetails("Data loss detected", "DATA_LOSS", nil)
	default:
		return mcpruntime.InternalError("an internal error occurred while processing the request")
	}
}

// mapInvalidArgument extracts field-level validation details from a
// ConnectRPC InvalidArgument error. It iterates over error details to
// find protovalidate Violations messages and formats them as actionable
// feedback for the LLM.
//
// Output format: "field 'user_id': value must be at least 3 characters"
func mapInvalidArgument(connectErr *connect.Error) *mcpruntime.CallToolResult {
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
		return mcpruntime.InvalidParamsMessage(connectErr.Message())
	}

	return mcpruntime.NewErrorResultWithDetails(
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
			FieldPath    string `json:"fieldPath"`
			ConstraintID string `json:"constraintId"`
			Message      string `json:"message"`
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

// asConnectError attempts to extract a *connect.Error from the given error.
// Uses errors.As to correctly unwrap wrapped errors, ensuring that
// middleware-wrapped connect errors retain their structured details
// (violation fields, error codes, etc.).
func asConnectError(err error) (*connect.Error, bool) {
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		return connectErr, true
	}
	return nil, false
}
