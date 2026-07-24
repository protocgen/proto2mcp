package mcpruntime

import (
	"context"
	"encoding/json"
)

// MacroStepDef defines a step in a macro-tool execution.
type MacroStepDef struct {
	ToolName     string
	Parallel     bool
	OutputKey    string
	InputMapping map[string]string // field -> JSONPath expression
}

// FailureStrategy defines how macro-tool failures are handled.
type FailureStrategy int

const (
	// FailFast aborts execution immediately on the first error.
	FailFast FailureStrategy = iota
	// PartialResult continues execution and returns successful results alongside errors.
	PartialResult
	// Rollback attempts to undo previous steps on failure.
	Rollback
)

// MacroExecutor orchestrates the execution of macro-tool steps.
// V3: Implementations include SequentialExecutor, ParallelExecutor, TemporalExecutor.
type MacroExecutor interface {
	Execute(ctx context.Context, steps []MacroStepDef, input json.RawMessage, lookup func(string) (HandlerFunc, bool)) (*CallToolResult, error)
}
