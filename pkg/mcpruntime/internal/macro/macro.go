// Package macro contains V3 macro-tool types that are not yet part of
// the public API. These will be promoted to mcpruntime when V3 is ready.
package macro

import (
	"context"
	"encoding/json"
	"fmt"
)

// HandlerFunc mirrors mcpruntime.HandlerFunc to avoid circular imports.
// When promoted to mcpruntime, this will be removed in favor of the real type.
type HandlerFunc func(ctx context.Context, req json.RawMessage) (json.RawMessage, error)

// StepDef defines a step in a macro-tool execution.
type StepDef struct {
	ToolName  string
	Parallel  bool
	OutputKey string
	// InputMapping maps input field names to JSONPath expressions for
	// wiring output from previous steps into the current step's input.
	InputMapping map[string]string
}

// FailureStrategy defines how macro-tool failures are handled.
type FailureStrategy int

const (
	// FailureStrategyUnspecified is the zero value.
	FailureStrategyUnspecified FailureStrategy = iota
	// FailFast aborts execution immediately on the first error.
	FailFast
	// PartialResult continues execution and returns successful results alongside errors.
	PartialResult
	// Rollback attempts to undo previous steps on failure.
	//
	// V3: For production saga/rollback patterns, use Temporal workflows
	// rather than implementing rollback logic in the in-process executor.
	Rollback
)

// Executor orchestrates the execution of macro-tool steps.
//
// This is a V3 seam. No implementations are provided in V1.
//
// DESIGN NOTE: For platforms with an existing workflow engine
// (e.g., Temporal, Conductor), the primary V3 implementation
// should delegate to that engine rather than reimplementing orchestration.
type Executor interface {
	Execute(ctx context.Context, steps []StepDef, input json.RawMessage) (json.RawMessage, error)
}

// SequentialExecutor runs macro steps one at a time, in order.
// This is the default V3 implementation for in-process orchestration.
//
// EXPERIMENTAL: This API is not yet stable.
type SequentialExecutor struct {
	// Lookup resolves a tool handler by name.
	Lookup func(toolName string) (HandlerFunc, bool)
}

// Execute runs each step sequentially, passing the original input
// to each step. Results are collected into a JSON object keyed by OutputKey.
// If a step fails, execution stops and the error is returned (fail-fast).
func (e *SequentialExecutor) Execute(ctx context.Context, steps []StepDef, input json.RawMessage) (json.RawMessage, error) {
	results := make(map[string]json.RawMessage)

	for i, step := range steps {
		handler, ok := e.Lookup(step.ToolName)
		if !ok {
			return nil, fmt.Errorf("macro step %d: tool %q not found", i, step.ToolName)
		}

		result, err := handler(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("macro step %d (%s): %w", i, step.ToolName, err)
		}

		key := step.OutputKey
		if key == "" {
			key = step.ToolName
		}
		results[key] = result
	}

	return json.Marshal(results)
}
