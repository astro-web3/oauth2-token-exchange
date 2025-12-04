package authz

import (
	"context"
	"time"

	"github.com/astro-web3/oauth2-token-exchange/internal/domain/authz"
	"github.com/astro-web3/oauth2-token-exchange/pkg/tracer"
	"go.opentelemetry.io/otel/attribute"
)

type Service interface {
	Check(
		ctx context.Context,
		pat string,
		cacheTTL time.Duration,
		headerKeys map[string]string,
	) (*authz.AuthzDecision, error)
}

type service struct {
	domainService authz.Service
}

func NewService(domainService authz.Service) Service {
	return &service{
		domainService: domainService,
	}
}

func (s *service) Check(
	ctx context.Context,
	pat string,
	cacheTTL time.Duration,
	headerKeys map[string]string,
) (*authz.AuthzDecision, error) {
	ctx, span := tracer.Start(ctx, "app.authz.Check")
	defer span.End()

	span.SetAttributes(
		attribute.String("pat.prefix", getPATPrefix(pat)),
	)

	decision, err := s.domainService.AuthorizePAT(ctx, pat, cacheTTL, headerKeys)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	if decision.Allow {
		span.SetAttributes(attribute.Bool("authz.allowed", true))
	} else {
		span.SetAttributes(
			attribute.Bool("authz.allowed", false),
			attribute.String("authz.reason", decision.Reason),
		)
	}

	return decision, nil
}

const patPrefixLength = 8

func getPATPrefix(pat string) string {
	if len(pat) > patPrefixLength {
		return pat[:patPrefixLength] + "..."
	}
	return "***"
}
