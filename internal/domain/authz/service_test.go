package authz

import (
	"context"
	"testing"
	"time"

	"github.com/astro-web3/oauth2-token-exchange/internal/infra/cache"
	"github.com/astro-web3/oauth2-token-exchange/internal/infra/zitadel"
)

type mockTokenCache struct {
	tokens map[string]*cache.CachedToken
}

func (m *mockTokenCache) Get(ctx context.Context, patHash string) (*cache.CachedToken, error) {
	return m.tokens[patHash], nil
}

func (m *mockTokenCache) Set(ctx context.Context, patHash string, value *cache.CachedToken, ttl time.Duration) error {
	m.tokens[patHash] = value
	return nil
}

type mockTokenExchanger struct {
	exchangeFunc func(ctx context.Context, pat string) (*zitadel.TokenResponse, error)
}

func (m *mockTokenExchanger) Exchange(ctx context.Context, pat string) (*zitadel.TokenResponse, error) {
	if m.exchangeFunc != nil {
		return m.exchangeFunc(ctx, pat)
	}
	return &zitadel.TokenResponse{
		AccessToken: "test-jwt-token",
		IDToken:     "header.eyJzdWIiOiJ1c2VyLTEyMyIsImVtYWlsIjoidGVzdEBleGFtcGxlLmNvbSIsImdyb3VwcyI6WyJncm91cDEiLCJncm91cDIiXX0.signature",
	}, nil
}

func TestService_AuthorizePAT_EmptyPAT(t *testing.T) {
	svc := NewService(&mockTokenCache{tokens: make(map[string]*cache.CachedToken)}, &mockTokenExchanger{})

	decision, err := svc.AuthorizePAT(context.Background(), "", 5*time.Minute, map[string]string{
		"user_id":     "x-user-id",
		"user_email":  "x-user-email",
		"user_groups": "x-user-groups",
		"user_jwt":    "x-user-jwt",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Allow {
		t.Error("expected decision to deny empty PAT")
	}
}

func TestService_AuthorizePAT_CacheHit(t *testing.T) {
	mockCache := &mockTokenCache{tokens: make(map[string]*cache.CachedToken)}
	patHash := hashPAT("test-token")
	mockCache.tokens[patHash] = &cache.CachedToken{
		AccessToken: "cached-jwt",
		UserID:      "user-123",
		Email:       "test@example.com",
		Groups:      []string{"group1"},
	}

	svc := NewService(mockCache, &mockTokenExchanger{})

	decision, err := svc.AuthorizePAT(context.Background(), "Bearer test-token", 5*time.Minute, map[string]string{
		"user_id":     "x-user-id",
		"user_email":  "x-user-email",
		"user_groups": "x-user-groups",
		"user_jwt":    "x-user-jwt",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !decision.Allow {
		t.Error("expected decision to allow cached token")
	}
	if decision.Headers["x-user-id"] != "user-123" {
		t.Errorf("expected user-id header, got %v", decision.Headers)
	}
}

func TestService_AuthorizePAT_CacheMiss(t *testing.T) {
	cache := &mockTokenCache{tokens: make(map[string]*cache.CachedToken)}
	exchanger := &mockTokenExchanger{}

	svc := NewService(cache, exchanger)

	decision, err := svc.AuthorizePAT(context.Background(), "Bearer valid-token", 5*time.Minute, map[string]string{
		"user_id":     "x-user-id",
		"user_email":  "x-user-email",
		"user_groups": "x-user-groups",
		"user_jwt":    "x-user-jwt",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !decision.Allow {
		t.Error("expected decision to allow after token exchange")
	}
	if decision.Headers["x-user-id"] != "user-123" {
		t.Errorf("expected user-id header, got %v", decision.Headers)
	}
}
