// Package tracing wires OpenTelemetry tracing for Keep It services.
//
// Init sets up an OTLP/gRPC exporter, a TracerProvider with service resource
// attributes, a W3C TraceContext+Baggage propagator and a parent-based ratio
// sampler. When Config.Enabled is false it installs a no-op TracerProvider
// (zero exporter, zero network) but still sets the propagator, so a disabled
// service keeps forwarding an inbound trace context to enabled downstreams.
package tracing

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace/noop"
)

// Config describes how a service exports traces.
type Config struct {
	Enabled     bool
	Endpoint    string  // OTLP/gRPC collector address, e.g. "otel-collector:4317"
	ServiceName string  // service.name resource attribute
	Environment string  // deployment.environment: local|stage|prod
	SampleRatio float64 // 0..1; <=0 never samples, >=1 always samples
	Version     string  // optional service.version
}

// ShutdownFunc flushes and stops the tracer provider. Safe to call on a
// disabled setup (it is a no-op).
type ShutdownFunc func(context.Context) error

// Init configures the global tracer provider and propagator from cfg.
// The returned ShutdownFunc must be called on graceful shutdown to flush
// pending spans.
func Init(ctx context.Context, cfg Config) (ShutdownFunc, error) {
	// Always set the propagator so trace context flows across services even
	// when this one is disabled. This only parses/serializes headers.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Surface exporter errors through slog instead of stderr.
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		slog.Error("otel error", "err", err)
	}))

	if !cfg.Enabled {
		otel.SetTracerProvider(noop.NewTracerProvider())
		return func(context.Context) error { return nil }, nil
	}

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.Merge(resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.DeploymentEnvironment(cfg.Environment),
			semconv.ServiceVersion(cfg.Version),
		),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler(cfg.SampleRatio)),
	)
	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}

// sampler builds a parent-based ratio sampler, clamping ratio to [0, 1].
func sampler(ratio float64) sdktrace.Sampler {
	switch {
	case ratio <= 0:
		return sdktrace.ParentBased(sdktrace.NeverSample())
	case ratio >= 1:
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	default:
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
	}
}
