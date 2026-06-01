package tracing

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestInitDisabled(t *testing.T) {
	shutdown, err := Init(context.Background(), Config{Enabled: false})
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	if shutdown == nil {
		t.Fatal("shutdown func is nil")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown returned error: %v", err)
	}

	// A disabled provider produces non-recording spans.
	_, span := otel.Tracer("test").Start(context.Background(), "op")
	if span.IsRecording() {
		t.Error("span should not be recording when tracing is disabled")
	}
	span.End()

	// Propagator must still be installed so context flows downstream.
	carrier := propagation.MapCarrier{
		"traceparent": "00-0102030405060708090a0b0c0d0e0f10-0102030405060708-01",
	}
	ctx := otel.GetTextMapPropagator().Extract(context.Background(), carrier)
	if !trace.SpanContextFromContext(ctx).IsValid() {
		t.Error("propagator did not extract a valid context when disabled")
	}
}

func TestSamplerClamp(t *testing.T) {
	tests := []struct {
		name  string
		ratio float64
		want  string
	}{
		{"negative -> never", -1, "ParentBased{root:AlwaysOffSampler"},
		{"zero -> never", 0, "ParentBased{root:AlwaysOffSampler"},
		{"one -> always", 1, "ParentBased{root:AlwaysOnSampler"},
		{"above one -> always", 2, "ParentBased{root:AlwaysOnSampler"},
		{"half -> ratio", 0.5, "ParentBased{root:TraceIDRatioBased"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sampler(tt.ratio).Description()
			if len(got) < len(tt.want) || got[:len(tt.want)] != tt.want {
				t.Errorf("sampler(%v).Description() = %q, want prefix %q",
					tt.ratio, got, tt.want)
			}
		})
	}
}
