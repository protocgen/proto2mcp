package mcpruntime

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/metric/noop"
)

func TestMetrics_NewMetrics(t *testing.T) {
	provider := noop.NewMeterProvider()
	meter := provider.Meter("test")

	metrics, err := NewMetrics(meter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if metrics == nil {
		t.Fatal("expected non-nil metrics")
	}

	// Verify no panics on RecordToolCall
	metrics.RecordToolCall(context.Background(), "my-tool", "tenant-123", "success", 100*time.Millisecond)
}

func TestMetrics_RecordToolCall_MultipleCalls(t *testing.T) {
	provider := noop.NewMeterProvider()
	meter := provider.Meter("test")
	metrics, _ := NewMetrics(meter)

	// Multiple calls to ensure it doesn't panic
	metrics.RecordToolCall(context.Background(), "tool1", "tenant1", "success", 10*time.Millisecond)
	metrics.RecordToolCall(context.Background(), "tool2", "tenant2", "error", 20*time.Millisecond)
	metrics.RecordToolCall(context.Background(), "tool1", "", "success", 5*time.Millisecond)
}
