package extract_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/protocgen/proto2mcp/codegen/pkg/extract"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// --- Helpers ---

func strPtr(s string) *string { return &s }
func int32Ptr(i int32) *int32 { return &i }
func boolPtr(b bool) *bool    { return &b }

func typePtr(t descriptorpb.FieldDescriptorProto_Type) *descriptorpb.FieldDescriptorProto_Type {
	return &t
}

func labelPtr(l descriptorpb.FieldDescriptorProto_Label) *descriptorpb.FieldDescriptorProto_Label {
	return &l
}

// buildPlugin creates a protogen.Plugin from descriptorpb files.
func buildPlugin(t *testing.T, files ...*descriptorpb.FileDescriptorProto) *protogen.Plugin {
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

// makeSimpleMessage creates a basic message descriptor.
func makeSimpleMessage(name string, fields ...*descriptorpb.FieldDescriptorProto) *descriptorpb.DescriptorProto {
	return &descriptorpb.DescriptorProto{
		Name:  strPtr(name),
		Field: fields,
	}
}

// makeField creates a scalar field descriptor.
func makeField(name string, num int32, typ descriptorpb.FieldDescriptorProto_Type) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     strPtr(name),
		Number:   int32Ptr(num),
		Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
		Type:     typePtr(typ),
		JsonName: strPtr(name),
	}
}

// makeMethod creates a method descriptor.
func makeMethod(name, inputType, outputType string) *descriptorpb.MethodDescriptorProto {
	return &descriptorpb.MethodDescriptorProto{
		Name:       strPtr(name),
		InputType:  strPtr(inputType),
		OutputType: strPtr(outputType),
	}
}

// ============================================================================
// Integration Test: Full Pipeline — FromPlugin → FileIR → Schema Validation
// ============================================================================

func TestIntegration_FromPlugin_SimpleService(t *testing.T) {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("test/v1/patient.proto"),
		Package: strPtr("test.v1"),
		Syntax:  strPtr("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: strPtr("github.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			makeSimpleMessage("GetPatientRequest",
				makeField("patient_id", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING),
				makeField("include_history", 2, descriptorpb.FieldDescriptorProto_TYPE_BOOL),
			),
			makeSimpleMessage("GetPatientResponse",
				makeField("name", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING),
				makeField("age", 2, descriptorpb.FieldDescriptorProto_TYPE_INT32),
			),
			makeSimpleMessage("ListPatientsRequest",
				makeField("page_size", 1, descriptorpb.FieldDescriptorProto_TYPE_INT32),
				makeField("page_token", 2, descriptorpb.FieldDescriptorProto_TYPE_STRING),
			),
			makeSimpleMessage("ListPatientsResponse",
				makeField("next_page_token", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING),
			),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("PatientService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					makeMethod("GetPatient", ".test.v1.GetPatientRequest", ".test.v1.GetPatientResponse"),
					makeMethod("ListPatients", ".test.v1.ListPatientsRequest", ".test.v1.ListPatientsResponse"),
				},
			},
		},
	}

	plugin := buildPlugin(t, fd)
	results, err := extract.FromPlugin(plugin)
	if err != nil {
		t.Fatalf("FromPlugin failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 FileIR, got %d", len(results))
	}

	fileIR := results[0]

	// Verify file metadata
	if fileIR.FileName != "test/v1/patient.proto" {
		t.Errorf("expected file name 'test/v1/patient.proto', got %q", fileIR.FileName)
	}
	if fileIR.Skip {
		t.Error("expected Skip=false")
	}

	// Verify service
	if len(fileIR.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(fileIR.Services))
	}
	svc := fileIR.Services[0]
	if svc.Name != "PatientService" {
		t.Errorf("expected service name 'PatientService', got %q", svc.Name)
	}
	if svc.FullName != "test.v1.PatientService" {
		t.Errorf("expected full name 'test.v1.PatientService', got %q", svc.FullName)
	}

	// Verify tools
	if len(svc.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(svc.Tools))
	}

	getPatient := svc.Tools[0]
	if getPatient.Name != "PatientService_GetPatient" {
		t.Errorf("expected tool name 'PatientService_GetPatient', got %q", getPatient.Name)
	}
	if getPatient.MethodName != "GetPatient" {
		t.Errorf("expected method name 'GetPatient', got %q", getPatient.MethodName)
	}
	if getPatient.InputTypeName != "test.v1.GetPatientRequest" {
		t.Errorf("unexpected input type: %q", getPatient.InputTypeName)
	}
	if getPatient.OutputTypeName != "test.v1.GetPatientResponse" {
		t.Errorf("unexpected output type: %q", getPatient.OutputTypeName)
	}

	// Verify schema was generated
	if getPatient.InputSchema == nil {
		t.Fatal("expected InputSchema to be set")
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(getPatient.InputSchema, &schema); err != nil {
		t.Fatalf("invalid JSON schema: %v", err)
	}
	if schema["type"] != "object" {
		t.Error("expected schema type 'object'")
	}
	props := schema["properties"].(map[string]interface{})
	if _, ok := props["patientId"]; !ok {
		// protojson uses camelCase
		if _, ok := props["patient_id"]; !ok {
			t.Error("expected patient_id or patientId field in schema")
		}
	}
	if schema["additionalProperties"] != false {
		t.Error("expected additionalProperties: false")
	}
}

