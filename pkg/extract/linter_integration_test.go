package extract_test

import (
	"strings"
	"testing"

	"github.com/protocgen/proto2mcp/pkg/extract"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// ============================================================================
// Integration Tests: Linter Warning Propagation
// ============================================================================

func TestLintIntegration_NoDescription(t *testing.T) {
	// Methods without comments should produce WarnWarning
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("test/v1/nodesc.proto"),
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
				Name: strPtr("NoDescService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					makeMethod("DoThing", ".test.v1.Req", ".test.v1.Resp"),
				},
			},
		},
	}

	plugin := buildPlugin(t, fd)
	file := plugin.FilesByPath["test/v1/nodesc.proto"]
	method := file.Services[0].Methods[0]

	warnings := extract.LintMethod(method)

	foundNoDesc := false
	for _, w := range warnings {
		if w.Severity == extract.WarnWarning && strings.Contains(w.Message, "no description") {
			foundNoDesc = true
		}
	}
	if !foundNoDesc {
		t.Error("expected WarnWarning about missing description")
	}
}

func TestLintIntegration_EmptyInput(t *testing.T) {
	// Methods with 0-field input message should produce WarnWarning
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("test/v1/emptyinput.proto"),
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
				Name: strPtr("EmptySvc"),
				Method: []*descriptorpb.MethodDescriptorProto{
					makeMethod("NoArgs", ".test.v1.EmptyReq", ".test.v1.EmptyResp"),
				},
			},
		},
	}

	plugin := buildPlugin(t, fd)
	method := plugin.FilesByPath["test/v1/emptyinput.proto"].Services[0].Methods[0]

	warnings := extract.LintMethod(method)

	foundEmpty := false
	for _, w := range warnings {
		if w.Severity == extract.WarnWarning && strings.Contains(w.Message, "no fields") {
			foundEmpty = true
		}
	}
	if !foundEmpty {
		t.Error("expected WarnWarning about empty input message")
	}
}

func TestLintIntegration_LargeInput(t *testing.T) {
	// Methods with >20 fields should produce WarnWarning
	fields := make([]*descriptorpb.FieldDescriptorProto, 25)
	for i := range fields {
		num := int32(i + 1)
		fields[i] = &descriptorpb.FieldDescriptorProto{
			Name:     strPtr("field_" + string(rune('a'+i))),
			Number:   &num,
			Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
			Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING),
			JsonName: strPtr("field" + string(rune('A'+i))),
		}
	}

	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("test/v1/large.proto"),
		Package: strPtr("test.v1"),
		Syntax:  strPtr("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: strPtr("github.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: strPtr("LargeReq"), Field: fields},
			makeSimpleMessage("LargeResp", makeField("ok", 1, descriptorpb.FieldDescriptorProto_TYPE_BOOL)),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("LargeSvc"),
				Method: []*descriptorpb.MethodDescriptorProto{
					makeMethod("BigMethod", ".test.v1.LargeReq", ".test.v1.LargeResp"),
				},
			},
		},
	}

	plugin := buildPlugin(t, fd)
	method := plugin.FilesByPath["test/v1/large.proto"].Services[0].Methods[0]

	warnings := extract.LintMethod(method)

	foundLarge := false
	for _, w := range warnings {
		if w.Severity == extract.WarnWarning && strings.Contains(w.Message, "large schemas") {
			foundLarge = true
		}
	}
	if !foundLarge {
		t.Error("expected WarnWarning about large schema (>20 fields)")
	}
}

func TestLintIntegration_StreamingMethods(t *testing.T) {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("test/v1/streaming.proto"),
		Package: strPtr("test.v1"),
		Syntax:  strPtr("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: strPtr("github.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			makeSimpleMessage("Req", makeField("data", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING)),
			makeSimpleMessage("Resp", makeField("result", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING)),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("StreamSvc"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:            strPtr("ServerStream"),
						InputType:       strPtr(".test.v1.Req"),
						OutputType:      strPtr(".test.v1.Resp"),
						ServerStreaming:  boolPtr(true),
					},
					{
						Name:            strPtr("ClientStream"),
						InputType:       strPtr(".test.v1.Req"),
						OutputType:      strPtr(".test.v1.Resp"),
						ClientStreaming:  boolPtr(true),
					},
					{
						Name:            strPtr("BidiStream"),
						InputType:       strPtr(".test.v1.Req"),
						OutputType:      strPtr(".test.v1.Resp"),
						ClientStreaming:  boolPtr(true),
						ServerStreaming:  boolPtr(true),
					},
				},
			},
		},
	}

	plugin := buildPlugin(t, fd)
	svc := plugin.FilesByPath["test/v1/streaming.proto"].Services[0]

	tests := []struct {
		name    string
		method  *protogen.Method
		wantMsg string
	}{
		{"ServerStream", svc.Methods[0], "server streaming"},
		{"ClientStream", svc.Methods[1], "client streaming"},
		{"BidiStream", svc.Methods[2], "bidirectional streaming"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := extract.LintMethod(tt.method)
			foundError := false
			for _, w := range warnings {
				if w.Severity == extract.WarnError && strings.Contains(w.Message, "streaming") {
					foundError = true
				}
			}
			if !foundError {
				t.Errorf("expected WarnError about streaming for %s", tt.name)
			}
		})
	}
}

