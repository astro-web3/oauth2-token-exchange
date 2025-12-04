package pat

import "errors"

var (
	ErrPATNotFound      = errors.New("PAT not found")
	ErrPATExpired       = errors.New("PAT expired")
	ErrInvalidExpiration = errors.New("invalid expiration date")
	ErrMachineUserNotFound = errors.New("machine user not found")
	ErrFailedToCreatePAT  = errors.New("failed to create PAT")
)

