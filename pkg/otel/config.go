package otel

import (
	"go.opentelemetry.io/otel/attribute"
)

type Config struct {
	ServiceName        string
	EndpointURL        string
	Enabled            bool
	SampleRatio        float64
	Insecure           bool
	ResourceAttributes map[string]string
}

func DefaultConfig() Config {
	return Config{
		ServiceName:        "unknown-service",
		EndpointURL:        "",
		Enabled:            false,
		SampleRatio:        1.0,
		Insecure:           true,
		ResourceAttributes: make(map[string]string),
	}
}

func (c Config) toResourceAttributes() []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, len(c.ResourceAttributes)+1)
	attrs = append(attrs, attribute.String("service.name", c.ServiceName))

	for k, v := range c.ResourceAttributes {
		attrs = append(attrs, attribute.String(k, v))
	}

	return attrs
}
