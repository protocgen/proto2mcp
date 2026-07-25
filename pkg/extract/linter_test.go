package extract

import (
	"encoding/json"
	"testing"
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
