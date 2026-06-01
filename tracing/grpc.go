package tracing

import (
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc/stats"
)

// ServerHandler returns a gRPC stats handler that creates server spans and
// extracts the incoming trace context from request metadata. Pass it to
// grpc.NewServer via grpc.StatsHandler.
func ServerHandler() stats.Handler {
	return otelgrpc.NewServerHandler()
}

// ClientHandler returns a gRPC stats handler that creates client spans and
// injects the trace context into request metadata. Pass it to grpc.NewClient
// via grpc.WithStatsHandler.
func ClientHandler() stats.Handler {
	return otelgrpc.NewClientHandler()
}
