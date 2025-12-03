package http

import (
	"context"
	"fmt"
	"net/http"

	authzapp "github.com/astro-web3/oauth2-token-exchange/internal/app/authz"
	"github.com/astro-web3/oauth2-token-exchange/internal/config"
	authzdomain "github.com/astro-web3/oauth2-token-exchange/internal/domain/authz"
	"github.com/astro-web3/oauth2-token-exchange/internal/infra/cache"
	"github.com/astro-web3/oauth2-token-exchange/internal/infra/zitadel"
	"github.com/astro-web3/oauth2-token-exchange/pkg/logger"
	"github.com/astro-web3/oauth2-token-exchange/pkg/otel"
	"github.com/astro-web3/oauth2-token-exchange/pkg/tracer"
)

type Server struct {
	httpServer *http.Server
}

const idleTimeoutMultiplier = 2

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
	router := NewRouter(handler, cfg)

	httpServer := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.ReadTimeout * idleTimeoutMultiplier,
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
