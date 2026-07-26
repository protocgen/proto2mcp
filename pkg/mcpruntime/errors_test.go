package mcpruntime

import (
	"encoding/json"
	"fmt"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

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

func TestUnmarshalToolInput_Valid(t *testing.T) {
	// google.protobuf.Struct uses direct JSON representation in protojson
	req := ToolRequest{Arguments: []byte(`{"name":"foo"}`)}

	dest := &structpb.Struct{}
	err := UnmarshalToolInput(req, dest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dest.Fields["name"].GetStringValue() != "foo" {
		t.Fatalf("expected 'foo', got %v", dest.Fields["name"].GetStringValue())
	}
}

func TestUnmarshalToolInput_InvalidJSON(t *testing.T) {
	req := ToolRequest{Arguments: []byte(`{invalid`)}
	dest := &structpb.Struct{}
	err := UnmarshalToolInput(req, dest)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestUnmarshalToolInput_Empty(t *testing.T) {
	req := ToolRequest{Arguments: []byte{}}
	dest := &structpb.Struct{}
	err := UnmarshalToolInput(req, dest)
	if err == nil {
		t.Fatal("expected error for empty arguments")
	}
}

func TestMarshalToolResult_Populated(t *testing.T) {
	msg, _ := structpb.NewStruct(map[string]interface{}{"result": "success"})
	res, err := MarshalToolResult(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatal("expected IsError=false")
	}

	// Struct serializes as plain JSON object via protojson
	var parsed map[string]interface{}
	if err := json.Unmarshal(res.Content, &parsed); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if parsed["result"] != "success" {
		t.Fatalf("expected result=success, got: %v", parsed)
	}
}

func TestMarshalToolResult_Nil(t *testing.T) {
	res, err := MarshalToolResult(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatal("expected IsError=false")
	}
	if string(res.Content) != "{}" {
		t.Fatalf("expected '{}', got %q", string(res.Content))
	}
}

func TestNewErrorResultWithDetails_EmptyDetails(t *testing.T) {
	res := NewErrorResultWithDetails("err", "CODE", []string{})
	ec := parseErrorContent(t, res)
	if len(ec.Violations) != 0 {
		t.Fatalf("expected 0 violations, got %d", len(ec.Violations))
	}
}

func TestNewErrorResultWithDetails_LongDetail(t *testing.T) {
	longStr := ""
	for i := 0; i < 1000; i++ {
		longStr += "x"
	}
	res := NewErrorResultWithDetails("err", "CODE", []string{longStr})
	ec := parseErrorContent(t, res)
	if len(ec.Violations) != 1 || ec.Violations[0] != longStr {
		t.Fatalf("expected violation to be preserved")
	}
}