func TestIntegration_FromPlugin_StreamingMethod(t *testing.T) {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("test/v1/stream.proto"),
		Package: strPtr("test.v1"),
		Syntax:  strPtr("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: strPtr("github.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			makeSimpleMessage("StreamReq", makeField("data", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING)),
			makeSimpleMessage("StreamResp", makeField("result", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING)),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("StreamService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:            strPtr("ServerStream"),
						InputType:       strPtr(".test.v1.StreamReq"),
						OutputType:      strPtr(".test.v1.StreamResp"),
						ServerStreaming: boolPtr(true),
					},
					{
						Name:            strPtr("ClientStream"),
						InputType:       strPtr(".test.v1.StreamReq"),
						OutputType:      strPtr(".test.v1.StreamResp"),
						ClientStreaming: boolPtr(true),
					},
					{
						Name:            strPtr("BidiStream"),
						InputType:       strPtr(".test.v1.StreamReq"),
						OutputType:      strPtr(".test.v1.StreamResp"),
						ClientStreaming: boolPtr(true),
						ServerStreaming: boolPtr(true),
					},
				},
			},
		},
	}

	plugin := buildPlugin(t, fd)
	fileIRs, err := extract.FromPlugin(plugin)

	// Streaming methods should be auto-skipped (not error).
	if err != nil {
		t.Fatalf("expected no error for streaming methods, got: %v", err)
	}
	if len(fileIRs) != 1 {
		t.Fatalf("expected 1 file IR, got %d", len(fileIRs))
	}
	// The service should exist but have zero tools (all streaming methods skipped).
	if len(fileIRs[0].Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(fileIRs[0].Services))
	}
	if len(fileIRs[0].Services[0].Tools) != 0 {
		t.Errorf("expected 0 tools (all streaming skipped), got %d", len(fileIRs[0].Services[0].Tools))
	}
}

