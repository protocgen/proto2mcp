package emit

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/protocgen/proto2mcp/codegen/pkg/extract"
)

var update = flag.Bool("update", false, "update golden files")

func TestGoldenFiles(t *testing.T) {
	tests := []struct {
		name   string
		golden string
		infos  []ServiceEmitInfo
	}{
		{
			name:   "simple_service",
			golden: "simple_service.golden",
			infos:  []ServiceEmitInfo{simpleServiceInfo()},
		},
		{
			name:   "multi_method",
			golden: "multi_method.golden",
			infos:  []ServiceEmitInfo{multiMethodInfo()},
		},
		{
			name:   "with_connect",
			golden: "with_connect.golden",
			infos:  []ServiceEmitInfo{withConnectInfo()},
		},
		{
			name:   "wellknown_types",
			golden: "wellknown_types.golden",
			infos:  []ServiceEmitInfo{wellKnownTypesInfo()},
		},
		{
			name:   "empty_service",
			golden: "empty_service.golden",
			infos:  []ServiceEmitInfo{emptyServiceInfo()},
		},
		{
			name:   "deprecated_method",
			golden: "deprecated_method.golden",
			infos:  []ServiceEmitInfo{deprecatedMethodInfo()},
		},
		{
			name:   "all_field_types",
			golden: "all_field_types.golden",
			infos:  []ServiceEmitInfo{allFieldTypesInfo()},
		},
		{
			name:   "no_tools",
			golden: "no_tools.golden",
			infos:  []ServiceEmitInfo{noToolsInfo()},
		},
		{
			name:   "multiple_services",
			golden: "multiple_services.golden",
			infos:  multiServiceInfo(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := GenerateFile(tt.infos)
			var buf strings.Builder
			if err := f.Render(&buf); err != nil {
				t.Fatalf("Render failed: %v", err)
			}
			got := buf.String()

			goldenPath := filepath.Join("testdata", tt.golden)

			if *update {
				if err := os.MkdirAll("testdata", 0o755); err != nil {
					t.Fatalf("creating testdata dir: %v", err)
				}
				if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
					t.Fatalf("writing golden file: %v", err)
				}
				return
			}

			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("reading golden file %s: %v\nRun with -update to create it", goldenPath, err)
			}

			// Normalize line endings for cross-platform compatibility.
			// Jennifer's Render() may produce \r\n on Windows.
			got = strings.ReplaceAll(got, "\r\n", "\n")
			wantStr := strings.ReplaceAll(string(want), "\r\n", "\n")

			if got != wantStr {
				t.Errorf("output mismatch for %s\nRun with -update to update golden file", tt.golden)
				// Show first diff line
				gotLines := strings.Split(got, "\n")
				wantLines := strings.Split(wantStr, "\n")
				for i := 0; i < len(gotLines) && i < len(wantLines); i++ {
					if gotLines[i] != wantLines[i] {
						t.Errorf("first diff at line %d:\n  got:  %s\n  want: %s", i+1, gotLines[i], wantLines[i])
						break
					}
				}
			}
		})
	}
}

// simpleServiceInfo returns a single-method service
func simpleServiceInfo() ServiceEmitInfo {
	return ServiceEmitInfo{
		Service: extract.ServiceIR{
			Name:        "PatientService",
			FullName:    "healthcare.v1.PatientService",
			Description: "Manages patient records",
		},
		Tools: []ToolEmitInfo{{
			Tool: extract.ToolIR{
				Name:           "PatientService_GetPatient",
				MethodName:     "GetPatient",
				Description:    "Get a patient by ID",
				InputTypeName:  "healthcare.v1.GetPatientRequest",
				OutputTypeName: "healthcare.v1.GetPatientResponse",
				InputSchema:    []byte(`{"type":"object","properties":{"patient_id":{"type":"string","description":"The patient ID","minLength":1}}}`),
			},
			InputType:  TypeRef{ImportPath: "github.com/example/gen/healthcare/v1", TypeName: "GetPatientRequest"},
			OutputType: TypeRef{ImportPath: "github.com/example/gen/healthcare/v1", TypeName: "GetPatientResponse"},
		}},
		GoPackage:    "server",
		GoImportPath: "github.com/example/cmd/server",
	}
}

