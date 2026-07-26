package extract

import (
	"encoding/json"
	"testing"

	"google.golang.org/protobuf/types/descriptorpb"
)

func TestWarningLevelConstants(t *testing.T) {
	tests := []struct {
		name     string
		level    WarningLevel
		expected int
	}{
		{"WarnInfo", WarnInfo, 0},
		{"WarnWarning", WarnWarning, 1},
		{"WarnError", WarnError, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.level) != tt.expected {
				t.Errorf("expected %s to be %d, got %d", tt.name, tt.expected, tt.level)
			}
		})
	}
}

func TestWarningStruct(t *testing.T) {
	w := Warning{
		Severity: WarnError,
		Method:   "TestMethod",
		Message:  "TestMessage",
	}

	if w.Severity != WarnError {
		t.Errorf("expected Severity to be WarnError, got %v", w.Severity)
	}
	if w.Method != "TestMethod" {
		t.Errorf("expected Method to be TestMethod, got %s", w.Method)
	}
	if w.Message != "TestMessage" {
		t.Errorf("expected Message to be TestMessage, got %s", w.Message)
	}
}

func TestExtensionConstants(t *testing.T) {
	if extMethodOptions != 1179 {
		t.Errorf("expected extMethodOptions to be 1179, got %d", extMethodOptions)
	}
	if extServiceOptions != 1180 {
		t.Errorf("expected extServiceOptions to be 1180, got %d", extServiceOptions)
	}
	if extFileOptions != 1181 {
		t.Errorf("expected extFileOptions to be 1181, got %d", extFileOptions)
	}
}

func TestMacroTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		macro    MacroType
		expected int
	}{
		{"MacroTypeNone", MacroTypeNone, 0},
		{"MacroTypeSequential", MacroTypeSequential, 1},
		{"MacroTypeParallel", MacroTypeParallel, 2},
		{"MacroTypeTemporal", MacroTypeTemporal, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.macro) != tt.expected {
				t.Errorf("expected %s to be %d, got %d", tt.name, tt.expected, tt.macro)
			}
		})
	}
}

func TestFileIRBehavior(t *testing.T) {
	fir := FileIR{
		FileName: "test.proto",
		Skip:     true,
	}

	if fir.FileName != "test.proto" {
		t.Errorf("expected FileName test.proto, got %s", fir.FileName)
	}
	if !fir.Skip {
		t.Errorf("expected Skip to be true, got %v", fir.Skip)
	}
}

func TestServiceOptionsStruct(t *testing.T) {
	so := ServiceOptions{
		ToolNamePrefix: "prefix",
		Description:    "desc",
	}

	if so.ToolNamePrefix != "prefix" {
		t.Errorf("expected ToolNamePrefix prefix, got %s", so.ToolNamePrefix)
	}
	if so.Description != "desc" {
		t.Errorf("expected Description desc, got %s", so.Description)
	}
}

func TestToolIRStruct(t *testing.T) {
	schema := json.RawMessage(`{"type":"object"}`)
	tool := ToolIR{
		Name:           "toolName",
		MethodName:     "methodName",
		Description:    "desc",
		InputSchema:    schema,
		InputTypeName:  "input",
		OutputTypeName: "output",
		IsResource:     true,
		ResourceURI:    "uri",
		IsReadOnly:     true,
		IsDestructive:  false,
		IsDeprecated:   true,
		Version:        2,
		SubTools: []ToolRef{
			{ToolName: "subtool", Parallel: true, OutputKey: "out"},
		},
		MacroType: MacroTypeSequential,
		Warnings: []Warning{
			{Severity: WarnInfo, Message: "warn"},
		},
	}

	if tool.Name != "toolName" {
		t.Errorf("expected Name toolName, got %s", tool.Name)
	}
	if tool.InputTypeName != "input" {
		t.Errorf("expected InputTypeName input, got %s", tool.InputTypeName)
	}
	if len(tool.SubTools) != 1 {
		t.Fatalf("expected 1 subtool, got %d", len(tool.SubTools))
	}
	if tool.SubTools[0].ToolName != "subtool" {
		t.Errorf("expected subtool name subtool, got %s", tool.SubTools[0].ToolName)
	}
	if tool.MacroType != MacroTypeSequential {
		t.Errorf("expected MacroTypeSequential, got %v", tool.MacroType)
	}
}

