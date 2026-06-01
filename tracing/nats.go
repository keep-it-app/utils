package tracing

import (
	"context"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// tracerName is the instrumentation scope used for manually-started spans.
const tracerName = "github.com/keep-it-app/utils/tracing"

// natsHeaderCarrier adapts nats.Header to a propagation.TextMapCarrier so the
// global propagator can inject/extract W3C trace context into NATS messages.
type natsHeaderCarrier nats.Header

func (c natsHeaderCarrier) Get(key string) string {
	if v := nats.Header(c).Values(key); len(v) > 0 {
		return v[0]
	}
	return ""
}

func (c natsHeaderCarrier) Set(key, value string) {
	nats.Header(c).Set(key, value)
}

func (c natsHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// InjectNATS writes the trace context from ctx into msg.Header, allocating the
// header map if needed.
func InjectNATS(ctx context.Context, msg *nats.Msg) {
	if msg.Header == nil {
		msg.Header = nats.Header{}
	}
	otel.GetTextMapPropagator().Inject(ctx, natsHeaderCarrier(msg.Header))
}

// ExtractNATS returns a context carrying the upstream trace context read from
// msg.Header. If the message has no header, ctx is returned unchanged.
func ExtractNATS(ctx context.Context, msg *nats.Msg) context.Context {
	if msg.Header == nil {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, natsHeaderCarrier(msg.Header))
}

// StartProducerSpan starts a producer-kind span and injects the resulting
// trace context into msg.Header. End the returned span after publishing.
func StartProducerSpan(
	ctx context.Context, msg *nats.Msg, name string,
) (context.Context, trace.Span) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, name,
		trace.WithSpanKind(trace.SpanKindProducer))
	InjectNATS(ctx, msg)
	return ctx, span
}

// StartConsumerSpan extracts the upstream trace context from msg.Header and
// starts a consumer-kind span as its child. End the returned span after the
// message is handled.
func StartConsumerSpan(
	ctx context.Context, msg *nats.Msg, name string,
) (context.Context, trace.Span) {
	ctx = ExtractNATS(ctx, msg)
	return otel.Tracer(tracerName).Start(ctx, name,
		trace.WithSpanKind(trace.SpanKindConsumer))
}
