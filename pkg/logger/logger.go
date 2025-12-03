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
	defaultLogger *slog.Logger
	initOnce      sync.Once
)

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

		defaultLogger = slog.New(&otelHandler{Handler: handler})
	})
}

func InfoContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	if defaultLogger != nil {
		defaultLogger.LogAttrs(ctx, slog.LevelInfo, msg, attrs...)
	}
}

func DebugContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	if defaultLogger != nil {
		defaultLogger.LogAttrs(ctx, slog.LevelDebug, msg, attrs...)
	}
}

func WarnContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	if defaultLogger != nil {
		defaultLogger.LogAttrs(ctx, slog.LevelWarn, msg, attrs...)
	}
}

func ErrorContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	if defaultLogger != nil {
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
