package mcpruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"connectrpc.com/connect"
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

	// Should NOT leak the segfault message
	assertErrorResult(t, result, "INTERNAL", "an internal error occurred while processing the request")
}

func TestMapConnectError_NonConnectError_ReturnsInternal(t *testing.T) {
	err := fmt.Errorf("random non-connect error")
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

func TestSanitizeErrorMessage_Clean(t *testing.T) {
	msg := sanitizeErrorMessage("field 'name' is required")
	if msg != "field 'name' is required" {
		t.Fatalf("expected clean passthrough, got %q", msg)
	}
}

func TestSanitizeErrorMessage_StackTrace(t *testing.T) {
	msg := sanitizeErrorMessage("goroutine 1 [running]:\nmain.main()")
	if msg != "invalid input parameters" {
		t.Fatalf("expected sanitized, got %q", msg)
	}
}

func TestSanitizeErrorMessage_GoFilePath(t *testing.T) {
	msg := sanitizeErrorMessage("error at /home/user/service/handler.go:142")
	if msg != "invalid input parameters" {
		t.Fatalf("expected sanitized for .go: pattern, got %q", msg)
	}
}

func TestSanitizeErrorMessage_AbsoluteFilePath(t *testing.T) {
	msg := sanitizeErrorMessage("failed to open /var/lib/data/patients.db")
	if msg != "invalid input parameters" {
		t.Fatalf("expected sanitized for absolute path, got %q", msg)
	}
}

func TestSanitizeErrorMessage_HostPort(t *testing.T) {
	msg := sanitizeErrorMessage("connection refused to backend-svc:50051")
	if msg != "invalid input parameters" {
		t.Fatalf("expected sanitized for host:port, got %q", msg)
	}
}

func TestSanitizeErrorMessage_Localhost(t *testing.T) {
	msg := sanitizeErrorMessage("dial tcp localhost:8080: connection refused")
	if msg != "invalid input parameters" {
		t.Fatalf("expected sanitized for localhost:port, got %q", msg)
	}
}

func TestSanitizeErrorMessage_TruncatesLong(t *testing.T) {
	long := ""
	for i := 0; i < 300; i++ {
		long += "x"
	}
	msg := sanitizeErrorMessage(long)
	if len(msg) > 210 { // 200 + "..."
		t.Fatalf("expected truncated to ~200 chars, got %d", len(msg))
	}
}

func TestSanitizeErrorMessage_MultiLine(t *testing.T) {
	msg := sanitizeErrorMessage("first line\nsecond line\nthird line")
	if msg != "first line" {
		t.Fatalf("expected first line only, got %q", msg)
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

func TestInvalidParamsError(t *testing.T) {
	result := InvalidParamsError(fmt.Errorf("name is required"))
	assertErrorResult(t, result, "INVALID_ARGUMENT", "invalid input: name is required")
}

func TestInternalError(t *testing.T) {
	result := InternalError("something broke")
	assertErrorResult(t, result, "INTERNAL", "something broke")
}

func TestInvalidParamsError_SanitizesInternals(t *testing.T) {
	result := InvalidParamsError(fmt.Errorf("failed at /home/user/svc/handler.go:42"))
	ec := parseErrorContent(t, result)
	if ec.Error != "invalid input: invalid input parameters" {
		t.Fatalf("expected sanitized error, got %q", ec.Error)
	}
}

func TestNewErrorResultWithDetails_MarshalFallback(t *testing.T) {
	// This tests the fallback path — errorContent should always marshal
	// successfully, but verify the result is valid JSON regardless.
	result := newErrorResultWithDetails("test error", "TEST", nil)
	if !result.IsError {
		t.Fatal("expected IsError=true")
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(result.Content, &parsed); err != nil {
		t.Fatalf("result content is not valid JSON: %v", err)
	}
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

// --- helpers ---

func parseErrorContent(t *testing.T, result *CallToolResult) errorContent {
	t.Helper()
	var ec errorContent
	if err := json.Unmarshal(result.Content, &ec); err != nil {
		t.Fatalf("failed to parse error content: %v\nraw: %s", err, string(result.Content))
	}
	return ec
}

func assertErrorResult(t *testing.T, result *CallToolResult, expectedCode, expectedError string) {
	t.Helper()
	if result == nil {
		t.Fatal("result is nil")
	}
	if !result.IsError {
		t.Fatal("expected IsError=true")
	}
	ec := parseErrorContent(t, result)
	if ec.Code != expectedCode {
		t.Fatalf("expected code %q, got %q", expectedCode, ec.Code)
	}
	if ec.Error != expectedError {
		t.Fatalf("expected error %q, got %q", expectedError, ec.Error)
	}
}

// --- context helpers for integration tests ---

func TestTenantRoundtrip(t *testing.T) {
	ctx := context.Background()

	// Initially empty.
	if got := TenantFromContext(ctx); got != "" {
		t.Fatalf("expected empty tenant, got %q", got)
	}

	// Set and retrieve.
	ctx = WithTenant(ctx, "tenant-abc")
	if got := TenantFromContext(ctx); got != "tenant-abc" {
		t.Fatalf("expected 'tenant-abc', got %q", got)
	}
}
