package pat

import (
	"context"
	"time"

	patdomain "github.com/astro-web3/oauth2-token-exchange/internal/domain/pat"
	"github.com/astro-web3/oauth2-token-exchange/pkg/tracer"
	"go.opentelemetry.io/otel/attribute"
)

type CommandService struct {
	domainService patdomain.Service
}

func NewCommandService(domainService patdomain.Service) *CommandService {
	return &CommandService{
		domainService: domainService,
	}
}

func (s *CommandService) CreatePAT(
	ctx context.Context,
	userID, email, preferredUsername string,
	expirationDate time.Time,
) (*patdomain.PAT, string, error) {
	ctx, span := tracer.Start(ctx, "app.pat.CreatePAT")
	defer span.End()

	span.SetAttributes(
		attribute.String("pat.user_id", userID),
		attribute.String("pat.email", email),
	)

	pat, token, err := s.domainService.CreatePAT(ctx, userID, email, preferredUsername, expirationDate)
	if err != nil {
		span.RecordError(err)
		return nil, "", err
	}

	span.SetAttributes(attribute.String("pat.id", pat.ID))

	return pat, token, nil
}

func (s *CommandService) DeletePAT(ctx context.Context, userID, patID string) error {
	ctx, span := tracer.Start(ctx, "app.pat.DeletePAT")
	defer span.End()

	span.SetAttributes(
		attribute.String("pat.user_id", userID),
		attribute.String("pat.id", patID),
	)

	err := s.domainService.DeletePAT(ctx, userID, patID)
	if err != nil {
		span.RecordError(err)
		return err
	}

	return nil
}
