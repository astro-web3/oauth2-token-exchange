package otel

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var (
	tracerProvider   *sdktrace.TracerProvider
	tracerProviderMu sync.Mutex
)

func InitTracer(cfg Config) (trace.Tracer, error) {
	tracerProviderMu.Lock()
	defer tracerProviderMu.Unlock()

	if !cfg.Enabled || cfg.EndpointURL == "" {
		tp := noop.NewTracerProvider()
		otel.SetTracerProvider(tp)
		return tp.Tracer(cfg.ServiceName), nil
	}

	ctx := context.Background()

	exporter, err := createExporter(ctx, cfg)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(cfg.toResourceAttributes()...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	sampler := sdktrace.TraceIDRatioBased(cfg.SampleRatio)
	if cfg.SampleRatio <= 0 {
		sampler = sdktrace.NeverSample()
	} else if cfg.SampleRatio >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracerProvider = tp
	return tp.Tracer(cfg.ServiceName), nil
}

func createExporter(ctx context.Context, cfg Config) (sdktrace.SpanExporter, error) {
	if strings.HasPrefix(cfg.EndpointURL, "grpc://") {
		return createGRPCExporter(ctx, cfg)
	}

	return createHTTPExporter(ctx, cfg)
}

func createGRPCExporter(ctx context.Context, cfg Config) (sdktrace.SpanExporter, error) {
	endpoint := strings.TrimPrefix(cfg.EndpointURL, "grpc://")

	grpcOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
	}
	if cfg.Insecure {
		grpcOpts = append(grpcOpts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, grpcOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP gRPC exporter: %w", err)
	}

	return exporter, nil
}

func createHTTPExporter(ctx context.Context, cfg Config) (sdktrace.SpanExporter, error) {
	httpOpts := []otlptracehttp.Option{
		otlptracehttp.WithEndpointURL(cfg.EndpointURL),
	}
	if cfg.Insecure {
		httpOpts = append(httpOpts, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(ctx, httpOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP HTTP exporter: %w", err)
	}

	return exporter, nil
}

func Shutdown(ctx context.Context) error {
	tracerProviderMu.Lock()
	defer tracerProviderMu.Unlock()

	if tracerProvider == nil {
		return nil
	}

	if err := tracerProvider.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown tracer provider: %w", err)
	}

	tracerProvider = nil
	return nil
}

func GetTracer(serviceName string) trace.Tracer {
	return otel.Tracer(serviceName)
}
