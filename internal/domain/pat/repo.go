package pat

import (
	"context"
)

type CommandRepository interface {
	Create(ctx context.Context, pat *PAT) error
	Delete(ctx context.Context, userID, patID string) error
}

type QueryRepository interface {
	ListByUserID(ctx context.Context, userID string) ([]*PAT, error)
	GetByID(ctx context.Context, patID string) (*PAT, error)
}
