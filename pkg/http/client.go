package http

import (
	"context"
	"net/http"
	"net/url"
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
			SetHeader("Content-Type", "application/json")
	})
	return client
}

// Client returns the shared HTTP client instance.
func Client() *resty.Client {
	return getClient()
}

func PostJSON(ctx context.Context, url, token string, body, result any) (*resty.Response, error) {
	ctx, span := startClientSpan(ctx, "http.PostJSON", http.MethodPost, url)
	defer span.End()

	resp, err := getClient().R().
		SetContext(ctx).
		SetAuthToken(token).
		SetBody(body).
		SetResult(result).
		SetError(result).
		Post(url)

	recordSpan(span, resp, err)
	return resp, err
}

func PostForm(
	ctx context.Context,
	url string,
	form url.Values,
	basicUser string,
	basicPass string,
	result any,
) (*resty.Response, error) {
	ctx, span := startClientSpan(ctx, "http.PostForm", http.MethodPost, url)
	defer span.End()

	request := getClient().R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetBody(form.Encode())

	if basicUser != "" {
		request.SetBasicAuth(basicUser, basicPass)
	}

	if result != nil {
		request.SetResult(result).
			SetError(result)
	}

	resp, err := request.Post(url)
	recordSpan(span, resp, err)
	return resp, err
}

func GetJSON(ctx context.Context, url, token string, result any) (*resty.Response, error) {
	ctx, span := startClientSpan(ctx, "http.GetJSON", http.MethodGet, url)
	defer span.End()

	resp, err := getClient().R().
		SetContext(ctx).
		SetAuthToken(token).
		SetResult(result).
		SetError(result).
		Get(url)

	recordSpan(span, resp, err)
	return resp, err
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
