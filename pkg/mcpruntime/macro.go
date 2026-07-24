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
	InputMapping map[string]string // field → JSONPath expression
}

// FailureStrategy defines how macro-tool failures are handled.
type FailureStrategy int

const (
	// FailureStrategyUnspecified is the zero value (proto: FAILURE_STRATEGY_UNSPECIFIED = 0).
	FailureStrategyUnspecified FailureStrategy = iota
	// FailFast aborts execution immediately on the first error (proto: FAILURE_STRATEGY_FAIL_FAST = 1).
	FailFast
	// PartialResult continues execution and returns successful results alongside errors (proto: FAILURE_STRATEGY_PARTIAL_RESULT = 2).
	PartialResult
	// Rollback attempts to undo previous steps on failure.
	//
	// V3: For production saga/rollback patterns, use Temporal workflows
	// rather than implementing rollback logic in the in-process executor.
	// The in-process MacroExecutor should only handle FailFast and
	// PartialResult strategies. Temporal provides durable execution,
	// visibility, and battle-tested compensation patterns.
	// (proto: FAILURE_STRATEGY_ROLLBACK = 3)
	Rollback
)

// MacroExecutor orchestrates the execution of macro-tool steps.
//
// This is a V3 seam. No implementations are provided in V1.
//
// DESIGN NOTE (from DE review): For platforms with an existing workflow
// engine (e.g., Temporal, Conductor), the primary V3 implementation
// should delegate to that engine rather than reimplementing orchestration.
// The Temporal-backed executor would:
//
//  1. Map each MacroStepDef to a Temporal Activity
//  2. Use Temporal's native saga/compensation for Rollback strategy
//  3. Use Temporal's parallel execution for steps with Parallel=true
//  4. Inherit Temporal's retry policies, timeouts, and audit trail
//
// A lightweight in-process executor (sequential/parallel) may be provided
// for simple compositions that don't warrant a workflow engine, but it
// should NOT attempt saga/rollback semantics — use Temporal for that.
type MacroExecutor interface {
	Execute(ctx context.Context, steps []MacroStepDef, input json.RawMessage, lookup func(string) (HandlerFunc, bool)) (*CallToolResult, error)
}