// multiMethodInfo returns a service with multiple methods
func multiMethodInfo() ServiceEmitInfo {
	return ServiceEmitInfo{
		Service: extract.ServiceIR{
			Name:     "OrderService",
			FullName: "commerce.v1.OrderService",
		},
		Tools: []ToolEmitInfo{
			{
				Tool: extract.ToolIR{
					Name:           "OrderService_CreateOrder",
					MethodName:     "CreateOrder",
					Description:    "Create a new order",
					InputTypeName:  "commerce.v1.CreateOrderRequest",
					OutputTypeName: "commerce.v1.CreateOrderResponse",
					InputSchema:    []byte(`{"type":"object","properties":{"item":{"type":"string"},"quantity":{"type":"integer"}}}`),
				},
				InputType:  TypeRef{ImportPath: "github.com/example/gen/commerce/v1", TypeName: "CreateOrderRequest"},
				OutputType: TypeRef{ImportPath: "github.com/example/gen/commerce/v1", TypeName: "CreateOrderResponse"},
			},
			{
				Tool: extract.ToolIR{
					Name:           "OrderService_GetOrder",
					MethodName:     "GetOrder",
					Description:    "Get an order by ID",
					InputTypeName:  "commerce.v1.GetOrderRequest",
					OutputTypeName: "commerce.v1.GetOrderResponse",
					InputSchema:    []byte(`{"type":"object","properties":{"order_id":{"type":"string"}}}`),
				},
				InputType:  TypeRef{ImportPath: "github.com/example/gen/commerce/v1", TypeName: "GetOrderRequest"},
				OutputType: TypeRef{ImportPath: "github.com/example/gen/commerce/v1", TypeName: "GetOrderResponse"},
			},
			{
				Tool: extract.ToolIR{
					Name:           "OrderService_CancelOrder",
					MethodName:     "CancelOrder",
					Description:    "Cancel an existing order",
					InputTypeName:  "commerce.v1.CancelOrderRequest",
					OutputTypeName: "commerce.v1.CancelOrderResponse",
					InputSchema:    []byte(`{"type":"object","properties":{"order_id":{"type":"string"},"reason":{"type":"string"}}}`),
				},
				InputType:  TypeRef{ImportPath: "github.com/example/gen/commerce/v1", TypeName: "CancelOrderRequest"},
				OutputType: TypeRef{ImportPath: "github.com/example/gen/commerce/v1", TypeName: "CancelOrderResponse"},
			},
		},
		GoPackage:    "server",
		GoImportPath: "github.com/example/cmd/server",
	}
}

// withConnectInfo returns a service with ConnectRPC forwarding enabled
func withConnectInfo() ServiceEmitInfo {
	info := simpleServiceInfo()
	info.GenerateConnect = true
	return info
}

// wellKnownTypesInfo returns a service using well-known types in input
func wellKnownTypesInfo() ServiceEmitInfo {
	return ServiceEmitInfo{
		Service: extract.ServiceIR{
			Name:     "AuditService",
			FullName: "audit.v1.AuditService",
		},
		Tools: []ToolEmitInfo{{
			Tool: extract.ToolIR{
				Name:           "AuditService_GetEvents",
				MethodName:     "GetEvents",
				Description:    "Get audit events in a time range",
				InputTypeName:  "audit.v1.GetEventsRequest",
				OutputTypeName: "audit.v1.GetEventsResponse",
				InputSchema:    []byte(`{"type":"object","properties":{"start_time":{"type":"string","format":"date-time"},"end_time":{"type":"string","format":"date-time"},"page_size":{"type":"integer"}}}`),
			},
			InputType:  TypeRef{ImportPath: "github.com/example/gen/audit/v1", TypeName: "GetEventsRequest"},
			OutputType: TypeRef{ImportPath: "github.com/example/gen/audit/v1", TypeName: "GetEventsResponse"},
		}},
		GoPackage:    "server",
		GoImportPath: "github.com/example/cmd/server",
	}
}

// emptyServiceInfo returns a service with no tools
func emptyServiceInfo() ServiceEmitInfo {
	return ServiceEmitInfo{
		Service: extract.ServiceIR{
			Name:     "EmptyService",
			FullName: "empty.v1.EmptyService",
		},
		GoPackage:    "server",
		GoImportPath: "github.com/example/cmd/server",
	}
}

