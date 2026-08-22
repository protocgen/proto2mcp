package macro

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestSequentialExecutor_HappyPath(t *testing.T) {
	e := &SequentialExecutor{
		Lookup: func(name string) (HandlerFunc, bool) {
			return func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
				return json.RawMessage(`{"tool":"` + name + `"}`), nil
			}, true
		},
	}

	steps := []StepDef{
		{ToolName: "step1", OutputKey: "first"},
		{ToolName: "step2", OutputKey: "second"},
	}

	result, err := e.Execute(context.Background(), steps, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	var out map[string]json.RawMessage
	if err := json.Unmarshal(result, &out); err != nil {
		t.Fatalf("Unmarshal result: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 results, got %d", len(out))
	}
	if _, ok := out["first"]; !ok {
		t.Error("missing 'first' key")
	}
	if _, ok := out["second"]; !ok {
		t.Error("missing 'second' key")
	}
}

func TestSequentialExecutor_ToolNotFound(t *testing.T) {
	e := &SequentialExecutor{
		Lookup: func(name string) (HandlerFunc, bool) {
			return nil, false
		},
	}

	_, err := e.Execute(context.Background(), []StepDef{{ToolName: "missing"}}, json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error for missing tool")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestSequentialExecutor_FailFast(t *testing.T) {
	callCount := 0
	e := &SequentialExecutor{
		Lookup: func(name string) (HandlerFunc, bool) {
			return func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
				callCount++
				if name == "fail" {
					return nil, fmt.Errorf("intentional failure")
				}
				return json.RawMessage(`{}`), nil
			}, true
		},
	}

	steps := []StepDef{
		{ToolName: "ok"},
		{ToolName: "fail"},
		{ToolName: "never"},
	}

	_, err := e.Execute(context.Background(), steps, json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls (fail-fast), got %d", callCount)
	}
}

func TestSequentialExecutor_DefaultOutputKey(t *testing.T) {
	e := &SequentialExecutor{
		Lookup: func(name string) (HandlerFunc, bool) {
			return func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
				return json.RawMessage(`{}`), nil
			}, true
		},
	}

	steps := []StepDef{
		{ToolName: "MyTool"}, // no OutputKey — should use ToolName
	}

	result, err := e.Execute(context.Background(), steps, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	var out map[string]json.RawMessage
	if err := json.Unmarshal(result, &out); err != nil {
		t.Fatalf("Unmarshal result: %v", err)
	}
	if _, ok := out["MyTool"]; !ok {
		t.Error("expected ToolName as default key")
	}
}

func TestSequentialExecutor_EmptySteps(t *testing.T) {
	e := &SequentialExecutor{
		Lookup: func(name string) (HandlerFunc, bool) {
			return nil, false
		},
	}

	result, err := e.Execute(context.Background(), nil, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Execute with nil steps should succeed: %v", err)
	}
	if string(result) != "{}" {
		t.Errorf("expected empty object, got %s", result)
	}
}
