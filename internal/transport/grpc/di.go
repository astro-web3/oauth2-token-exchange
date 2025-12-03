package grpc

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"log/slog"

	"connectrpc.com/connect"
	authzapp "github.com/astro-web3/oauth2-token-exchange/internal/app/authz"
	"github.com/astro-web3/oauth2-token-exchange/internal/config"
	authzdomain "github.com/astro-web3/oauth2-token-exchange/internal/domain/authz"
	"github.com/astro-web3/oauth2-token-exchange/internal/infra/cache"
	"github.com/astro-web3/oauth2-token-exchange/internal/infra/zitadel"
	authv3connect "github.com/astro-web3/oauth2-token-exchange/pb/gen/go/envoy/service/auth/v3/authv3connect"
	"github.com/astro-web3/oauth2-token-exchange/pkg/logger"
	"github.com/astro-web3/oauth2-token-exchange/pkg/otel"
	"github.com/astro-web3/oauth2-token-exchange/pkg/tracer"
)

type Server struct {
	httpServer *http.Server
}

func NewServer(cfg *config.Config) (*Server, error) {
	logger.InitLogger(cfg.Observability.LogLevel, "json")

	otelCfg := otel.Config{
		ServiceName:        "oauth2-token-exchange",
		EndpointURL:        cfg.Observability.TracingEndpointURL,
		Enabled:            cfg.Observability.TraceEnabled,
		SampleRatio:        1.0,
		Insecure:           true,
		ResourceAttributes: make(map[string]string),
	}
	if err := tracer.InitTracer("oauth2-token-exchange", otelCfg); err != nil {
		return nil, fmt.Errorf("failed to initialize tracer: %w", err)
	}

	redisClient, err := cache.NewRedisClient(cfg.Redis.URL, cfg.Redis.PoolSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create redis client: %w", err)
	}

	tokenCache := cache.NewTokenCache(redisClient)
	tokenExchanger := zitadel.NewClient(
		cfg.Auth.Zitadel.Issuer,
		cfg.Auth.Zitadel.ClientID,
		cfg.Auth.Zitadel.ClientSecret,
	)

	domainService := authzdomain.NewService(tokenCache, tokenExchanger)
	appService := authzapp.NewService(domainService)

	handler := NewHandler(appService, cfg)

	mux := http.NewServeMux()
	path, httpHandler := authv3connect.NewAuthorizationHandler(handler, connect.WithInterceptors(
		recoveryInterceptor(),
		loggingInterceptor(),
	))
	mux.Handle(path, httpHandler)

	httpServer := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.ReadTimeout * 2,
	}

	return &Server{
		httpServer: httpServer,
	}, nil
}

func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func recoveryInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			defer func() {
				if r := recover(); r != nil {
					logger.ErrorContext(ctx, "panic recovered", slog.Any("panic", r))
				}
			}()
			return next(ctx, req)
		}
	}
}

func loggingInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()
			resp, err := next(ctx, req)
			duration := time.Since(start)

			if err != nil {
				logger.ErrorContext(ctx, "request failed",
					slog.String("method", req.Spec().Procedure),
					slog.Duration("duration", duration),
					slog.String("error", err.Error()),
				)
			} else {
				logger.InfoContext(ctx, "request completed",
					slog.String("method", req.Spec().Procedure),
					slog.Duration("duration", duration),
				)
			}

			return resp, err
		}
	}
}
