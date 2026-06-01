package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// StartSpan starts a span on the shared tracer scope. It is a convenience for
// instrumenting code that has no transport-level auto-instrumentation
// (external API clients, background handlers).
func StartSpan(
	ctx context.Context, name string, opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, name, opts...)
}

// RecordError marks the span as failed and records err. It is a no-op when err
// is nil, so it can wrap the tail of a function unconditionally.
func RecordError(span trace.Span, err error) {
	if err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
