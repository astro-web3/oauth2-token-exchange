package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/astro-web3/oauth2-token-exchange/internal/config"
	authzdomain "github.com/astro-web3/oauth2-token-exchange/internal/domain/authz"
	httptransport "github.com/astro-web3/oauth2-token-exchange/internal/transport/http"
	"github.com/gin-gonic/gin"
)

type mockAppService struct {
	checkFunc func(_ context.Context, pat string, cacheTTL time.Duration, headerKeys map[string]string) (*authzdomain.AuthzDecision, error)
}

func (m *mockAppService) Check(
	ctx context.Context,
	pat string,
	cacheTTL time.Duration,
	headerKeys map[string]string,
) (*authzdomain.AuthzDecision, error) {
	if m.checkFunc != nil {
		return m.checkFunc(ctx, pat, cacheTTL, headerKeys)
	}
	return &authzdomain.AuthzDecision{
		Allow:   true,
		Headers: map[string]string{"x-user-id": "user-123"},
	}, nil
}

func createTestConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Auth.CacheTTL = 5 * time.Minute
	cfg.Auth.HeaderKeys.UserID = "x-user-id"
	cfg.Auth.HeaderKeys.UserEmail = "x-user-email"
	cfg.Auth.HeaderKeys.UserGroups = "x-user-groups"
	cfg.Auth.HeaderKeys.UserJWT = "x-user-jwt"
	return cfg
}

func TestHandler_Check_MissingAuthorizationHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := &mockAppService{}
	cfg := createTestConfig()

	handler := httptransport.NewHandler(mockService, cfg)
	router := gin.New()
	router.Any("/oauth2/token-exchange/*path", handler.Check)

	req := httptest.NewRequest(http.MethodGet, "/oauth2/token-exchange/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandler_Check_ValidPAT(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := &mockAppService{
		checkFunc: func(_ context.Context, _ string, _ time.Duration, _ map[string]string) (*authzdomain.AuthzDecision, error) {
			return &authzdomain.AuthzDecision{
				Allow: true,
				Headers: map[string]string{
					"x-user-id":     "user-123",
					"x-user-email":  "test@example.com",
					"x-user-groups": "group1,group2",
					"x-user-jwt":    "jwt-token-here",
				},
			}, nil
		},
	}

	cfg := createTestConfig()
	handler := httptransport.NewHandler(mockService, cfg)
	router := gin.New()
	router.Any("/oauth2/token-exchange/*path", handler.Check)

	req := httptest.NewRequest(http.MethodGet, "/oauth2/token-exchange/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Header().Get("X-User-Id") != "user-123" {
		t.Errorf("expected x-user-id header, got %s", w.Header().Get("X-User-Id"))
	}
	if w.Header().Get("X-User-Email") != "test@example.com" {
		t.Errorf("expected x-user-email header, got %s", w.Header().Get("X-User-Email"))
	}
	if w.Header().Get("X-User-Jwt") != "jwt-token-here" {
		t.Errorf("expected x-user-jwt header, got %s", w.Header().Get("X-User-Jwt"))
	}
}

func TestHandler_Check_InvalidPAT(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := &mockAppService{
		checkFunc: func(_ context.Context, _ string, _ time.Duration, _ map[string]string) (*authzdomain.AuthzDecision, error) {
			return &authzdomain.AuthzDecision{
				Allow:  false,
				Reason: "invalid token",
			}, nil
		},
	}

	cfg := createTestConfig()
	handler := httptransport.NewHandler(mockService, cfg)
	router := gin.New()
	router.Any("/oauth2/token-exchange/*path", handler.Check)

	req := httptest.NewRequest(http.MethodGet, "/oauth2/token-exchange/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandler_Check_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := &mockAppService{
		checkFunc: func(_ context.Context, _ string, _ time.Duration, _ map[string]string) (*authzdomain.AuthzDecision, error) {
			return nil, context.DeadlineExceeded
		},
	}

	cfg := createTestConfig()
	handler := httptransport.NewHandler(mockService, cfg)
	router := gin.New()
	router.Any("/oauth2/token-exchange/*path", handler.Check)

	req := httptest.NewRequest(http.MethodGet, "/oauth2/token-exchange/test", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestHandler_Check_LowercaseAuthorizationHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := &mockAppService{
		checkFunc: func(_ context.Context, pat string, _ time.Duration, _ map[string]string) (*authzdomain.AuthzDecision, error) {
			if pat != "valid-token" {
				t.Errorf("expected pat 'valid-token', got '%s'", pat)
			}
			return &authzdomain.AuthzDecision{
				Allow:   true,
				Headers: map[string]string{"x-user-id": "user-123"},
			}, nil
		},
	}

	cfg := createTestConfig()
	handler := httptransport.NewHandler(mockService, cfg)
	router := gin.New()
	router.Any("/oauth2/token-exchange/*path", handler.Check)

	req := httptest.NewRequest(http.MethodGet, "/oauth2/token-exchange/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}
