package mcpruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

func TestSanitizeErrorMessage_Clean(t *testing.T) {
	msg := SanitizeErrorMessage("field 'name' is required")
	if msg != "field 'name' is required" {
		t.Fatalf("expected clean passthrough, got %q", msg)
	}
}

func TestSanitizeErrorMessage_StackTrace(t *testing.T) {
	msg := SanitizeErrorMessage("goroutine 1 [running]:\nmain.main()")
	if msg != "invalid input parameters" {
		t.Fatalf("expected sanitized, got %q", msg)
	}
}

func TestSanitizeErrorMessage_GoFilePath(t *testing.T) {
	msg := SanitizeErrorMessage("error at /home/user/service/handler.go:142")
	if msg != "invalid input parameters" {
		t.Fatalf("expected sanitized for .go: pattern, got %q", msg)
	}
}

func TestSanitizeErrorMessage_AbsoluteFilePath(t *testing.T) {
	msg := SanitizeErrorMessage("failed to open /var/lib/data/patients.db")
	if msg != "invalid input parameters" {
		t.Fatalf("expected sanitized for absolute path, got %q", msg)
	}
}

func TestSanitizeErrorMessage_HostPort(t *testing.T) {
	msg := SanitizeErrorMessage("connection refused to backend-svc:50051")
	if msg != "invalid input parameters" {
		t.Fatalf("expected sanitized for host:port, got %q", msg)
	}
}

func TestSanitizeErrorMessage_Localhost(t *testing.T) {
	msg := SanitizeErrorMessage("dial tcp localhost:8080: connection refused")
	if msg != "invalid input parameters" {
		t.Fatalf("expected sanitized for localhost:port, got %q", msg)
	}
}

func TestSanitizeErrorMessage_TruncatesLong(t *testing.T) {
	long := ""
	for i := 0; i < 300; i++ {
		long += "x"
	}
	msg := SanitizeErrorMessage(long)
	if len(msg) > 210 { // 200 + "..."
		t.Fatalf("expected truncated to ~200 chars, got %d", len(msg))
	}
}

func TestSanitizeErrorMessage_MultiLine(t *testing.T) {
	msg := SanitizeErrorMessage("first line\nsecond line\nthird line")
	if msg != "first line" {
		t.Fatalf("expected first line only, got %q", msg)
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
	result := NewErrorResultWithDetails("test error", "TEST", nil)
	if !result.IsError {
		t.Fatal("expected IsError=true")
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(result.Content, &parsed); err != nil {
		t.Fatalf("result content is not valid JSON: %v", err)
	}
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
