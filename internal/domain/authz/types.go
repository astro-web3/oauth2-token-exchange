package authz

type TokenClaims struct {
	UserID string
	Email  string
	Groups []string
	JWT    string
}

type AuthzDecision struct {
	Allow   bool
	Headers map[string]string
	Reason  string
}