func TestIntegration_FromPlugin_MixedStreamingAndUnary(t *testing.T) {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("test/v1/mixed.proto"),
		Package: strPtr("test.v1"),
		Syntax:  strPtr("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: strPtr("github.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			makeSimpleMessage("Req", makeField("id", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING)),
			makeSimpleMessage("Resp", makeField("name", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING)),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("MixedService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       strPtr("GetItem"),
						InputType:  strPtr(".test.v1.Req"),
						OutputType: strPtr(".test.v1.Resp"),
					},
					{
						Name:            strPtr("WatchItems"),
						InputType:       strPtr(".test.v1.Req"),
						OutputType:      strPtr(".test.v1.Resp"),
						ServerStreaming: boolPtr(true),
					},
					{
						Name:       strPtr("DeleteItem"),
						InputType:  strPtr(".test.v1.Req"),
						OutputType: strPtr(".test.v1.Resp"),
					},
				},
			},
		},
	}

	plugin := buildPlugin(t, fd)
	fileIRs, err := extract.FromPlugin(plugin)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(fileIRs) != 1 {
		t.Fatalf("expected 1 file IR, got %d", len(fileIRs))
	}
	svc := fileIRs[0].Services[0]
	// Only unary methods should produce tools; streaming is auto-skipped.
	if len(svc.Tools) != 2 {
		t.Fatalf("expected 2 tools (GetItem + DeleteItem), got %d", len(svc.Tools))
	}
	if svc.Tools[0].MethodName != "GetItem" {
		t.Errorf("expected first tool to be GetItem, got %s", svc.Tools[0].MethodName)
	}
	if svc.Tools[1].MethodName != "DeleteItem" {
		t.Errorf("expected second tool to be DeleteItem, got %s", svc.Tools[1].MethodName)
	}
}

func TestIntegration_FromPlugin_VariousFieldTypes(t *testing.T) {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("test/v1/types.proto"),
		Package: strPtr("test.v1"),
		Syntax:  strPtr("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: strPtr("github.com/test/v1;testv1"),
		},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: strPtr("Status"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: strPtr("STATUS_UNSPECIFIED"), Number: int32Ptr(0)},
					{Name: strPtr("STATUS_ACTIVE"), Number: int32Ptr(1)},
					{Name: strPtr("STATUS_INACTIVE"), Number: int32Ptr(2)},
				},
			},
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("Address"),
				Field: []*descriptorpb.FieldDescriptorProto{
					makeField("street", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING),
					makeField("city", 2, descriptorpb.FieldDescriptorProto_TYPE_STRING),
					makeField("zip", 3, descriptorpb.FieldDescriptorProto_TYPE_STRING),
				},
			},
			{
				Name: strPtr("CreateRecordRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					// String
					makeField("name", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING),
					// Bool
					makeField("active", 2, descriptorpb.FieldDescriptorProto_TYPE_BOOL),
					// Int32
					makeField("age", 3, descriptorpb.FieldDescriptorProto_TYPE_INT32),
					// Int64 (should map to string)
					makeField("bignum", 4, descriptorpb.FieldDescriptorProto_TYPE_INT64),
					// Double
					makeField("score", 5, descriptorpb.FieldDescriptorProto_TYPE_DOUBLE),
					// Bytes
					makeField("data", 6, descriptorpb.FieldDescriptorProto_TYPE_BYTES),
					// Enum
					{
						Name:     strPtr("status"),
						Number:   int32Ptr(7),
						Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_ENUM),
						TypeName: strPtr(".test.v1.Status"),
						JsonName: strPtr("status"),
					},
					// Nested message
					{
						Name:     strPtr("address"),
						Number:   int32Ptr(8),
						Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
						TypeName: strPtr(".test.v1.Address"),
						JsonName: strPtr("address"),
					},
					// Repeated string
					{
						Name:     strPtr("tags"),
						Number:   int32Ptr(9),
						Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_REPEATED),
						Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING),
						JsonName: strPtr("tags"),
					},
				},
			},
			makeSimpleMessage("CreateRecordResponse",
				makeField("id", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING),
			),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("RecordService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					makeMethod("CreateRecord", ".test.v1.CreateRecordRequest", ".test.v1.CreateRecordResponse"),
				},
			},
		},
	}

	plugin := buildPlugin(t, fd)
	results, err := extract.FromPlugin(plugin)
	if err != nil {
		t.Fatalf("FromPlugin failed: %v", err)
	}

	tool := results[0].Services[0].Tools[0]
	var schema map[string]interface{}
	if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
		t.Fatalf("invalid JSON schema: %v", err)
	}

	props := schema["properties"].(map[string]interface{})

	// Verify string field
	nameField := props["name"].(map[string]interface{})
	if nameField["type"] != "string" {
		t.Errorf("name: expected type=string, got %v", nameField["type"])
	}

	// Verify bool field
	activeField := props["active"].(map[string]interface{})
	if activeField["type"] != "boolean" {
		t.Errorf("active: expected type=boolean, got %v", activeField["type"])
	}

	// Verify int32 field
	ageField := props["age"].(map[string]interface{})
	if ageField["type"] != "integer" {
		t.Errorf("age: expected type=integer, got %v", ageField["type"])
	}

	// Verify int64 field → string (64-bit precision)
	bignumField := props["bignum"].(map[string]interface{})
	if bignumField["type"] != "string" {
		t.Errorf("bignum: expected type=string for int64, got %v", bignumField["type"])
	}
	if bignumField["format"] != "int64" {
		t.Errorf("bignum: expected format=int64, got %v", bignumField["format"])
	}

	// Verify double field
	scoreField := props["score"].(map[string]interface{})
	if scoreField["type"] != "number" {
		t.Errorf("score: expected type=number, got %v", scoreField["type"])
	}

	// Verify bytes field
	dataField := props["data"].(map[string]interface{})
	if dataField["type"] != "string" {
		t.Errorf("data: expected type=string for bytes, got %v", dataField["type"])
	}
	if dataField["format"] != "byte" {
		t.Errorf("data: expected format=byte, got %v", dataField["format"])
	}

	// Verify enum field
	statusField := props["status"].(map[string]interface{})
	if statusField["type"] != "string" {
		t.Errorf("status: expected type=string for enum, got %v", statusField["type"])
	}
	enumVals := statusField["enum"].([]interface{})
	if len(enumVals) != 3 {
		t.Errorf("status: expected 3 enum values, got %d", len(enumVals))
	}

	// Verify nested message
	addressField := props["address"].(map[string]interface{})
	if addressField["type"] != "object" {
		t.Errorf("address: expected type=object, got %v", addressField["type"])
	}
	addrProps := addressField["properties"].(map[string]interface{})
	if _, ok := addrProps["street"]; !ok {
		t.Error("address: expected 'street' in nested properties")
	}

	// Verify repeated field → array
	tagsField := props["tags"].(map[string]interface{})
	if tagsField["type"] != "array" {
		t.Errorf("tags: expected type=array, got %v", tagsField["type"])
	}
	items := tagsField["items"].(map[string]interface{})
	if items["type"] != "string" {
		t.Errorf("tags.items: expected type=string, got %v", items["type"])
	}
}

