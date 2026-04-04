package logger

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// NewHandler returns a JSON slog.Handler writing to stdout.
func NewHandler(opts *slog.HandlerOptions) slog.Handler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return slog.NewJSONHandler(os.Stdout, opts)
}

// SetupCustomLogger configures the global slog logger with JSON output.
func SetupCustomLogger() {
	handler := NewHandler(&slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))
}

// errEnrichHandler wraps a slog.Handler and, for ERROR records, appends
// request_id, source file and line number as structured fields.
type errEnrichHandler struct {
	next slog.Handler
}

func (h *errEnrichHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *errEnrichHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &errEnrichHandler{next: h.next.WithAttrs(attrs)}
}

func (h *errEnrichHandler) WithGroup(name string) slog.Handler {
	return &errEnrichHandler{next: h.next.WithGroup(name)}
}

func (h *errEnrichHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level == slog.LevelError {
		_, file, line, ok := runtime.Caller(3)
		if !ok {
			file = "unknown"
			line = 0
		}
		r.AddAttrs(
			slog.String("request_id", middleware.GetReqID(ctx)),
			slog.String("file", file),
			slog.Int("line", line),
		)
	}
	return h.next.Handle(ctx, r)
}

// NewHandlerWithErrEnrich returns a JSON handler that enriches ERROR records
// with request_id and source location.
func NewHandlerWithErrEnrich(opts *slog.HandlerOptions) slog.Handler {
	return &errEnrichHandler{next: NewHandler(opts)}
}

func New(log *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		log = log.With(
			slog.String("component", "middleware/logger"),
		)

		log.Info("logger middleware enabled")

		fn := func(w http.ResponseWriter, r *http.Request) {
			entry := log.With(
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
				slog.String("request_id", middleware.GetReqID(r.Context())),
			)

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			t1 := time.Now()
			defer func() {
				entry.Info("request completed",
					slog.Int("status", ww.Status()),
					slog.Int("size", ww.BytesWritten()),
					slog.Duration("duration", time.Since(t1)),
				)
			}()
			next.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}
}
