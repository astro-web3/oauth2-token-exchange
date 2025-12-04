package authz

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"log/slog"

	"github.com/astro-web3/oauth2-token-exchange/internal/infra/cache"
	"github.com/astro-web3/oauth2-token-exchange/internal/infra/zitadel"
	"github.com/astro-web3/oauth2-token-exchange/pkg/logger"
)

const jwtPartsCount = 3

type Service interface {
	AuthorizePAT(
		ctx context.Context,
		pat string,
		cacheTTL time.Duration,
		headerKeys map[string]string,
	) (*AuthzDecision, error)
}

type service struct {
	tokenCache     cache.TokenCache
	tokenExchanger zitadel.TokenExchanger
	userInfoGetter zitadel.UserInfoGetter
	adminPAT       string
}

func NewService(tokenCache cache.TokenCache, tokenExchanger zitadel.TokenExchanger) Service {
	return &service{
		tokenCache:     tokenCache,
		tokenExchanger: tokenExchanger,
	}
}

func NewServiceWithMachineUserSupport(
	tokenCache cache.TokenCache,
	tokenExchanger zitadel.TokenExchanger,
	userInfoGetter zitadel.UserInfoGetter,
	adminPAT string,
) Service {
	return &service{
		tokenCache:     tokenCache,
		tokenExchanger: tokenExchanger,
		userInfoGetter: userInfoGetter,
		adminPAT:       adminPAT,
	}
}

func (s *service) AuthorizePAT(
	ctx context.Context,
	pat string,
	cacheTTL time.Duration,
	headerKeys map[string]string,
) (*AuthzDecision, error) {
	if pat == "" {
		return &AuthzDecision{
			Allow:  false,
			Reason: "PAT is empty",
		}, nil
	}

	pat = strings.TrimPrefix(pat, "Bearer ")
	pat = strings.TrimSpace(pat)

	if pat == "" {
		return &AuthzDecision{
			Allow:  false,
			Reason: "PAT is empty after trimming",
		}, nil
	}

	patHash := hashPAT(pat)

	cached, err := s.tokenCache.Get(ctx, patHash)
	if err != nil && !errors.Is(err, cache.ErrCacheMiss) {
		logger.WarnContext(ctx, "failed to get from cache, will exchange token", slog.String("error", err.Error()))
	}

	if err == nil && cached != nil {
		return s.buildDecision(cached, headerKeys), nil
	}

	var tokenResp *zitadel.TokenResponse
	var parseErr error

	if s.userInfoGetter == nil || s.adminPAT == "" {
		return &AuthzDecision{
			Allow:  false,
			Reason: "user info getter or admin pat is not set",
		}, nil
	}

	userInfo, err := s.userInfoGetter.GetUserInfo(ctx, pat)

	if err != nil || userInfo == nil {
		logger.WarnContext(ctx, "failed to get user info", slog.String("error", err.Error()))
		return &AuthzDecision{
			Allow:  false,
			Reason: fmt.Sprintf("failed to get user info or user info is nil: %v", err),
		}, nil
	}

	client, ok := s.tokenExchanger.(zitadel.Client)

	if !ok {
		return &AuthzDecision{
			Allow:  false,
			Reason: "token exchanger is not a zitadel client",
		}, nil
	}

	tokenResp, err = client.ExchangeWithActor(
		ctx,
		userInfo.Username,
		"urn:zitadel:params:oauth:token-type:user_id",
		s.adminPAT,
	)

	if err != nil {
		return &AuthzDecision{
			Allow:  false,
			Reason: fmt.Sprintf("token exchange with actor failed: %v", err),
		}, nil
	}

	idTokenClaims, parseErr := parseIDTokenClaims(tokenResp.IDToken)
	if parseErr != nil {
		return &AuthzDecision{
			Allow:  false,
			Reason: fmt.Sprintf("parse id token failed: %v", parseErr),
		}, nil
	}

	cachedToken := &cache.CachedToken{
		AccessToken:       tokenResp.AccessToken,
		UserID:            idTokenClaims.Sub,
		Email:             idTokenClaims.Email,
		Groups:            idTokenClaims.Groups,
		PreferredUsername: idTokenClaims.PreferredUsername,
	}

	if setErr := s.tokenCache.Set(ctx, patHash, cachedToken, cacheTTL); setErr != nil {
		logger.WarnContext(ctx, "failed to set cache", slog.String("error", setErr.Error()))
	}

	tokenClaims := &TokenClaims{
		UserID:            idTokenClaims.Sub,
		Email:             idTokenClaims.Email,
		Groups:            idTokenClaims.Groups,
		PreferredUsername: idTokenClaims.PreferredUsername,
		JWT:               tokenResp.AccessToken,
	}

	return s.buildDecisionFromClaims(tokenClaims, headerKeys), nil
}

func (s *service) buildDecision(cached *cache.CachedToken, headerKeys map[string]string) *AuthzDecision {
	headers := make(map[string]string)
	if cached.UserID != "" {
		headers[headerKeys["user_id"]] = cached.UserID
	}
	if cached.Email != "" {
		headers[headerKeys["user_email"]] = cached.Email
	}
	if len(cached.Groups) > 0 {
		headers[headerKeys["user_groups"]] = strings.Join(cached.Groups, ",")
	}
	if cached.PreferredUsername != "" {
		headers[headerKeys["user_preferred_username"]] = cached.PreferredUsername
	}
	if cached.AccessToken != "" {
		headers[headerKeys["user_jwt"]] = cached.AccessToken
	}

	return &AuthzDecision{
		Allow:   true,
		Headers: headers,
	}
}

func (s *service) buildDecisionFromClaims(claims *TokenClaims, headerKeys map[string]string) *AuthzDecision {
	headers := make(map[string]string)
	if claims.UserID != "" {
		headers[headerKeys["user_id"]] = claims.UserID
	}
	if claims.Email != "" {
		headers[headerKeys["user_email"]] = claims.Email
	}
	if len(claims.Groups) > 0 {
		headers[headerKeys["user_groups"]] = strings.Join(claims.Groups, ",")
	}
	if claims.PreferredUsername != "" {
		headers[headerKeys["user_preferred_username"]] = claims.PreferredUsername
	}
	if claims.JWT != "" {
		headers[headerKeys["user_jwt"]] = claims.JWT
	}

	return &AuthzDecision{
		Allow:   true,
		Headers: headers,
	}
}

func hashPAT(pat string) string {
	hash := sha256.Sum256([]byte(pat))
	return hex.EncodeToString(hash[:])
}

type idTokenClaims struct {
	Sub               string   `json:"sub"`
	Email             string   `json:"email"`
	Groups            []string `json:"groups"`
	PreferredUsername string   `json:"preferred_username"`
}

func parseIDTokenClaims(idToken string) (*idTokenClaims, error) {
	if idToken == "" {
		return nil, errors.New("id token is empty")
	}

	parts := strings.Split(idToken, ".")
	if len(parts) != jwtPartsCount {
		return nil, errors.New("invalid jwt format")
	}

	payloadSegment := parts[1]

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadSegment)
	if err != nil {
		return nil, fmt.Errorf("failed to decode jwt payload: %w", err)
	}

	var claims idTokenClaims
	if err = json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal jwt payload: %w", err)
	}

	return &claims, nil
}
