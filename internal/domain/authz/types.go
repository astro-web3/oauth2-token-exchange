package authz

type TokenClaims struct {
	UserID string
	Email  string
	Groups []string
	JWT    string
}

// AuthzDecision represents the authorization decision returned by the domain service.
//
//nolint:revive // AuthzDecision keeps the domain name in the type for clarity
type AuthzDecision struct {
	Allow   bool
	Headers map[string]string
	Reason  string
}
