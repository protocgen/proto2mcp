package connectbridge

import (
	"encoding/json"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/protocgen/proto2mcp/pkg/mcpruntime"
)

func TestMapConnectError_Nil(t *testing.T) {
	result := MapConnectError(nil)
	if result != nil {
		t.Fatalf("expected nil, got %+v", result)
	}
}

func TestMapConnectError_NotFound(t *testing.T) {
	err := connect.NewError(connect.CodeNotFound, fmt.Errorf("patient 123 not found"))
	result := MapConnectError(err)
	assertErrorResult(t, result, "NOT_FOUND", "resource not found")
}

func TestMapConnectError_PermissionDenied(t *testing.T) {
	err := connect.NewError(connect.CodePermissionDenied, fmt.Errorf("forbidden"))
	result := MapConnectError(err)
	assertErrorResult(t, result, "PERMISSION_DENIED", "permission denied")
}

func TestMapConnectError_Unauthenticated(t *testing.T) {
	err := connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("no token"))
	result := MapConnectError(err)
	assertErrorResult(t, result, "UNAUTHENTICATED", "unauthenticated: valid credentials required")
}

func TestMapConnectError_InvalidArgument_NoDetails(t *testing.T) {
	err := connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("bad input"))
	result := MapConnectError(err)
	assertErrorResult(t, result, "INVALID_ARGUMENT", "invalid input: bad input")
}

func TestMapConnectError_UnknownCode_ReturnsInternal(t *testing.T) {
	err := connect.NewError(connect.CodeInternal, fmt.Errorf("segfault at 0xDEADBEEF"))
	result := MapConnectError(err)
	assertErrorResult(t, result, "INTERNAL", "an internal error occurred while processing the request")
}

func TestMapConnectError_NonConnectError_ReturnsInternal(t *testing.T) {
	err := fmt.Errorf("random non-connect error")
	result := MapConnectError(err)
	assertErrorResult(t, result, "INTERNAL", "an internal error occurred while processing the request")
}

func TestMapConnectError_AlreadyExists(t *testing.T) {
	err := connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("duplicate"))
	result := MapConnectError(err)
	assertErrorResult(t, result, "ALREADY_EXISTS", "resource already exists")
}

func TestMapConnectError_FailedPrecondition(t *testing.T) {
	err := connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("not ready"))
	result := MapConnectError(err)
	assertErrorResult(t, result, "FAILED_PRECONDITION", "operation precondition not met")
}

func TestMapConnectError_ResourceExhausted(t *testing.T) {
	err := connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("rate limited"))
	result := MapConnectError(err)
	assertErrorResult(t, result, "RESOURCE_EXHAUSTED", "rate limit exceeded, try again later")
}

func TestMapConnectError_Canceled(t *testing.T) {
	err := connect.NewError(connect.CodeCanceled, fmt.Errorf("canceled"))
	result := MapConnectError(err)
	assertErrorResult(t, result, "CANCELED", "Request was canceled")
}

func TestMapConnectError_DeadlineExceeded(t *testing.T) {
	err := connect.NewError(connect.CodeDeadlineExceeded, fmt.Errorf("timeout"))
	result := MapConnectError(err)
	assertErrorResult(t, result, "DEADLINE_EXCEEDED", "Request timed out")
}

func TestMapConnectError_Aborted(t *testing.T) {
	err := connect.NewError(connect.CodeAborted, fmt.Errorf("aborted"))
	result := MapConnectError(err)
	assertErrorResult(t, result, "ABORTED", "Operation was aborted, please retry")
}

func TestMapConnectError_Unimplemented(t *testing.T) {
	err := connect.NewError(connect.CodeUnimplemented, fmt.Errorf("unimplemented"))
	result := MapConnectError(err)
	assertErrorResult(t, result, "UNIMPLEMENTED", "This operation is not supported")
}

func TestMapConnectError_Unavailable(t *testing.T) {
	err := connect.NewError(connect.CodeUnavailable, fmt.Errorf("unavailable"))
	result := MapConnectError(err)
	assertErrorResult(t, result, "UNAVAILABLE", "Service temporarily unavailable, please retry")
}

func TestMapConnectError_DataLoss(t *testing.T) {
	err := connect.NewError(connect.CodeDataLoss, fmt.Errorf("data loss"))
	result := MapConnectError(err)
	assertErrorResult(t, result, "DATA_LOSS", "Data loss detected")
}

func TestMapConnectError_Unknown(t *testing.T) {
	err := connect.NewError(connect.CodeUnknown, fmt.Errorf("unknown"))
	result := MapConnectError(err)
	assertErrorResult(t, result, "INTERNAL", "an internal error occurred while processing the request")
}

