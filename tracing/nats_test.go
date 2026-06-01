package tracing

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	// Tests rely on the W3C propagator being installed.
	otel.SetTextMapPropagator(propagation.TraceContext{})
}

func TestNATSHeaderCarrier(t *testing.T) {
	t.Run("set then get", func(t *testing.T) {
		c := natsHeaderCarrier(nats.Header{})
		c.Set("traceparent", "value")
		if got := c.Get("traceparent"); got != "value" {
			t.Fatalf("Get = %q, want %q", got, "value")
		}
	})

	t.Run("get missing returns empty", func(t *testing.T) {
		c := natsHeaderCarrier(nats.Header{})
		if got := c.Get("nope"); got != "" {
			t.Fatalf("Get = %q, want empty", got)
		}
	})

	t.Run("keys", func(t *testing.T) {
		c := natsHeaderCarrier(nats.Header{})
		c.Set("a", "1")
		c.Set("b", "2")
		if got := len(c.Keys()); got != 2 {
			t.Fatalf("len(Keys) = %d, want 2", got)
		}
	})
}

func TestInjectExtractRoundTrip(t *testing.T) {
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10},
		SpanID:     trace.SpanID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	msg := &nats.Msg{Subject: "new-note"}
	InjectNATS(ctx, msg)

	if msg.Header == nil || msg.Header.Get("traceparent") == "" {
		t.Fatal("InjectNATS did not write traceparent header")
	}

	got := trace.SpanContextFromContext(ExtractNATS(context.Background(), msg))
	if got.TraceID() != sc.TraceID() {
		t.Errorf("TraceID = %v, want %v", got.TraceID(), sc.TraceID())
	}
	if got.SpanID() != sc.SpanID() {
		t.Errorf("SpanID = %v, want %v", got.SpanID(), sc.SpanID())
	}
	if !got.IsRemote() {
		t.Error("extracted span context should be remote")
	}
}

func TestExtractNATSNilHeader(t *testing.T) {
	ctx := context.Background()
	if ExtractNATS(ctx, &nats.Msg{}) != ctx {
		t.Fatal("ExtractNATS with nil header should return ctx unchanged")
	}
}
