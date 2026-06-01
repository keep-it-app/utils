package tracing

import (
	"context"
	"net"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
)

// TestGRPCPropagation stands up an in-process gRPC server with the server
// handler and a client with the client handler over bufconn, then asserts that
// a server span and a client span are produced sharing one trace ID.
func TestGRPCPropagation(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer(grpc.StatsHandler(ServerHandler()))
	healthgrpc.RegisterHealthServer(srv, health.NewServer())
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(ClientHandler()),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	_, err = healthgrpc.NewHealthClient(conn).
		Check(context.Background(), &healthgrpc.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("health check: %v", err)
	}

	spans := exporter.GetSpans()
	if len(spans) < 2 {
		t.Fatalf("got %d spans, want >= 2 (client + server)", len(spans))
	}
	traceID := spans[0].SpanContext.TraceID()
	for _, s := range spans {
		if s.SpanContext.TraceID() != traceID {
			t.Fatalf("spans span multiple traces: %v vs %v",
				s.SpanContext.TraceID(), traceID)
		}
	}
}
