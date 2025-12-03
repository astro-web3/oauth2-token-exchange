package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync"

	"go.opentelemetry.io/otel/trace"
)

var (
	//nolint:gochecknoglobals // Global logger is intentional for application-wide logging
	defaultLogger *slog.Logger
	//nolint:gochecknoglobals // Global initOnce is intentional for thread-safe initialization
	initOnce sync.Once
)

// otelHandler wraps a slog.Handler to add OpenTelemetry trace context to logs.
type otelHandler struct {
	slog.Handler
}

func (h *otelHandler) Handle(ctx context.Context, r slog.Record) error {
	span := trace.SpanFromContext(ctx)
	spanCtx := span.SpanContext()
	if spanCtx.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
		if spanCtx.IsSampled() {
			r.AddAttrs(slog.Bool("trace_sampled", true))
		}
	}
	return h.Handler.Handle(ctx, r)
}

// InitLogger initializes the global logger.
// It is safe to call multiple times, but only the first call will take effect.
func InitLogger(level, format string) {
	initOnce.Do(func() {
		var handler slog.Handler
		if format == "json" {
			handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level:     parseLevel(level),
				AddSource: false,
				ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
					if a.Key == slog.TimeKey {
						return slog.Attr{
							Key:   "timestamp",
							Value: a.Value,
						}
					}
					return a
				},
			})
		} else {
			handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level:     parseLevel(level),
				AddSource: false,
			})
		}

		// Wrap handler with OpenTelemetry trace context integration.
		defaultLogger = slog.New(&otelHandler{Handler: handler})
	})
}

// InfoContext logs at Info level with context.
func InfoContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	if defaultLogger != nil {
		//nolint:sloglint // Using global logger is intentional for this package API
		defaultLogger.LogAttrs(ctx, slog.LevelInfo, msg, attrs...)
	}
}

// DebugContext logs at Debug level with context.
func DebugContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	if defaultLogger != nil {
		//nolint:sloglint // Using global logger is intentional for this package API
		defaultLogger.LogAttrs(ctx, slog.LevelDebug, msg, attrs...)
	}
}

// WarnContext logs at Warn level with context.
func WarnContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	if defaultLogger != nil {
		//nolint:sloglint // Using global logger is intentional for this package API
		defaultLogger.LogAttrs(ctx, slog.LevelWarn, msg, attrs...)
	}
}

// ErrorContext logs at Error level with context.
func ErrorContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	if defaultLogger != nil {
		//nolint:sloglint // Using global logger is intentional for this package API
		defaultLogger.LogAttrs(ctx, slog.LevelError, msg, attrs...)
	}
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
