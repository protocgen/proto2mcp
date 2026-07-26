package emit

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/protocgen/proto2mcp/codegen/pkg/extract"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// ============================================================================
// Helpers — build protogen.Plugin from descriptorpb
// ============================================================================

func e2eBuildPlugin(t *testing.T, files ...*descriptorpb.FileDescriptorProto) *protogen.Plugin {
	t.Helper()
	req := &pluginpb.CodeGeneratorRequest{}
	for _, fd := range files {
		req.ProtoFile = append(req.ProtoFile, fd)
		req.FileToGenerate = append(req.FileToGenerate, fd.GetName())
	}
	plugin, err := protogen.Options{}.New(req)
	if err != nil {
		t.Fatalf("failed to create protogen.Plugin: %v", err)
	}
	return plugin
}

func e2eStrPtr(s string) *string                                  { return &s }
func e2eInt32Ptr(i int32) *int32                                   { return &i }
func e2eBoolPtr(b bool) *bool                                      { return &b }
func e2eTypePtr(t descriptorpb.FieldDescriptorProto_Type) *descriptorpb.FieldDescriptorProto_Type {
	return &t
}
func e2eLabelPtr(l descriptorpb.FieldDescriptorProto_Label) *descriptorpb.FieldDescriptorProto_Label {
	return &l
}

func e2eField(name string, num int32, typ descriptorpb.FieldDescriptorProto_Type) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name: e2eStrPtr(name), Number: e2eInt32Ptr(num),
		Label: e2eLabelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
		Type: e2eTypePtr(typ), JsonName: e2eStrPtr(name),
	}
}

func e2eMsg(name string, fields ...*descriptorpb.FieldDescriptorProto) *descriptorpb.DescriptorProto {
	return &descriptorpb.DescriptorProto{Name: e2eStrPtr(name), Field: fields}
}

func e2eMethod(name, inType, outType string) *descriptorpb.MethodDescriptorProto {
	return &descriptorpb.MethodDescriptorProto{
		Name: e2eStrPtr(name), InputType: e2eStrPtr(inType), OutputType: e2eStrPtr(outType),
	}
}

// e2eBuildEmitInfos runs the full extract → emit-info resolution pipeline.
func e2eBuildEmitInfos(t *testing.T, plugin *protogen.Plugin) []ServiceEmitInfo {
	t.Helper()

	results, err := extract.FromPlugin(plugin)
	if err != nil {
		t.Fatalf("FromPlugin failed: %v", err)
	}

	var allInfos []ServiceEmitInfo
	for _, fileIR := range results {
		if fileIR.Skip {
			continue
		}
		protoFile := plugin.FilesByPath[fileIR.FileName]
		if protoFile == nil {
			continue
		}

		// Build method index for this file's services.
		svcMethodMap := make(map[string]map[string]*protogen.Method)
		for _, ps := range protoFile.Services {
			mm := make(map[string]*protogen.Method)
			for _, m := range ps.Methods {
				mm[string(m.Desc.Name())] = m
			}
			svcMethodMap[string(ps.Desc.FullName())] = mm
		}

		for _, svcIR := range fileIR.Services {
			methods := svcMethodMap[svcIR.FullName]
			var tools []ToolEmitInfo
			for _, toolIR := range svcIR.Tools {
				m := methods[toolIR.MethodName]
				tools = append(tools, ToolEmitInfo{
					Tool: toolIR,
					InputType: TypeRef{
						ImportPath: string(m.Input.GoIdent.GoImportPath),
						TypeName:   m.Input.GoIdent.GoName,
					},
					OutputType: TypeRef{
						ImportPath: string(m.Output.GoIdent.GoImportPath),
						TypeName:   m.Output.GoIdent.GoName,
					},
				})
			}
			allInfos = append(allInfos, ServiceEmitInfo{
				Service:      svcIR,
				Tools:        tools,
				GoPackage:    string(protoFile.GoPackageName),
				GoImportPath: string(protoFile.GoImportPath),
			})
		}
	}

	return allInfos
}

// ============================================================================
// E2E Golden Tests — descriptorpb → extract → emit → render → golden
// ============================================================================

