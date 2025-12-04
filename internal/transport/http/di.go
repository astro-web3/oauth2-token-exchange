package http

import (
	"context"
	"fmt"
	"net/http"

	authzapp "github.com/astro-web3/oauth2-token-exchange/internal/app/authz"
	patapp "github.com/astro-web3/oauth2-token-exchange/internal/app/pat"
	"github.com/astro-web3/oauth2-token-exchange/internal/config"
	authzdomain "github.com/astro-web3/oauth2-token-exchange/internal/domain/authz"
	patdomain "github.com/astro-web3/oauth2-token-exchange/internal/domain/pat"
	"github.com/astro-web3/oauth2-token-exchange/internal/infra/cache"
	"github.com/astro-web3/oauth2-token-exchange/internal/infra/zitadel"
	pathandler "github.com/astro-web3/oauth2-token-exchange/internal/transport/http/handler"
	"github.com/astro-web3/oauth2-token-exchange/pkg/logger"
	"github.com/astro-web3/oauth2-token-exchange/pkg/otel"
	"github.com/astro-web3/oauth2-token-exchange/pkg/tracer"
)

type Server struct {
	httpServer *http.Server
}

const (
	idleTimeoutMultiplier = 2
	serviceName           = "oauth2-token-exchange"
)

func NewServer(cfg *config.Config) (*Server, error) {
	logger.InitLogger(cfg.Observability.LogLevel, cfg.Observability.Format, cfg.Observability.LogSource)

	otelCfg := otel.Config{
		ServiceName:        serviceName,
		EndpointURL:        cfg.Observability.TracingEndpointURL,
		Enabled:            cfg.Observability.TraceEnabled,
		SampleRatio:        1.0,
		Insecure:           true,
		ResourceAttributes: make(map[string]string),
	}
	if err := tracer.InitTracer(serviceName, otelCfg); err != nil {
		return nil, fmt.Errorf("failed to initialize tracer: %w", err)
	}

	redisClient, err := cache.NewRedisClient(cfg.Redis.URL, cfg.Redis.PoolSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create redis client: %w", err)
	}

	tokenCache := cache.NewTokenCache(redisClient)
	zitadelClient := zitadel.NewClient(
		cfg.Auth.Zitadel.Issuer,
		cfg.Auth.Zitadel.ClientID,
		cfg.Auth.Zitadel.ClientSecret,
		cfg.Auth.Zitadel.OrganizationID,
	)

	var authzDomainService authzdomain.Service
	if cfg.Auth.AdminMachineUser.PAT != "" {
		authzDomainService = authzdomain.NewServiceWithMachineUserSupport(
			tokenCache,
			zitadelClient,
			zitadelClient,
			cfg.Auth.AdminMachineUser.PAT,
		)
	} else {
		authzDomainService = authzdomain.NewService(tokenCache, zitadelClient)
	}
	appService := authzapp.NewService(authzDomainService)

	patDomainService := patdomain.NewService(zitadelClient, cfg.Auth.AdminMachineUser.PAT)
	patCommandService := patapp.NewCommandService(patDomainService)
	patQueryService := patapp.NewQueryService(patDomainService)
	patHandler := pathandler.NewPATHandler(patCommandService, patQueryService)

	handler := NewHandler(appService, cfg)
	router := NewRouter(handler, cfg, patHandler)

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
