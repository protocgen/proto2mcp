package emit

import (
	"strings"
	"testing"

	"github.com/protocgen/proto2mcp/pkg/extract"
)

// testServiceEmitInfo creates a ServiceEmitInfo for testing.
func testServiceEmitInfo() ServiceEmitInfo {
	return ServiceEmitInfo{
		Service: extract.ServiceIR{
			Name:     "PatientService",
			FullName: "myapp.v1.PatientService",
			Description: "Manages patient records",
		},
		Tools: []ToolEmitInfo{
			{
				Tool: extract.ToolIR{
					Name:           "PatientService_GetPatient",
					MethodName:     "GetPatient",
					Description:    "Get a patient by ID",
					InputTypeName:  "myapp.v1.GetPatientRequest",
					OutputTypeName: "myapp.v1.GetPatientResponse",
					InputSchema:    []byte(`{"type":"object","properties":{"patient_id":{"type":"string"}}}`),
				},
				InputType: TypeRef{
					ImportPath: "github.com/myapp/gen/myapp/v1",
					TypeName:   "GetPatientRequest",
				},
				OutputType: TypeRef{
					ImportPath: "github.com/myapp/gen/myapp/v1",
					TypeName:   "GetPatientResponse",
				},
			},
			{
				Tool: extract.ToolIR{
					Name:           "PatientService_ListPatients",
					MethodName:     "ListPatients",
					Description:    "List all patients",
					InputTypeName:  "myapp.v1.ListPatientsRequest",
					OutputTypeName: "myapp.v1.ListPatientsResponse",
					InputSchema:    []byte(`{"type":"object","properties":{"page_size":{"type":"integer"}}}`),
				},
				InputType: TypeRef{
					ImportPath: "github.com/myapp/gen/myapp/v1",
					TypeName:   "ListPatientsRequest",
				},
				OutputType: TypeRef{
					ImportPath: "github.com/myapp/gen/myapp/v1",
					TypeName:   "ListPatientsResponse",
				},
			},
		},
		GoPackage:    "myappv1",
		GoImportPath: "github.com/myapp/gen/myapp/v1",
	}
}