func TestLintIntegration_FieldsWithoutComments(t *testing.T) {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("test/v1/uncommented.proto"),
		Package: strPtr("test.v1"),
		Syntax:  strPtr("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: strPtr("github.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			makeSimpleMessage("UncommentedReq",
				makeField("name", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING),
				makeField("age", 2, descriptorpb.FieldDescriptorProto_TYPE_INT32),
			),
			makeSimpleMessage("UncommentedResp", makeField("ok", 1, descriptorpb.FieldDescriptorProto_TYPE_BOOL)),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("UncommentedSvc"),
				Method: []*descriptorpb.MethodDescriptorProto{
					makeMethod("DoIt", ".test.v1.UncommentedReq", ".test.v1.UncommentedResp"),
				},
			},
		},
	}

	plugin := buildPlugin(t, fd)
	method := plugin.FilesByPath["test/v1/uncommented.proto"].Services[0].Methods[0]

	warnings := extract.LintMethod(method)

	// Fields without comments should produce WarnInfo
	infoCount := 0
	for _, w := range warnings {
		if w.Severity == extract.WarnInfo && strings.Contains(w.Message, "no description") {
			infoCount++
		}
	}
	// We have 2 fields on input + 1 on output without comments = at least 2 from input
	if infoCount < 2 {
		t.Errorf("expected at least 2 WarnInfo about uncommented fields, got %d", infoCount)
	}
}

func TestLintIntegration_DeepNesting(t *testing.T) {
	// Create messages that nest 6 levels deep
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("test/v1/deep.proto"),
		Package: strPtr("test.v1"),
		Syntax:  strPtr("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: strPtr("github.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			makeSimpleMessage("Level5", makeField("value", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING)),
			{
				Name: strPtr("Level4"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("child"), Number: int32Ptr(1), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE), TypeName: strPtr(".test.v1.Level5"), JsonName: strPtr("child")},
				},
			},
			{
				Name: strPtr("Level3"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("child"), Number: int32Ptr(1), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE), TypeName: strPtr(".test.v1.Level4"), JsonName: strPtr("child")},
				},
			},
			{
				Name: strPtr("Level2"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("child"), Number: int32Ptr(1), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE), TypeName: strPtr(".test.v1.Level3"), JsonName: strPtr("child")},
				},
			},
			{
				Name: strPtr("Level1"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("child"), Number: int32Ptr(1), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE), TypeName: strPtr(".test.v1.Level2"), JsonName: strPtr("child")},
				},
			},
			{
				Name: strPtr("DeepReq"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("root"), Number: int32Ptr(1), Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE), TypeName: strPtr(".test.v1.Level1"), JsonName: strPtr("root")},
				},
			},
			makeSimpleMessage("DeepResp", makeField("ok", 1, descriptorpb.FieldDescriptorProto_TYPE_BOOL)),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("DeepSvc"),
				Method: []*descriptorpb.MethodDescriptorProto{
					makeMethod("DeepCall", ".test.v1.DeepReq", ".test.v1.DeepResp"),
				},
			},
		},
	}

	plugin := buildPlugin(t, fd)
	method := plugin.FilesByPath["test/v1/deep.proto"].Services[0].Methods[0]

	warnings := extract.LintMethod(method)

	foundDepth := false
	for _, w := range warnings {
		if w.Severity == extract.WarnWarning && strings.Contains(w.Message, "nesting depth") {
			foundDepth = true
		}
	}
	if !foundDepth {
		t.Error("expected WarnWarning about nesting depth exceeding maximum")
	}
}

func TestLintIntegration_GoogleProtobufAny(t *testing.T) {
	// google.protobuf.Any should produce WarnError
	anyFd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("google/protobuf/any.proto"),
		Package: strPtr("google.protobuf"),
		Syntax:  strPtr("proto3"),
		Options: &descriptorpb.FileOptions{
			GoPackage: strPtr("google.golang.org/protobuf/types/known/anypb"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("Any"),
				Field: []*descriptorpb.FieldDescriptorProto{
					makeField("type_url", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING),
					makeField("value", 2, descriptorpb.FieldDescriptorProto_TYPE_BYTES),
				},
			},
		},
	}

	mainFd := &descriptorpb.FileDescriptorProto{
		Name:       strPtr("test/v1/anyfield.proto"),
		Package:    strPtr("test.v1"),
		Syntax:     strPtr("proto3"),
		Dependency: []string{"google/protobuf/any.proto"},
		Options: &descriptorpb.FileOptions{
			GoPackage: strPtr("github.com/test/v1;testv1"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("AnyReq"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name: strPtr("payload"), Number: int32Ptr(1),
						Label: labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
						TypeName: strPtr(".google.protobuf.Any"), JsonName: strPtr("payload"),
					},
				},
			},
			makeSimpleMessage("AnyResp", makeField("ok", 1, descriptorpb.FieldDescriptorProto_TYPE_BOOL)),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("AnySvc"),
				Method: []*descriptorpb.MethodDescriptorProto{
					makeMethod("WithAny", ".test.v1.AnyReq", ".test.v1.AnyResp"),
				},
			},
		},
	}

	req := &pluginpb.CodeGeneratorRequest{
		ProtoFile:      []*descriptorpb.FileDescriptorProto{anyFd, mainFd},
		FileToGenerate: []string{"test/v1/anyfield.proto"},
	}
	plugin, err := protogen.Options{}.New(req)
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}

	method := plugin.FilesByPath["test/v1/anyfield.proto"].Services[0].Methods[0]
	warnings := extract.LintMethod(method)

	foundAny := false
	for _, w := range warnings {
		if w.Severity == extract.WarnError && strings.Contains(w.Message, "Any") {
			foundAny = true
		}
	}
	if !foundAny {
		t.Error("expected WarnError about google.protobuf.Any")
	}
}
