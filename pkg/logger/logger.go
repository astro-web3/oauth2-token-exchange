package logger

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"
)

var (
	//nolint:gochecknoglobals // Global logger is intentional for application-wide logging
	defaultLogger *slog.Logger
	//nolint:gochecknoglobals // Global initOnce is intentional for thread-safe initialization
	initOnce sync.Once
	//nolint:gochecknoglobals // Global addSource is intentional for configuration
	addSource bool
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
func InitLogger(level, format string, enableSource bool) {
	initOnce.Do(func() {
		addSource = enableSource

		var handler slog.Handler
		if format == "json" {
			handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level:     parseLevel(level),
				AddSource: addSource,
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
				AddSource: addSource,
			})
		}

		// Wrap handler with OpenTelemetry trace context integration.
		defaultLogger = slog.New(&otelHandler{Handler: handler})
	})
}

// InfoContext logs at Info level with context.
func InfoContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	if defaultLogger != nil {
		if addSource {
			logWithCaller(ctx, slog.LevelInfo, msg, attrs...)
		} else {
			//nolint:sloglint // Using global logger is intentional for this package API
			defaultLogger.LogAttrs(ctx, slog.LevelInfo, msg, attrs...)
		}
	}
}

// DebugContext logs at Debug level with context.
func DebugContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	if defaultLogger != nil {
		if addSource {
			logWithCaller(ctx, slog.LevelDebug, msg, attrs...)
		} else {
			//nolint:sloglint // Using global logger is intentional for this package API
			defaultLogger.LogAttrs(ctx, slog.LevelDebug, msg, attrs...)
		}
	}
}

// WarnContext logs at Warn level with context.
func WarnContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	if defaultLogger != nil {
		if addSource {
			logWithCaller(ctx, slog.LevelWarn, msg, attrs...)
		} else {
			//nolint:sloglint // Using global logger is intentional for this package API
			defaultLogger.LogAttrs(ctx, slog.LevelWarn, msg, attrs...)
		}
	}
}

// ErrorContext logs at Error level with context.
func ErrorContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	if defaultLogger != nil {
		if addSource {
			logWithCaller(ctx, slog.LevelError, msg, attrs...)
		} else {
			//nolint:sloglint // Using global logger is intentional for this package API
			defaultLogger.LogAttrs(ctx, slog.LevelError, msg, attrs...)
		}
	}
}

// logWithCaller creates a log record with the correct caller information.
func logWithCaller(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	pc, _, _, ok := runtime.Caller(2)
	if !ok {
		// Fallback to default behavior if caller info is not available
		//nolint:sloglint // Using global logger is intentional for this package API
		defaultLogger.LogAttrs(ctx, level, msg, attrs...)
		return
	}

	// Create a new record with the correct caller information
	// The PC will be used by the handler to extract file and line information
	r := slog.NewRecord(time.Now(), level, msg, pc)
	r.AddAttrs(attrs...)

	if defaultLogger.Handler().Enabled(ctx, level) {
		_ = defaultLogger.Handler().Handle(ctx, r)
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