// deprecatedMethodInfo returns a service with a deprecated method
func deprecatedMethodInfo() ServiceEmitInfo {
	return ServiceEmitInfo{
		Service: extract.ServiceIR{
			Name:     "LegacyService",
			FullName: "legacy.v1.LegacyService",
		},
		Tools: []ToolEmitInfo{{
			Tool: extract.ToolIR{
				Name:           "LegacyService_OldMethod",
				MethodName:     "OldMethod",
				Description:    "[DEPRECATED] An old method",
				InputTypeName:  "legacy.v1.OldRequest",
				OutputTypeName: "legacy.v1.OldResponse",
				InputSchema:    []byte(`{"type":"object"}`),
				IsDeprecated:   true,
			},
			InputType:  TypeRef{ImportPath: "github.com/example/gen/legacy/v1", TypeName: "OldRequest"},
			OutputType: TypeRef{ImportPath: "github.com/example/gen/legacy/v1", TypeName: "OldResponse"},
		}},
		GoPackage:    "server",
		GoImportPath: "github.com/example/cmd/server",
	}
}

// allFieldTypesInfo returns a service with various JSON schema types
func allFieldTypesInfo() ServiceEmitInfo {
	return ServiceEmitInfo{
		Service: extract.ServiceIR{
			Name:     "ComplexService",
			FullName: "complex.v1.ComplexService",
		},
		Tools: []ToolEmitInfo{{
			Tool: extract.ToolIR{
				Name:           "ComplexService_DoComplex",
				MethodName:     "DoComplex",
				Description:    "Does complex things",
				InputTypeName:  "complex.v1.ComplexRequest",
				OutputTypeName: "complex.v1.ComplexResponse",
				InputSchema:    []byte(`{"type":"object","properties":{"string_field":{"type":"string"},"int_field":{"type":"integer"},"bool_field":{"type":"boolean"},"array_field":{"type":"array","items":{"type":"string"}},"object_field":{"type":"object","properties":{"nested":{"type":"string"}}}}}`),
			},
			InputType:  TypeRef{ImportPath: "github.com/example/gen/complex/v1", TypeName: "ComplexRequest"},
			OutputType: TypeRef{ImportPath: "github.com/example/gen/complex/v1", TypeName: "ComplexResponse"},
		}},
		GoPackage:    "server",
		GoImportPath: "github.com/example/cmd/server",
	}
}

// noToolsInfo returns a service with no tools (edge case)
func noToolsInfo() ServiceEmitInfo {
	return ServiceEmitInfo{
		Service: extract.ServiceIR{
			Name:     "SkippedService",
			FullName: "skipped.v1.SkippedService",
		},
		Tools:        []ToolEmitInfo{},
		GoPackage:    "server",
		GoImportPath: "github.com/example/cmd/server",
	}
}

// multiServiceInfo returns multiple services
func multiServiceInfo() []ServiceEmitInfo {
	return []ServiceEmitInfo{
		{
			Service: extract.ServiceIR{
				Name:     "ServiceA",
				FullName: "multi.v1.ServiceA",
			},
			Tools: []ToolEmitInfo{{
				Tool: extract.ToolIR{
					Name:           "ServiceA_MethodA",
					MethodName:     "MethodA",
					Description:    "Method A",
					InputTypeName:  "multi.v1.RequestA",
					OutputTypeName: "multi.v1.ResponseA",
					InputSchema:    []byte(`{"type":"object"}`),
				},
				InputType:  TypeRef{ImportPath: "github.com/example/gen/multi/v1", TypeName: "RequestA"},
				OutputType: TypeRef{ImportPath: "github.com/example/gen/multi/v1", TypeName: "ResponseA"},
			}},
			GoPackage:    "server",
			GoImportPath: "github.com/example/cmd/server",
		},
		{
			Service: extract.ServiceIR{
				Name:     "ServiceB",
				FullName: "multi.v1.ServiceB",
			},
			Tools: []ToolEmitInfo{{
				Tool: extract.ToolIR{
					Name:           "ServiceB_MethodB",
					MethodName:     "MethodB",
					Description:    "Method B",
					InputTypeName:  "multi.v1.RequestB",
					OutputTypeName: "multi.v1.ResponseB",
					InputSchema:    []byte(`{"type":"object"}`),
				},
				InputType:  TypeRef{ImportPath: "github.com/example/gen/multi/v1", TypeName: "RequestB"},
				OutputType: TypeRef{ImportPath: "github.com/example/gen/multi/v1", TypeName: "ResponseB"},
			}},
			GoPackage:    "server",
			GoImportPath: "github.com/example/cmd/server",
		},
	}
}
