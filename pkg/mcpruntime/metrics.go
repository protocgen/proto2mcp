package mcpruntime

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics holds the OTel instruments for MCP tool call metrics.
type Metrics struct {
	toolCallsTotal   metric.Int64Counter
	toolCallDuration metric.Float64Histogram
}

// NewMetrics creates OTel metric instruments.
// Emits: mcp_tool_calls_total{tool, tenant, status}
//        mcp_tool_call_duration_seconds{tool, tenant}
func NewMetrics(meter metric.Meter) (*Metrics, error) {
	total, err := meter.Int64Counter("mcp_tool_calls_total",
		metric.WithDescription("Total number of MCP tool calls"),
	)
	if err != nil {
		return nil, err
	}

	duration, err := meter.Float64Histogram("mcp_tool_call_duration_seconds",
		metric.WithDescription("Duration of MCP tool calls in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		toolCallsTotal:   total,
		toolCallDuration: duration,
	}, nil
}

// RecordToolCall records a tool call metric.
func (m *Metrics) RecordToolCall(ctx context.Context, toolName, tenantID, status string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("tool", toolName),
		attribute.String("tenant", tenantID),
	}

	m.toolCallDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	// Add status for the total counter
	attrs = append(attrs, attribute.String("status", status))
	m.toolCallsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
}