func TestAsConnectError_DirectError(t *testing.T) {
	original := connect.NewError(connect.CodeNotFound, fmt.Errorf("not found"))
	got, ok := asConnectError(original)
	if !ok {
		t.Fatal("expected ok=true for direct connect.Error")
	}
	if got.Code() != connect.CodeNotFound {
		t.Fatalf("expected CodeNotFound, got %v", got.Code())
	}
}

func TestAsConnectError_WrappedError(t *testing.T) {
	original := connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("bad field"))
	wrapped := fmt.Errorf("middleware: %w", original)
	got, ok := asConnectError(wrapped)
	if !ok {
		t.Fatal("expected ok=true for wrapped connect.Error — errors.As should unwrap")
	}
	if got.Code() != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", got.Code())
	}
}

func TestAsConnectError_NonConnectError(t *testing.T) {
	err := fmt.Errorf("not a connect error")
	_, ok := asConnectError(err)
	if ok {
		t.Fatal("expected ok=false for non-connect error")
	}
}

func TestParseViolationsFromJSON_Valid(t *testing.T) {
	data := `{
		"violations": [
			{"fieldPath": "user_id", "constraintId": "string.min_len", "message": "value must be at least 3 characters"},
			{"fieldPath": "email", "constraintId": "string.email", "message": "value must be a valid email"}
		]
	}`
	results := parseViolationsFromJSON([]byte(data))
	if len(results) != 2 {
		t.Fatalf("expected 2 violations, got %d", len(results))
	}
	if results[0] != "field 'user_id': value must be at least 3 characters" {
		t.Fatalf("unexpected violation[0]: %q", results[0])
	}
	if results[1] != "field 'email': value must be a valid email" {
		t.Fatalf("unexpected violation[1]: %q", results[1])
	}
}

func TestParseViolationsFromJSON_NoFieldPath(t *testing.T) {
	data := `{"violations": [{"message": "something is wrong"}]}`
	results := parseViolationsFromJSON([]byte(data))
	if len(results) != 1 || results[0] != "something is wrong" {
		t.Fatalf("expected message-only violation, got %v", results)
	}
}

func TestParseViolationsFromJSON_Invalid(t *testing.T) {
	results := parseViolationsFromJSON([]byte("not json"))
	if results != nil {
		t.Fatalf("expected nil for invalid JSON, got %v", results)
	}
}

func TestParseViolationsFromJSON_Empty(t *testing.T) {
	results := parseViolationsFromJSON([]byte(`{"violations": []}`))
	if len(results) != 0 {
		t.Fatalf("expected empty, got %v", results)
	}
}

func assertErrorResult(t *testing.T, result *mcpruntime.CallToolResult, expectedCode, expectedError string) {
	t.Helper()
	if result == nil {
		t.Fatal("result is nil")
	}
	if !result.IsError {
		t.Fatal("expected IsError=true")
	}
	var ec struct {
		Error string `json:"error"`
		Code  string `json:"code,omitempty"`
	}
	if err := json.Unmarshal(result.Content, &ec); err != nil {
		t.Fatalf("failed to parse error content: %v\nraw: %s", err, string(result.Content))
	}
	if ec.Code != expectedCode {
		t.Fatalf("expected code %q, got %q", expectedCode, ec.Code)
	}
	if ec.Error != expectedError {
		t.Fatalf("expected error %q, got %q", expectedError, ec.Error)
	}
}

func TestMapConnectError_OutOfRange(t *testing.T) {
	err := connect.NewError(connect.CodeOutOfRange, fmt.Errorf("out of range"))
	result := MapConnectError(err)
	assertErrorResult(t, result, "INTERNAL", "an internal error occurred while processing the request")
}

func TestAsConnectError_DeeplyWrapped(t *testing.T) {
	original := connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("bad field"))
	wrapped1 := fmt.Errorf("layer 1: %w", original)
	wrapped2 := fmt.Errorf("layer 2: %w", wrapped1)
	got, ok := asConnectError(wrapped2)
	if !ok {
		t.Fatal("expected ok=true for deeply wrapped connect.Error")
	}
	if got.Code() != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", got.Code())
	}
}

func TestErrorMapper_NonVerbose(t *testing.T) {
	mapper := &ErrorMapper{VerboseErrors: false}
	err := connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("field 'x' is too short"))
	result := mapper.MapConnectError(err)
	// Should NOT contain field-level details
	assertErrorResult(t, result, "INVALID_ARGUMENT", "invalid input")
}

func TestErrorMapper_Verbose(t *testing.T) {
	mapper := &ErrorMapper{VerboseErrors: true}
	err := connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("bad input"))
	result := mapper.MapConnectError(err)
	if result == nil {
		t.Error("expected non-nil result")
	}
}
