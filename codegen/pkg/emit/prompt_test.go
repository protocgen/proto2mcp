package emit

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dave/jennifer/jen"
	"github.com/protocgen/proto2mcp/codegen/pkg/extract"
)

func TestGeneratePrompts(t *testing.T) {
	prompts := []extract.PromptIR{
		{
			Name:        "OnboardPatient",
			Description: "Onboard a new patient",
			Tools:       []string{"PatientService_CreatePatient", "BillingService_CreateAccount"},
			Arguments: []extract.PromptArgIR{
				{
					Name:        "patient_name",
					Description: "Name of the patient",
					Required:    true,
				},
				{
					Name:        "insurance_id",
					Description: "Insurance ID",
					Required:    false,
				},
			},
		},
	}

	f := jen.NewFilePath("generated")
	GeneratePrompts(f, prompts, "RegisterPatientPrompts")

	var buf bytes.Buffer
	if err := f.Render(&buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "type OnboardPatientPromptHandler interface") {
		t.Errorf("Missing interface:\n%s", out)
	}
	if !strings.Contains(out, "func RegisterPatientPrompts") {
		t.Errorf("Missing register func:\n%s", out)
	}

	goldenFile := filepath.Join("testdata", "prompts.golden")
	if *update {
		if err := os.MkdirAll("testdata", 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenFile, buf.Bytes(), 0644); err != nil {
			t.Fatal(err)
		}
	}

	expected, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("Failed to read golden file %s: %v", goldenFile, err)
	}

	got := strings.ReplaceAll(out, "\r\n", "\n")
	wantStr := strings.ReplaceAll(string(expected), "\r\n", "\n")
	if got != wantStr {
		t.Errorf("Output mismatch. Run tests with -update to update golden file. Got:\n%s", got)
	}
}
