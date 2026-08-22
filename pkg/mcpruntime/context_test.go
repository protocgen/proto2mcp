package mcpruntime

import (
	"net/http"
	"testing"
)

func TestFilterHeaders(t *testing.T) {
	allowlist := map[string]bool{
		"allowed-header": true,
	}

	// Test with allowlist filters correctly
	headers := http.Header{
		"Allowed-Header": []string{"val1"},
		"Blocked-Header": []string{"val2"},
	}
	filtered := FilterHeaders(headers, allowlist)
	if len(filtered) != 1 {
		t.Errorf("Expected 1 header, got %d", len(filtered))
	}
	if _, ok := filtered["Allowed-Header"]; !ok {
		t.Errorf("Expected Allowed-Header to be present")
	}

	// Test with nil allowlist returns all headers
	filteredNil := FilterHeaders(headers, nil)
	if len(filteredNil) != 2 {
		t.Errorf("Expected 2 headers, got %d", len(filteredNil))
	}

	// Test with empty headers returns empty
	emptyHeaders := http.Header{}
	filteredEmpty := FilterHeaders(emptyHeaders, allowlist)
	if len(filteredEmpty) != 0 {
		t.Errorf("Expected 0 headers, got %d", len(filteredEmpty))
	}
}

func TestDefaultHeaderAllowlist(t *testing.T) {
	expectedKeys := []string{
		"authorization",
		"x-request-id",
		"traceparent",
		"tracestate",
	}

	for _, k := range expectedKeys {
		if !DefaultHeaderAllowlist[k] {
			t.Errorf("Expected DefaultHeaderAllowlist to contain %s", k)
		}
	}
}
