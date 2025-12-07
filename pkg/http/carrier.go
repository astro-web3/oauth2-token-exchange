package http

import (
	"context"

	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel"
)

type restyHeaderCarrier struct {
	request *resty.Request
}

func (c *restyHeaderCarrier) Get(key string) string {
	return c.request.Header.Get(key)
}

func (c *restyHeaderCarrier) Set(key, value string) {
	c.request.SetHeader(key, value)
}

func (c *restyHeaderCarrier) Keys() []string {
	headers := c.request.Header
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	return keys
}

func injectTracingHeaders(ctx context.Context, request *resty.Request) {
	propagator := otel.GetTextMapPropagator()
	if propagator != nil {
		carrier := &restyHeaderCarrier{request: request}
		propagator.Inject(ctx, carrier)
	}
}
