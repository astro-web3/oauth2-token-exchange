package tracer

import (
	"context"
	"sync"

	"github.com/astro-web3/oauth2-token-exchange/pkg/otel"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var (
	defaultTracer trace.Tracer
	initOnce      sync.Once
	errInit       error
)

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

func Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if defaultTracer == nil {
		noopTracer := noop.NewTracerProvider().Tracer("noop")
		newCtx, span := noopTracer.Start(ctx, spanName, opts...)
		return newCtx, span
	}

	newCtx, span := defaultTracer.Start(ctx, spanName, opts...)
	return newCtx, span
}