func TestLintMethod(t *testing.T) {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("linter_test.proto"),
		Package: strPtr("testpkg"),
		Options: &descriptorpb.FileOptions{
			GoPackage: strPtr("github.com/test/testpkg"),
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("EmptyInput"),
			},
			{
				Name: strPtr("LargeInput"),
			},

			{
				Name: strPtr("DeepMessage1"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     strPtr("next"),
						Number:   func(i int32) *int32 { return &i }(1),
						Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
						TypeName: strPtr(".testpkg.DeepMessage2"),
					},
				},
			},
			{
				Name: strPtr("DeepMessage2"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     strPtr("next"),
						Number:   func(i int32) *int32 { return &i }(1),
						Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
						TypeName: strPtr(".testpkg.DeepMessage3"),
					},
				},
			},
			{
				Name: strPtr("DeepMessage3"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     strPtr("next"),
						Number:   func(i int32) *int32 { return &i }(1),
						Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
						TypeName: strPtr(".testpkg.DeepMessage4"),
					},
				},
			},
			{
				Name: strPtr("DeepMessage4"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     strPtr("next"),
						Number:   func(i int32) *int32 { return &i }(1),
						Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
						TypeName: strPtr(".testpkg.DeepMessage5"),
					},
				},
			},
			{
				Name: strPtr("DeepMessage5"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     strPtr("next"),
						Number:   func(i int32) *int32 { return &i }(1),
						Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
						TypeName: strPtr(".testpkg.EmptyInput"),
					},
				},
			},
			{
				Name: strPtr("NormalOutput"),
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("TestService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       strPtr("BidiStream"),
						InputType:  strPtr(".testpkg.EmptyInput"),
						OutputType: strPtr(".testpkg.EmptyInput"),
						ClientStreaming: func(b bool) *bool { return &b }(true),
						ServerStreaming: func(b bool) *bool { return &b }(true),
					},
					{
						Name:       strPtr("ClientStream"),
						InputType:  strPtr(".testpkg.EmptyInput"),
						OutputType: strPtr(".testpkg.EmptyInput"),
						ClientStreaming: func(b bool) *bool { return &b }(true),
					},
					{
						Name:       strPtr("ServerStream"),
						InputType:  strPtr(".testpkg.EmptyInput"),
						OutputType: strPtr(".testpkg.EmptyInput"),
						ServerStreaming: func(b bool) *bool { return &b }(true),
					},
					{
						Name:       strPtr("NoDescription"),
						InputType:  strPtr(".testpkg.EmptyInput"),
						OutputType: strPtr(".testpkg.EmptyInput"),
					},
					{
						Name:       strPtr("LargeSchema"),
						InputType:  strPtr(".testpkg.LargeInput"),
						OutputType: strPtr(".testpkg.EmptyInput"),
					},

					{
						Name:       strPtr("DeeplyNested"),
						InputType:  strPtr(".testpkg.DeepMessage1"),
						OutputType: strPtr(".testpkg.EmptyInput"),
					},
				},
			},
		},
	}

	for i := 1; i <= 21; i++ {
		fd.MessageType[1].Field = append(fd.MessageType[1].Field, &descriptorpb.FieldDescriptorProto{
			Name:   strPtr(func(i int) string { return "f" + string(rune('a'+i)) }(i)),
			Number: func(i int32) *int32 { return &i }(int32(i)),
			Label:  labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
			Type:   typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING),
		})
	}

	plugin := buildTestPlugin(t, fd)
	file := plugin.FilesByPath["linter_test.proto"]
	service := file.Services[0]

	t.Run("BidiStream", func(t *testing.T) {
		warns := LintMethod(service.Methods[0])
		if len(warns) == 0 || warns[0].Message != "MCP does not support bidirectional streaming" {
			t.Error("expected bidi stream warning")
		}
	})

	t.Run("ClientStream", func(t *testing.T) {
		warns := LintMethod(service.Methods[1])
		if len(warns) == 0 || warns[0].Message != "MCP does not support client streaming" {
			t.Error("expected client stream warning")
		}
	})

	t.Run("ServerStream", func(t *testing.T) {
		warns := LintMethod(service.Methods[2])
		if len(warns) == 0 || warns[0].Message != "MCP does not support server streaming" {
			t.Error("expected server stream warning")
		}
	})

	t.Run("NoDescription", func(t *testing.T) {
		warns := LintMethod(service.Methods[3])
		found := false
		for _, w := range warns {
			if w.Message == "Method has no description; LLM will not understand its purpose" {
				found = true
			}
		}
		if !found {
			t.Error("expected no description warning")
		}
	})
	
	t.Run("EmptyInputFields", func(t *testing.T) {
		warns := LintMethod(service.Methods[3])
		found := false
		for _, w := range warns {
			if w.Message == "Input message has no fields; tool has no parameters" {
				found = true
			}
		}
		if !found {
			t.Error("expected empty input warning")
		}
	})

	t.Run("LargeSchema", func(t *testing.T) {
		warns := LintMethod(service.Methods[4])
		found := false
		for _, w := range warns {
			if w.Message == "Input message has 21 fields; large schemas may confuse LLMs" {
				found = true
			}
		}
		if !found {
			t.Error("expected large schema warning")
		}
	})


	t.Run("DeeplyNested", func(t *testing.T) {
		warns := LintMethod(service.Methods[5])
		found := false
		for _, w := range warns {
			if w.Message == "Message nesting depth 5 exceeds recommended maximum of 4" {
				found = true
			}
		}
		if !found {
			t.Error("expected nesting depth warning")
		}
	})
}
