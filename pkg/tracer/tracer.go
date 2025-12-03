package tracer

import (
	"context"
	"sync"

	"github.com/astro-web3/oauth2-token-exchange/pkg/otel"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var (
	//nolint:gochecknoglobals // Global tracer is intentional for application-wide tracing
	defaultTracer trace.Tracer
	//nolint:gochecknoglobals // Global initOnce is intentional for thread-safe initialization
	initOnce sync.Once
	errInit  error
)

// InitTracer initializes the global tracer.
// It is safe to call multiple times, but only the first call will take effect.
// Returns error only from the first call.
func InitTracer(serviceName string, cfg otel.Config) error {
	initOnce.Do(func() {
		cfg.ServiceName = serviceName
		t, err := otel.InitTracer(cfg)
		if err != nil {
			errInit = err
			return
		}

		defaultTracer = t
	})

	return errInit
}

// Start starts a new span with the given name.
// The returned span should be ended by the caller.
//
//nolint:spancheck // Span is returned to caller, caller is responsible for ending it
func Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if defaultTracer == nil {
		// Return noop span if tracer is not initialized.
		noopTracer := noop.NewTracerProvider().Tracer("noop")
		newCtx, span := noopTracer.Start(ctx, spanName, opts...)
		return newCtx, span
	}

	newCtx, span := defaultTracer.Start(ctx, spanName, opts...)
	return newCtx, span
}
