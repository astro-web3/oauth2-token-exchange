package pat

import (
	"context"
	"time"
)

type Service interface {
	CreatePAT(
		ctx context.Context,
		userID, email, preferredUsername string,
		expirationDate time.Time,
	) (*PAT, string, error)

	ListPATs(ctx context.Context, userID string) ([]*PAT, error)

	DeletePAT(ctx context.Context, userID, patID string) error
}