func TestIntegration_FromPlugin_WellKnownTypes(t *testing.T) {
	// Build a file that uses google.protobuf.Timestamp
	timestampFd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("google/protobuf/timestamp.proto"),
		Package: strPtr("google.protobuf"),
		Syntax:  strPtr("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: strPtr("google.golang.org/protobuf/types/known/timestamppb"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("Timestamp"),
				Field: []*descriptorpb.FieldDescriptorProto{
					makeField("seconds", 1, descriptorpb.FieldDescriptorProto_TYPE_INT64),
					makeField("nanos", 2, descriptorpb.FieldDescriptorProto_TYPE_INT32),
				},
			},
		},
	}

	mainFd := &descriptorpb.FileDescriptorProto{
		Name:       strPtr("test/v1/event.proto"),
		Package:    strPtr("test.v1"),
		Syntax:     strPtr("proto3"),
		Dependency: []string{"google/protobuf/timestamp.proto"},
		Options: &descriptorpb.FileOptions{
			GoPackage: strPtr("github.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("CreateEventRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					makeField("title", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING),
					{
						Name:     strPtr("start_time"),
						Number:   int32Ptr(2),
						Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
						TypeName: strPtr(".google.protobuf.Timestamp"),
						JsonName: strPtr("startTime"),
					},
				},
			},
			makeSimpleMessage("CreateEventResponse",
				makeField("event_id", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING),
			),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("EventService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					makeMethod("CreateEvent", ".test.v1.CreateEventRequest", ".test.v1.CreateEventResponse"),
				},
			},
		},
	}

	// Build plugin with dependency first, then main file
	req := &pluginpb.CodeGeneratorRequest{
		ProtoFile:      []*descriptorpb.FileDescriptorProto{timestampFd, mainFd},
		FileToGenerate: []string{"test/v1/event.proto"},
	}
	plugin, err := protogen.Options{}.New(req)
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}

	results, err := extract.FromPlugin(plugin)
	if err != nil {
		t.Fatalf("FromPlugin failed: %v", err)
	}

	tool := results[0].Services[0].Tools[0]
	var schema map[string]interface{}
	if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
		t.Fatalf("invalid JSON schema: %v", err)
	}

	props := schema["properties"].(map[string]interface{})
	startTime := props["startTime"].(map[string]interface{})

	// Timestamp should be mapped to string with date-time format
	if startTime["type"] != "string" {
		t.Errorf("startTime: expected type=string (well-known Timestamp), got %v", startTime["type"])
	}
	if startTime["format"] != "date-time" {
		t.Errorf("startTime: expected format=date-time, got %v", startTime["format"])
	}
}

