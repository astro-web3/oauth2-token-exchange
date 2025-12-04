package pat

import "time"

type PAT struct {
	ID             string
	UserID         string
	ExpirationDate time.Time
	CreatedAt      time.Time
}

