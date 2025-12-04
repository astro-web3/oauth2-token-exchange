package authz_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/astro-web3/oauth2-token-exchange/internal/domain/authz"
	"github.com/astro-web3/oauth2-token-exchange/internal/infra/cache"
	"github.com/astro-web3/oauth2-token-exchange/internal/infra/zitadel"
)

type mockTokenCache struct {
	tokens map[string]*cache.CachedToken
}

func (m *mockTokenCache) Get(_ context.Context, patHash string) (*cache.CachedToken, error) {
	return m.tokens[patHash], nil
}

func (m *mockTokenCache) Set(_ context.Context, patHash string, value *cache.CachedToken, _ time.Duration) error {
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

func (m *mockTokenExchanger) ExchangeWithActor(
	ctx context.Context,
	subjectToken, _ string, _ string,
) (*zitadel.TokenResponse, error) {
	return m.Exchange(ctx, subjectToken)
}

type mockUserInfoGetter struct {
	userInfoFunc func(ctx context.Context, pat string) (*zitadel.UserInfo, error)
}

func (m *mockUserInfoGetter) GetUserInfo(ctx context.Context, pat string) (*zitadel.UserInfo, error) {
	if m.userInfoFunc != nil {
		return m.userInfoFunc(ctx, pat)
	}
	return &zitadel.UserInfo{
		Sub:      "user-123",
		Username: "user-123",
		Email:    "test@example.com",
		Name:     "Test User",
	}, nil
}

type mockZitadelClient struct {
	*mockTokenExchanger
	*mockUserInfoGetter
}

func (m *mockZitadelClient) GetMachineUserByUsername(ctx context.Context, adminPAT, username string) (*zitadel.MachineUser, error) {
	return nil, nil
}

func (m *mockZitadelClient) CreateMachineUser(ctx context.Context, adminPAT, username, name, description string) (*zitadel.MachineUser, error) {
	return nil, nil
}

func (m *mockZitadelClient) AddPersonalAccessToken(ctx context.Context, adminPAT, userID string, expirationDate time.Time) (*zitadel.PersonalAccessToken, string, error) {
	return nil, "", nil
}

func (m *mockZitadelClient) ListPersonalAccessTokens(ctx context.Context, adminPAT, userID string) ([]*zitadel.PersonalAccessToken, error) {
	return nil, nil
}

func (m *mockZitadelClient) RemovePersonalAccessToken(ctx context.Context, adminPAT, userID, patID string) error {
	return nil
}

func TestService_AuthorizePAT_EmptyPAT(t *testing.T) {
	svc := authz.NewService(&mockTokenCache{tokens: make(map[string]*cache.CachedToken)}, &mockTokenExchanger{})

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
	patHash := hashPATForTest("test-token")
	mockCache.tokens[patHash] = &cache.CachedToken{
		AccessToken: "cached-jwt",
		UserID:      "user-123",
		Email:       "test@example.com",
		Groups:      []string{"group1"},
	}

	svc := authz.NewService(mockCache, &mockTokenExchanger{})

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
	client := &mockZitadelClient{
		mockTokenExchanger: &mockTokenExchanger{},
		mockUserInfoGetter: &mockUserInfoGetter{},
	}

	svc := authz.NewServiceWithMachineUserSupport(cache, client, client, "admin-pat")

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

func hashPATForTest(pat string) string {
	hash := sha256.Sum256([]byte(pat))
	return hex.EncodeToString(hash[:])
}