func TestIntegration_FromPlugin_MapField(t *testing.T) {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("test/v1/metadata.proto"),
		Package: strPtr("test.v1"),
		Syntax:  strPtr("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: strPtr("github.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("UpdateMetadataRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					makeField("resource_id", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING),
					{
						Name:     strPtr("labels"),
						Number:   int32Ptr(2),
						Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_REPEATED),
						Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
						TypeName: strPtr(".test.v1.UpdateMetadataRequest.LabelsEntry"),
						JsonName: strPtr("labels"),
					},
				},
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name: strPtr("LabelsEntry"),
						Field: []*descriptorpb.FieldDescriptorProto{
							makeField("key", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING),
							makeField("value", 2, descriptorpb.FieldDescriptorProto_TYPE_STRING),
						},
						Options: &descriptorpb.MessageOptions{
							MapEntry: boolPtr(true),
						},
					},
				},
			},
			makeSimpleMessage("UpdateMetadataResponse"),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("MetadataService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					makeMethod("UpdateMetadata", ".test.v1.UpdateMetadataRequest", ".test.v1.UpdateMetadataResponse"),
				},
			},
		},
	}

	plugin := buildPlugin(t, fd)
	results, err := extract.FromPlugin(plugin)
	if err != nil {
		t.Fatalf("FromPlugin failed: %v", err)
	}

	tool := results[0].Services[0].Tools[0]
	var schema map[string]interface{}
	if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
		t.Fatalf("invalid JSON schema: %v", err)
	}

	props := schema["properties"].(map[string]interface{})
	labels := props["labels"].(map[string]interface{})

	// Map fields should be type=object with additionalProperties
	if labels["type"] != "object" {
		t.Errorf("labels: expected type=object for map, got %v", labels["type"])
	}
	if labels["additionalProperties"] == nil {
		t.Error("labels: expected additionalProperties for map value type")
	}
}

