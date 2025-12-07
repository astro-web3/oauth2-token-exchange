package http

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/astro-web3/oauth2-token-exchange/pkg/tracer"
	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	DefaultTimeout = 60 * time.Second
	DefaultRetry   = 2
)

var (
	//nolint:gochecknoglobals // Global HTTP client is intentional for application-wide requests
	client *resty.Client
	//nolint:gochecknoglobals // Global once is intentional for thread-safe initialization
	once sync.Once
)

func getClient() *resty.Client {
	once.Do(func() {
		client = resty.New().
			SetTimeout(DefaultTimeout).
			SetRetryCount(DefaultRetry).
			SetHeader("Content-Type", "application/json").
			SetHeader("Accept", "application/json")
	})
	return client
}

// Client returns the shared HTTP client instance.
func Client() *resty.Client {
	return getClient()
}

type RequestOption func(*resty.Request)

func WithAuthToken(token string) RequestOption {
	return func(r *resty.Request) {
		r.SetAuthToken(token)
	}
}

func WithBasicAuth(user, pass string) RequestOption {
	return func(r *resty.Request) {
		if user != "" {
			r.SetBasicAuth(user, pass)
		}
	}
}

func WithBody(body any) RequestOption {
	return func(r *resty.Request) {
		r.SetBody(body)
	}
}

func WithResult(result any) RequestOption {
	return func(r *resty.Request) {
		if result != nil {
			r.SetResult(result).SetError(result)
		}
	}
}

func WithHeader(key, value string) RequestOption {
	return func(r *resty.Request) {
		r.SetHeader(key, value)
	}
}

func WithContentType(contentType string) RequestOption {
	return func(r *resty.Request) {
		r.SetHeader("Content-Type", contentType)
	}
}

func Request(ctx context.Context, method, url string, opts ...RequestOption) (*resty.Response, error) {
	ctx, span := startClientSpan(ctx, "http.Request", method, url)
	defer span.End()

	request := getClient().R().SetContext(ctx)

	for _, opt := range opts {
		opt(request)
	}

	injectTracingHeaders(ctx, request)

	var resp *resty.Response
	var err error

	switch method {
	case http.MethodGet:
		resp, err = request.Get(url)
	case http.MethodPost:
		resp, err = request.Post(url)
	case http.MethodPut:
		resp, err = request.Put(url)
	case http.MethodPatch:
		resp, err = request.Patch(url)
	case http.MethodDelete:
		resp, err = request.Delete(url)
	default:
		resp, err = request.Execute(method, url)
	}

	recordSpan(span, resp, err)
	return resp, err
}

func Get(ctx context.Context, url string, opts ...RequestOption) (*resty.Response, error) {
	return Request(ctx, http.MethodGet, url, opts...)
}

func Post(ctx context.Context, url string, opts ...RequestOption) (*resty.Response, error) {
	return Request(ctx, http.MethodPost, url, opts...)
}

func Delete(ctx context.Context, url string, opts ...RequestOption) (*resty.Response, error) {
	return Request(ctx, http.MethodDelete, url, opts...)
}

func startClientSpan(
	ctx context.Context,
	spanName string,
	method string,
	url string,
) (context.Context, trace.Span) {
	if ctx == nil {
		ctx = context.Background()
	}
	return tracer.Start(ctx, spanName, trace.WithAttributes(
		attribute.String("http.method", method),
		attribute.String("http.url", url),
	))
}

func recordSpan(span trace.Span, resp *resty.Response, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}
	if resp == nil {
		return
	}
	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode()))
	if resp.IsError() {
		span.SetStatus(codes.Error, resp.Status())
		return
	}
	span.SetStatus(codes.Ok, "")
}
