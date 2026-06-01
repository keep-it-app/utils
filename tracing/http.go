package tracing

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

// Middleware returns a chi-compatible HTTP middleware that starts a server
// span per request. The incoming W3C trace context is extracted by otelhttp
// via the global propagator. Span names use the matched chi route pattern
// (e.g. "GET /api/notes/{id}") to keep span cardinality low — never the raw
// path.
//
// Register it after chi's RequestID middleware and before the request logger,
// so spans wrap the log lines.
func Middleware(service string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		instrumented := otelhttp.NewHandler(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)

				// RoutePattern is only known after chi has matched, which
				// happens inside next; rename the span here.
				if rc := chi.RouteContext(r.Context()); rc != nil {
					if pattern := rc.RoutePattern(); pattern != "" {
						trace.SpanFromContext(r.Context()).
							SetName(r.Method + " " + pattern)
					}
				}
			}),
			service,
			otelhttp.WithSpanNameFormatter(
				func(_ string, r *http.Request) string { return r.Method },
			),
		)

		return instrumented
	}
}