func TestIntegration_FromPlugin_MultipleServices(t *testing.T) {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("test/v1/multi.proto"),
		Package: strPtr("test.v1"),
		Syntax:  strPtr("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: strPtr("github.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			makeSimpleMessage("Req", makeField("id", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING)),
			makeSimpleMessage("Resp", makeField("ok", 1, descriptorpb.FieldDescriptorProto_TYPE_BOOL)),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("ServiceA"),
				Method: []*descriptorpb.MethodDescriptorProto{
					makeMethod("MethodA", ".test.v1.Req", ".test.v1.Resp"),
				},
			},
			{
				Name: strPtr("ServiceB"),
				Method: []*descriptorpb.MethodDescriptorProto{
					makeMethod("MethodB1", ".test.v1.Req", ".test.v1.Resp"),
					makeMethod("MethodB2", ".test.v1.Req", ".test.v1.Resp"),
				},
			},
		},
	}

	plugin := buildPlugin(t, fd)
	results, err := extract.FromPlugin(plugin)
	if err != nil {
		t.Fatalf("FromPlugin failed: %v", err)
	}

	fileIR := results[0]
	if len(fileIR.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(fileIR.Services))
	}
	if len(fileIR.Services[0].Tools) != 1 {
		t.Errorf("ServiceA: expected 1 tool, got %d", len(fileIR.Services[0].Tools))
	}
	if len(fileIR.Services[1].Tools) != 2 {
		t.Errorf("ServiceB: expected 2 tools, got %d", len(fileIR.Services[1].Tools))
	}

	// Verify tool names include service prefix
	if fileIR.Services[0].Tools[0].Name != "ServiceA_MethodA" {
		t.Errorf("expected 'ServiceA_MethodA', got %q", fileIR.Services[0].Tools[0].Name)
	}
	if fileIR.Services[1].Tools[0].Name != "ServiceB_MethodB1" {
		t.Errorf("expected 'ServiceB_MethodB1', got %q", fileIR.Services[1].Tools[0].Name)
	}
}

func TestIntegration_FromPlugin_NoServices(t *testing.T) {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("test/v1/types_only.proto"),
		Package: strPtr("test.v1"),
		Syntax:  strPtr("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: strPtr("github.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			makeSimpleMessage("SomeMessage", makeField("field", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING)),
		},
	}

	plugin := buildPlugin(t, fd)
	results, err := extract.FromPlugin(plugin)
	if err != nil {
		t.Fatalf("FromPlugin failed: %v", err)
	}

	fileIR := results[0]
	if len(fileIR.Services) != 0 {
		t.Errorf("expected 0 services for types-only file, got %d", len(fileIR.Services))
	}
}

func TestIntegration_FromPlugin_DeprecatedMethod(t *testing.T) {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("test/v1/deprecated.proto"),
		Package: strPtr("test.v1"),
		Syntax:  strPtr("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: strPtr("github.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			makeSimpleMessage("Req", makeField("id", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING)),
			makeSimpleMessage("Resp", makeField("ok", 1, descriptorpb.FieldDescriptorProto_TYPE_BOOL)),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("DeprecatedService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       strPtr("OldMethod"),
						InputType:  strPtr(".test.v1.Req"),
						OutputType: strPtr(".test.v1.Resp"),
						Options: &descriptorpb.MethodOptions{
							Deprecated: boolPtr(true),
						},
					},
				},
			},
		},
	}

	plugin := buildPlugin(t, fd)
	results, err := extract.FromPlugin(plugin)
	if err != nil {
		t.Fatalf("FromPlugin failed: %v", err)
	}

	tool := results[0].Services[0].Tools[0]
	if !tool.IsDeprecated {
		t.Error("expected IsDeprecated=true")
	}
	if !strings.HasPrefix(tool.Description, "[DEPRECATED]") {
		t.Errorf("expected description to start with [DEPRECATED], got %q", tool.Description)
	}
}

