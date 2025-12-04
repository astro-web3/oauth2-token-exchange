package pat

import "time"

type PAT struct {
	ID             string
	MachineUserID  string
	HumanUserID    string
	ExpirationDate time.Time
	CreatedAt      time.Time
}
