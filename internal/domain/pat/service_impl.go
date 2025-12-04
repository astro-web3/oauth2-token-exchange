package pat

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/astro-web3/oauth2-token-exchange/internal/infra/zitadel"
	"github.com/astro-web3/oauth2-token-exchange/pkg/logger"
)

type service struct {
	zitadelClient zitadel.Client
	adminPAT      string
}

func NewService(zitadelClient zitadel.Client, adminPAT string) Service {
	return &service{
		zitadelClient: zitadelClient,
		adminPAT:      adminPAT,
	}
}

func (s *service) CreatePAT(
	ctx context.Context,
	userID, email, preferredUsername string,
	expirationDate time.Time,
) (*PAT, string, error) {
	if expirationDate.Before(time.Now()) {
		return nil, "", ErrInvalidExpiration
	}

	machineUser, err := s.zitadelClient.GetMachineUserByUsername(ctx, s.adminPAT, userID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get user by username: %w", err)
	}

	if machineUser == nil {
		machineUser, err = s.zitadelClient.CreateMachineUser(ctx, s.adminPAT, userID, preferredUsername, email)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create machine user: %w", err)
		}

		logger.DebugContext(ctx, "Create machine user",
			slog.String("machine_user", fmt.Sprintf("%+v", machineUser)),
		)
	}

	if machineUser == nil || machineUser.ID == "" {
		return nil, "", fmt.Errorf("machine user is nil or has empty ID after get/create")
	}

	zitadelPAT, token, err := s.zitadelClient.AddPersonalAccessToken(ctx, s.adminPAT, machineUser.ID, expirationDate)
	if err != nil {
		return nil, "", err
	}

	return &PAT{
		ID:             zitadelPAT.ID,
		MachineUserID:  machineUser.ID,
		HumanUserID:    userID,
		ExpirationDate: zitadelPAT.ExpirationDate,
		CreatedAt:      zitadelPAT.CreatedAt,
	}, token, nil
}

func (s *service) ListPATs(ctx context.Context, userID string) ([]*PAT, error) {
	machineUser, err := s.zitadelClient.GetMachineUserByUsername(ctx, s.adminPAT, userID)
	if err != nil {
		return nil, err
	}

	if machineUser == nil {
		return []*PAT{}, nil
	}

	zitadelPATs, err := s.zitadelClient.ListPersonalAccessTokens(ctx, s.adminPAT, machineUser.ID)
	if err != nil {
		return nil, err
	}

	pats := make([]*PAT, 0, len(zitadelPATs))
	for _, zp := range zitadelPATs {
		pats = append(pats, &PAT{
			ID:             zp.ID,
			MachineUserID:  machineUser.ID,
			HumanUserID:    userID,
			ExpirationDate: zp.ExpirationDate,
			CreatedAt:      zp.CreatedAt,
		})
	}

	return pats, nil
}

func (s *service) DeletePAT(ctx context.Context, userID, patID string) error {
	machineUser, err := s.zitadelClient.GetMachineUserByUsername(ctx, s.adminPAT, userID)
	if err != nil {
		return err
	}

	if machineUser == nil {
		return ErrMachineUserNotFound
	}

	return s.zitadelClient.RemovePersonalAccessToken(ctx, s.adminPAT, machineUser.ID, patID)
}