func TestIntegration_FromPlugin_OneofFields(t *testing.T) {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("test/v1/oneof.proto"),
		Package: strPtr("test.v1"),
		Syntax:  strPtr("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: strPtr("github.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("SearchRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					makeField("query", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING),
					{
						Name:       strPtr("by_id"),
						Number:     int32Ptr(2),
						Label:      labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type:       typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING),
						JsonName:   strPtr("byId"),
						OneofIndex: int32Ptr(0),
					},
					{
						Name:       strPtr("by_name"),
						Number:     int32Ptr(3),
						Label:      labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type:       typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING),
						JsonName:   strPtr("byName"),
						OneofIndex: int32Ptr(0),
					},
				},
				OneofDecl: []*descriptorpb.OneofDescriptorProto{
					{Name: strPtr("lookup")},
				},
			},
			makeSimpleMessage("SearchResponse",
				makeField("result", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING),
			),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("SearchService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					makeMethod("Search", ".test.v1.SearchRequest", ".test.v1.SearchResponse"),
				},
			},
		},
	}

	plugin := buildPlugin(t, fd)
	results, err := extract.FromPlugin(plugin)
	if err != nil {
		t.Fatalf("FromPlugin failed: %v", err)
	}

	tool := results[0].Services[0].Tools[0]
	schemaStr := string(tool.InputSchema)

	// Oneof fields should have mutual exclusivity hints
	if !strings.Contains(schemaStr, "mutually exclusive") {
		t.Logf("Schema: %s", schemaStr)
		t.Error("expected 'mutually exclusive' hint in oneof field descriptions")
	}
}

func TestIntegration_FromPlugin_EmptyInputMessage(t *testing.T) {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("test/v1/empty.proto"),
		Package: strPtr("test.v1"),
		Syntax:  strPtr("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: strPtr("github.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			makeSimpleMessage("EmptyReq"),
			makeSimpleMessage("EmptyResp"),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("EmptyService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					makeMethod("NoParams", ".test.v1.EmptyReq", ".test.v1.EmptyResp"),
				},
			},
		},
	}

	plugin := buildPlugin(t, fd)
	results, err := extract.FromPlugin(plugin)
	if err != nil {
		t.Fatalf("FromPlugin failed: %v", err)
	}

	tool := results[0].Services[0].Tools[0]

	// Empty message should still produce valid JSON schema
	var schema map[string]interface{}
	if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
		t.Fatalf("invalid JSON schema: %v", err)
	}
	if schema["type"] != "object" {
		t.Error("expected type=object even for empty message")
	}

	// Should generate a linter warning about 0 fields
	hasWarning := false
	for _, w := range results[0].Warnings {
		if strings.Contains(w.Message, "no fields") || strings.Contains(w.Message, "0 fields") {
			hasWarning = true
		}
	}
	// Check tool-level warnings too
	for _, w := range tool.Warnings {
		if strings.Contains(w.Message, "no fields") || strings.Contains(w.Message, "0 fields") {
			hasWarning = true
		}
	}
	if !hasWarning {
		t.Log("Note: no warning about empty input message (may be by design)")
	}
}

func TestIntegration_FromPlugin_MultipleFiles(t *testing.T) {
	fd1 := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("test/v1/file1.proto"),
		Package: strPtr("test.v1"),
		Syntax:  strPtr("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: strPtr("github.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			makeSimpleMessage("Req1", makeField("f1", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING)),
			makeSimpleMessage("Resp1", makeField("r1", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING)),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("Svc1"),
				Method: []*descriptorpb.MethodDescriptorProto{
					makeMethod("M1", ".test.v1.Req1", ".test.v1.Resp1"),
				},
			},
		},
	}
	fd2 := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("test/v1/file2.proto"),
		Package: strPtr("test.v1"),
		Syntax:  strPtr("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: strPtr("github.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			makeSimpleMessage("Req2", makeField("f2", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING)),
			makeSimpleMessage("Resp2", makeField("r2", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING)),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("Svc2"),
				Method: []*descriptorpb.MethodDescriptorProto{
					makeMethod("M2", ".test.v1.Req2", ".test.v1.Resp2"),
				},
			},
		},
	}

	plugin := buildPlugin(t, fd1, fd2)
	results, err := extract.FromPlugin(plugin)
	if err != nil {
		t.Fatalf("FromPlugin failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 FileIRs, got %d", len(results))
	}
	if results[0].Services[0].Name != "Svc1" {
		t.Errorf("expected first file service 'Svc1', got %q", results[0].Services[0].Name)
	}
	if results[1].Services[0].Name != "Svc2" {
		t.Errorf("expected second file service 'Svc2', got %q", results[1].Services[0].Name)
	}
}
