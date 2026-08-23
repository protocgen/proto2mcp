package mcpruntime

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const unknownToolSentinel = "__unknown__"
const maxTenantLen = 128

// Metrics holds the OTel instruments for MCP tool call metrics.
type Metrics struct {
	toolCallsTotal   metric.Int64Counter
	toolCallDuration metric.Float64Histogram
	validTools       map[string]bool // bounded set of known tool names
}

// NewMetrics creates OTel metric instruments with unbounded cardinality.
// For production use, prefer NewBoundedMetrics to prevent cardinality explosion.
func NewMetrics(meter metric.Meter) (*Metrics, error) {
	return NewBoundedMetrics(meter, nil)
}

// NewBoundedMetrics creates OTel metric instruments with bounded cardinality.
// validTools constrains the tool label; unknown tools are recorded as a fixed
// sentinel value. If validTools is nil, all tool names are accepted.
func NewBoundedMetrics(meter metric.Meter, validTools []string) (*Metrics, error) {
	var toolSet map[string]bool
	if validTools != nil {
		toolSet = make(map[string]bool, len(validTools))
		for _, t := range validTools {
			if t == unknownToolSentinel {
				continue // reserved sentinel — skip
			}
			toolSet[t] = true
		}
	}

	total, err := meter.Int64Counter("mcp_tool_calls_total",
		metric.WithDescription("Total number of MCP tool calls"),
	)
	if err != nil {
		return nil, fmt.Errorf("creating mcp_tool_calls_total counter: %w", err)
	}

	duration, err := meter.Float64Histogram("mcp_tool_call_duration_seconds",
		metric.WithDescription("Duration of MCP tool calls in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("creating mcp_tool_call_duration_seconds histogram: %w", err)
	}

	return &Metrics{
		toolCallsTotal:   total,
		toolCallDuration: duration,
		validTools:       toolSet,
	}, nil
}

func truncateUTF8(s string, maxBytes int) string {
	// Strip any invalid UTF-8 sequences first (e.g., from untrusted input).
	s = strings.ToValidUTF8(s, "")
	if len(s) <= maxBytes {
		return s
	}
	// Truncate and back up to the last complete rune boundary.
	s = s[:maxBytes]
	// Since the input is now valid UTF-8, the only issue is a partial
	// rune at the end from the byte-level truncation.
	for i := len(s) - 1; i >= 0; i-- {
		if utf8.RuneStart(s[i]) {
			// Found the start of the last rune — check if it's complete.
			r, size := utf8.DecodeRuneInString(s[i:])
			if r == utf8.RuneError && size == 1 {
				// Incomplete rune — truncate before it.
				return s[:i]
			}
			return s
		}
	}
	return ""
}

// RecordToolCall records a tool call metric.
func (m *Metrics) RecordToolCall(ctx context.Context, toolName, tenantID, status string, duration time.Duration) {
	// Bound cardinality: unknown tools get a fixed label
	safeTool := toolName
	if m.validTools != nil && !m.validTools[toolName] {
		safeTool = unknownToolSentinel
	}

	// Cap tenantID length to prevent cardinality via long/unique IDs
	safeTenant := truncateUTF8(tenantID, maxTenantLen)

	attrs := []attribute.KeyValue{
		attribute.String("tool", safeTool),
		attribute.String("tenant", safeTenant),
	}

	m.toolCallDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	// Add status for the total counter
	attrs = append(attrs, attribute.String("status", status))
	m.toolCallsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
}
