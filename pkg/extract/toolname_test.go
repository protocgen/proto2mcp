package extract

import (
	"strings"
	"testing"
)

func TestValidateToolName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		wantMsg  string
	}{
		{
			name:  "valid simple name",
			input: "PatientService_GetPatient",
		},
		{
			name:  "valid with hyphen",
			input: "patient-service_get-patient",
		},
		{
			name:  "valid single char",
			input: "x",
		},
		{
			name:  "valid at max length (64)",
			input: strings.Repeat("a", 64),
		},
		{
			name:    "empty name",
			input:   "",
			wantErr: true,
			wantMsg: "empty",
		},
		{
			name:    "exceeds max length (65)",
			input:   strings.Repeat("a", 65),
			wantErr: true,
			wantMsg: "exceeds MCP maximum",
		},
		{
			name:    "way too long (100)",
			input:   strings.Repeat("b", 100),
			wantErr: true,
			wantMsg: "exceeds MCP maximum",
		},
		{
			name:    "contains dots",
			input:   "service.method",
			wantErr: true,
			wantMsg: "invalid characters",
		},
		{
			name:    "contains spaces",
			input:   "service method",
			wantErr: true,
			wantMsg: "invalid characters",
		},
		{
			name:    "contains slash",
			input:   "service/method",
			wantErr: true,
			wantMsg: "invalid characters",
		},
		{
			name:    "contains colon",
			input:   "service:method",
			wantErr: true,
			wantMsg: "invalid characters",
		},
		{
			name:    "unicode characters",
			input:   "service_méthod",
			wantErr: true,
			wantMsg: "invalid characters",
		},
		{
			name:  "all valid char types",
			input: "abcXYZ_019-test",
		},
		{
			name:    "realistic long name that exceeds limit",
			input:   "HealthcarePatientManagementServiceV2_GetComprehensiveMedicalHistoryRecordByPatientIdentifier",
			wantErr: true,
			wantMsg: "exceeds MCP maximum",
		},
		{
			// Regression: multibyte characters. 30 runes = 60 bytes (under 64).
			// Rejected by regex (invalid chars), NOT by length check.
			name:    "multibyte under byte limit",
			input:   strings.Repeat("é", 30),
			wantErr: true,
			wantMsg: "invalid characters",
		},
		{
			// Regression: multibyte at boundary. 40 runes = 80 bytes (over 64).
			// Gets BOTH length and regex warnings.
			name:    "multibyte over byte limit",
			input:   strings.Repeat("ñ", 40),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := ValidateToolName(tt.input)

			hasError := false
			for _, w := range warnings {
				if w.Severity == WarnError {
					hasError = true
					if tt.wantMsg != "" && !strings.Contains(w.Message, tt.wantMsg) {
						t.Errorf("expected message containing %q, got %q", tt.wantMsg, w.Message)
					}
				}
			}

			if tt.wantErr && !hasError {
				t.Errorf("expected WarnError for %q, got none", tt.input)
			}
			if !tt.wantErr && hasError {
				t.Errorf("unexpected WarnError for %q: %v", tt.input, warnings)
			}
		})
	}
}

func TestValidateToolName_Constants(t *testing.T) {
	if MaxToolNameLength != 64 {
		t.Errorf("expected MaxToolNameLength=64, got %d", MaxToolNameLength)
	}
}