func TestE2EGoldenFiles(t *testing.T) {
	tests := []struct {
		name   string
		golden string
		fd     *descriptorpb.FileDescriptorProto
	}{
		{
			name:   "e2e_simple",
			golden: "e2e_simple.golden",
			fd: &descriptorpb.FileDescriptorProto{
				Name: e2eStrPtr("test/v1/simple.proto"), Package: e2eStrPtr("test.v1"),
				Syntax: e2eStrPtr("proto3"),
				Options: &descriptorpb.FileOptions{
					GoPackage: e2eStrPtr("github.com/test/gen/test/v1;testv1"),
				},
				MessageType: []*descriptorpb.DescriptorProto{
					e2eMsg("GetPatientRequest",
						e2eField("patient_id", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING),
					),
					e2eMsg("GetPatientResponse",
						e2eField("name", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING),
						e2eField("age", 2, descriptorpb.FieldDescriptorProto_TYPE_INT32),
						e2eField("active", 3, descriptorpb.FieldDescriptorProto_TYPE_BOOL),
					),
					e2eMsg("ListPatientsRequest",
						e2eField("page_size", 1, descriptorpb.FieldDescriptorProto_TYPE_INT32),
					),
					e2eMsg("ListPatientsResponse",
						e2eField("count", 1, descriptorpb.FieldDescriptorProto_TYPE_INT32),
					),
				},
				Service: []*descriptorpb.ServiceDescriptorProto{{
					Name: e2eStrPtr("PatientService"),
					Method: []*descriptorpb.MethodDescriptorProto{
						e2eMethod("GetPatient", ".test.v1.GetPatientRequest", ".test.v1.GetPatientResponse"),
						e2eMethod("ListPatients", ".test.v1.ListPatientsRequest", ".test.v1.ListPatientsResponse"),
					},
				}},
			},
		},
		{
			name:   "e2e_all_types",
			golden: "e2e_all_types.golden",
			fd: &descriptorpb.FileDescriptorProto{
				Name: e2eStrPtr("test/v1/alltypes.proto"), Package: e2eStrPtr("test.v1"),
				Syntax: e2eStrPtr("proto3"),
				Options: &descriptorpb.FileOptions{
					GoPackage: e2eStrPtr("github.com/test/gen/test/v1;testv1"),
				},
				EnumType: []*descriptorpb.EnumDescriptorProto{{
					Name: e2eStrPtr("Status"),
					Value: []*descriptorpb.EnumValueDescriptorProto{
						{Name: e2eStrPtr("STATUS_UNSPECIFIED"), Number: e2eInt32Ptr(0)},
						{Name: e2eStrPtr("STATUS_ACTIVE"), Number: e2eInt32Ptr(1)},
						{Name: e2eStrPtr("STATUS_INACTIVE"), Number: e2eInt32Ptr(2)},
					},
				}},
				MessageType: []*descriptorpb.DescriptorProto{
					// Nested message
					{
						Name: e2eStrPtr("Address"),
						Field: []*descriptorpb.FieldDescriptorProto{
							e2eField("street", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING),
							e2eField("zip", 2, descriptorpb.FieldDescriptorProto_TYPE_STRING),
						},
					},
					// Request with all types
					{
						Name: e2eStrPtr("AllTypesRequest"),
						Field: []*descriptorpb.FieldDescriptorProto{
							e2eField("str_field", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING),
							e2eField("int32_field", 2, descriptorpb.FieldDescriptorProto_TYPE_INT32),
							e2eField("int64_field", 3, descriptorpb.FieldDescriptorProto_TYPE_INT64),
							e2eField("uint32_field", 4, descriptorpb.FieldDescriptorProto_TYPE_UINT32),
							e2eField("float_field", 5, descriptorpb.FieldDescriptorProto_TYPE_FLOAT),
							e2eField("double_field", 6, descriptorpb.FieldDescriptorProto_TYPE_DOUBLE),
							e2eField("bool_field", 7, descriptorpb.FieldDescriptorProto_TYPE_BOOL),
							e2eField("bytes_field", 8, descriptorpb.FieldDescriptorProto_TYPE_BYTES),
							// Enum field
							{
								Name: e2eStrPtr("status"), Number: e2eInt32Ptr(9),
								Label: e2eLabelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
								Type: e2eTypePtr(descriptorpb.FieldDescriptorProto_TYPE_ENUM),
								TypeName: e2eStrPtr(".test.v1.Status"), JsonName: e2eStrPtr("status"),
							},
							// Nested message field
							{
								Name: e2eStrPtr("address"), Number: e2eInt32Ptr(10),
								Label: e2eLabelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
								Type: e2eTypePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
								TypeName: e2eStrPtr(".test.v1.Address"), JsonName: e2eStrPtr("address"),
							},
							// Repeated string
							{
								Name: e2eStrPtr("tags"), Number: e2eInt32Ptr(11),
								Label: e2eLabelPtr(descriptorpb.FieldDescriptorProto_LABEL_REPEATED),
								Type: e2eTypePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING),
								JsonName: e2eStrPtr("tags"),
							},
							// Map field (references nested MetadataEntry)
							{
								Name: e2eStrPtr("metadata"), Number: e2eInt32Ptr(12),
								Label: e2eLabelPtr(descriptorpb.FieldDescriptorProto_LABEL_REPEATED),
								Type: e2eTypePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
								TypeName: e2eStrPtr(".test.v1.AllTypesRequest.MetadataEntry"), JsonName: e2eStrPtr("metadata"),
							},
						},
						NestedType: []*descriptorpb.DescriptorProto{
							{
								Name: e2eStrPtr("MetadataEntry"),
								Field: []*descriptorpb.FieldDescriptorProto{
									e2eField("key", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING),
									e2eField("value", 2, descriptorpb.FieldDescriptorProto_TYPE_STRING),
								},
								Options: &descriptorpb.MessageOptions{MapEntry: e2eBoolPtr(true)},
							},
						},
					},
					e2eMsg("AllTypesResponse",
						e2eField("ok", 1, descriptorpb.FieldDescriptorProto_TYPE_BOOL),
					),
				},
				Service: []*descriptorpb.ServiceDescriptorProto{{
					Name: e2eStrPtr("AllTypesService"),
					Method: []*descriptorpb.MethodDescriptorProto{
						e2eMethod("ProcessAllTypes", ".test.v1.AllTypesRequest", ".test.v1.AllTypesResponse"),
					},
				}},
			},
		},
		{
			name:   "e2e_multi_service",
			golden: "e2e_multi_service.golden",
			fd: &descriptorpb.FileDescriptorProto{
				Name: e2eStrPtr("test/v1/multi.proto"), Package: e2eStrPtr("test.v1"),
				Syntax: e2eStrPtr("proto3"),
				Options: &descriptorpb.FileOptions{
					GoPackage: e2eStrPtr("github.com/test/gen/test/v1;testv1"),
				},
				MessageType: []*descriptorpb.DescriptorProto{
					e2eMsg("PingRequest", e2eField("message", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING)),
					e2eMsg("PingResponse", e2eField("reply", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING)),
					e2eMsg("StatusRequest"),
					e2eMsg("StatusResponse", e2eField("healthy", 1, descriptorpb.FieldDescriptorProto_TYPE_BOOL)),
				},
				Service: []*descriptorpb.ServiceDescriptorProto{
					{
						Name: e2eStrPtr("PingService"),
						Method: []*descriptorpb.MethodDescriptorProto{
							e2eMethod("Ping", ".test.v1.PingRequest", ".test.v1.PingResponse"),
						},
					},
					{
						Name: e2eStrPtr("HealthService"),
						Method: []*descriptorpb.MethodDescriptorProto{
							e2eMethod("CheckStatus", ".test.v1.StatusRequest", ".test.v1.StatusResponse"),
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := e2eBuildPlugin(t, tt.fd)
			infos := e2eBuildEmitInfos(t, plugin)

			f := GenerateFile(infos)
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

			if got != string(want) {
				t.Errorf("output mismatch for %s\nRun with -update to update golden file", tt.golden)
				gotLines := strings.Split(got, "\n")
				wantLines := strings.Split(string(want), "\n")
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

// TestE2EGolden_ParsesAsValidGo verifies that every E2E golden file
// parses as syntactically valid Go code.
func TestE2EGolden_ParsesAsValidGo(t *testing.T) {
	goldens := []string{
		"e2e_simple.golden",
		"e2e_all_types.golden",
		"e2e_multi_service.golden",
	}

	for _, name := range goldens {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("testdata", name)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Skipf("golden file not found: %v (run with -update first)", err)
			}

			fset := token.NewFileSet()
			_, err = parser.ParseFile(fset, "generated.go", data, parser.AllErrors)
			if err != nil {
				t.Errorf("generated code is not valid Go:\n%v", err)
			}
		})
	}
}
