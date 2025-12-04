package pat

import (
	"context"
	"log/slog"

	patdomain "github.com/astro-web3/oauth2-token-exchange/internal/domain/pat"
	"github.com/astro-web3/oauth2-token-exchange/pkg/logger"
	"github.com/astro-web3/oauth2-token-exchange/pkg/tracer"
	"go.opentelemetry.io/otel/attribute"
)

type QueryService struct {
	domainService patdomain.Service
}

func NewQueryService(domainService patdomain.Service) *QueryService {
	return &QueryService{
		domainService: domainService,
	}
}

func (s *QueryService) ListPATs(ctx context.Context, userID string) ([]*patdomain.PAT, error) {
	ctx, span := tracer.Start(ctx, "app.pat.ListPATs")
	defer span.End()

	span.SetAttributes(attribute.String("pat.user_id", userID))

	logger.InfoContext(ctx, "listing PATs", slog.String("user_id", userID))

	pats, err := s.domainService.ListPATs(ctx, userID)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(attribute.Int("pat.count", len(pats)))
	logger.InfoContext(ctx, "PATs listed successfully", slog.Int("count", len(pats)))

	return pats, nil
}