func TestGenerateFile_ContainsInterface(t *testing.T) {
	info := testServiceEmitInfo()
	f := GenerateFile([]ServiceEmitInfo{info})

	var buf strings.Builder
	if err := f.Render(&buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	output := buf.String()

	// Must contain the handler interface.
	if !strings.Contains(output, "PatientServiceMCPHandler") {
		t.Error("expected PatientServiceMCPHandler interface in output")
	}
	// Must contain method signatures.
	if !strings.Contains(output, "GetPatient(") {
		t.Error("expected GetPatient method in interface")
	}
	if !strings.Contains(output, "ListPatients(") {
		t.Error("expected ListPatients method in interface")
	}
}

func TestGenerateFile_ContainsRegisterFunc(t *testing.T) {
	info := testServiceEmitInfo()
	f := GenerateFile([]ServiceEmitInfo{info})

	var buf strings.Builder
	if err := f.Render(&buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "RegisterPatientServiceMCP(") {
		t.Error("expected RegisterPatientServiceMCP function in output")
	}
	// Must reference the runtime registry interface.
	if !strings.Contains(output, "Registry") {
		t.Error("expected Registry interface in RegisterPatientServiceMCP")
	}
	// Must reference UnmarshalToolInput.
	if !strings.Contains(output, "UnmarshalToolInput") {
		t.Error("expected UnmarshalToolInput call in handler")
	}
	// Must reference MapConnectError.
	if !strings.Contains(output, "MapConnectError") {
		t.Error("expected MapConnectError call in handler")
	}
}

func TestGenerateFile_ContainsToolConstants(t *testing.T) {
	info := testServiceEmitInfo()
	f := GenerateFile([]ServiceEmitInfo{info})

	var buf strings.Builder
	if err := f.Render(&buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	output := buf.String()

	// Name constants.
	if !strings.Contains(output, "PatientService_GetPatientName") {
		t.Error("expected tool name constant")
	}
	if !strings.Contains(output, "PatientService_GetPatientDescription") {
		t.Error("expected tool description constant")
	}
	if !strings.Contains(output, "PatientService_GetPatientSchema") {
		t.Error("expected tool schema variable")
	}
}

func TestGenerateFile_ContainsGeneratedComment(t *testing.T) {
	info := testServiceEmitInfo()
	f := GenerateFile([]ServiceEmitInfo{info})

	var buf strings.Builder
	if err := f.Render(&buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "DO NOT EDIT") {
		t.Error("expected 'DO NOT EDIT' comment in generated file")
	}
}

func TestGenerateFile_NoConnectByDefault(t *testing.T) {
	info := testServiceEmitInfo()
	f := GenerateFile([]ServiceEmitInfo{info})

	var buf strings.Builder
	if err := f.Render(&buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	output := buf.String()

	if strings.Contains(output, "ForwardToConnect") {
		t.Error("expected no ConnectRPC forwarder when GenerateConnect is false")
	}
}

func TestGenerateFile_WithConnect(t *testing.T) {
	info := testServiceEmitInfo()
	info.GenerateConnect = true
	f := GenerateFile([]ServiceEmitInfo{info})

	var buf strings.Builder
	if err := f.Render(&buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "PatientServiceForwardToConnect") {
		t.Error("expected ConnectRPC forwarder when GenerateConnect is true")
	}
	if !strings.Contains(output, "PatientServiceServiceClient") {
		t.Error("expected Connect client interface type")
	}
}

func TestGenerateFile_TypeImports(t *testing.T) {
	// Use a different output package than the proto types package
	// so jennifer emits the import.
	info := testServiceEmitInfo()
	info.GoPackage = "server"
	info.GoImportPath = "github.com/myapp/cmd/server"
	f := GenerateFile([]ServiceEmitInfo{info})

	var buf strings.Builder
	if err := f.Render(&buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	output := buf.String()

	// Must import the runtime package.
	if !strings.Contains(output, "github.com/protocgen/proto2mcp/pkg/mcpruntime") {
		t.Error("expected mcpruntime import")
	}
	// Must import the proto types package (since output package differs).
	if !strings.Contains(output, "github.com/myapp/gen/myapp/v1") {
		t.Error("expected proto types import")
	}
	// Must import context.
	if !strings.Contains(output, `"context"`) {
		t.Error("expected context import")
	}
}

func TestTypeRef(t *testing.T) {
	ref := TypeRef{
		ImportPath: "github.com/example/pkg",
		TypeName:   "MyType",
	}
	if ref.ImportPath != "github.com/example/pkg" {
		t.Errorf("unexpected ImportPath: %s", ref.ImportPath)
	}
	if ref.TypeName != "MyType" {
		t.Errorf("unexpected TypeName: %s", ref.TypeName)
	}
}

func TestGenerateFile_EmptyService(t *testing.T) {
	info := ServiceEmitInfo{
		Service: extract.ServiceIR{
			Name:     "EmptyService",
			FullName: "myapp.v1.EmptyService",
		},
		GoPackage:    "myappv1",
		GoImportPath: "github.com/myapp/gen/myapp/v1",
	}

	f := GenerateFile([]ServiceEmitInfo{info})
	var buf strings.Builder
	if err := f.Render(&buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	output := buf.String()

	// Should still have the interface (empty) and register func.
	if !strings.Contains(output, "EmptyServiceMCPHandler") {
		t.Error("expected empty interface")
	}
	if !strings.Contains(output, "RegisterEmptyServiceMCP") {
		t.Error("expected register function even with no tools")
	}
}

func TestGenerateFile_EmptyToolName(t *testing.T) {
	info := testServiceEmitInfo()
	info.Tools[0].Tool.Name = ""
	f := GenerateFile([]ServiceEmitInfo{info})

	var buf strings.Builder
	if err := f.Render(&buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	output := buf.String()

	// Should handle empty tool name without panicking, probably generates invalid code
	// but we just check it doesn't crash
	if output == "" {
		t.Error("expected output even with empty tool name")
	}
}

func TestGenerateFile_SpecialCharactersInDescription(t *testing.T) {
	info := testServiceEmitInfo()
	info.Tools[0].Tool.Description = "Has \"quotes\" and \n newlines and `backticks`"
	f := GenerateFile([]ServiceEmitInfo{info})

	var buf strings.Builder
	if err := f.Render(&buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	output := buf.String()

	// Verify the description doesn't cause render failure — the fact
	// we got here without Render error is the test. Jennifer handles
	// string escaping internally.
	if len(output) == 0 {
		t.Fatal("expected non-empty output")
	}
}

func TestGenerateFile_LargeSchema(t *testing.T) {
	info := testServiceEmitInfo()
	largeSchema := `{"type":"object","properties":{`
	for i := 0; i < 100; i++ {
		largeSchema += `"field` + string(rune(i)) + `":{"type":"string"}`
		if i < 99 {
			largeSchema += ","
		}
	}
	largeSchema += `}}`
	
	info.Tools[0].Tool.InputSchema = []byte(largeSchema)
	f := GenerateFile([]ServiceEmitInfo{info})

	var buf strings.Builder
	if err := f.Render(&buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "PatientService_GetPatientSchema = []byte") {
		t.Error("expected large schema to be handled correctly")
	}
}

func TestGenerateFile_ConnectForwarder_NoMethods(t *testing.T) {
	info := ServiceEmitInfo{
		Service: extract.ServiceIR{
			Name:     "EmptyConnectService",
			FullName: "empty.v1.EmptyConnectService",
		},
		GoPackage:       "emptyv1",
		GoImportPath:    "github.com/empty/gen/empty/v1",
		GenerateConnect: true,
	}

	f := GenerateFile([]ServiceEmitInfo{info})
	var buf strings.Builder
	if err := f.Render(&buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	output := buf.String()

	// Should have the Connect forwarder function but no method cases inside the switch
	if !strings.Contains(output, "EmptyConnectServiceForwardToConnect") {
		t.Error("expected forwarder function")
	}
}
