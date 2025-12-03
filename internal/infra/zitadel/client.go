package zitadel

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"log/slog"

	httpclient "github.com/astro-web3/oauth2-token-exchange/pkg/http"
	"github.com/astro-web3/oauth2-token-exchange/pkg/logger"
)

type TokenExchangeRequest struct {
	GrantType          string  `json:"grant_type"`
	SubjectToken       string  `json:"subject_token"`
	SubjectTokenType   string  `json:"subject_token_type"`
	RequestedTokenType *string `json:"requested_token_type,omitempty"`
	Scope              *string `json:"scope,omitempty"`
	Audience           *string `json:"audience,omitempty"`
}

type TokenResponse struct {
	AccessToken     string `json:"access_token"`
	TokenType       string `json:"token_type"`
	IssuedTokenType string `json:"issued_token_type,omitempty"`
	RefreshToken    string `json:"refresh_token,omitempty"`
	IDToken         string `json:"id_token,omitempty"`
	ExpiresIn       int64  `json:"expires_in,omitempty"`
	Scope           string `json:"scope,omitempty"`
}

type TokenExchanger interface {
	Exchange(ctx context.Context, pat string) (*TokenResponse, error)
}

type zitadelClient struct {
	issuer       string
	clientID     string
	clientSecret string
}

func NewClient(issuer, clientID, clientSecret string) TokenExchanger {
	issuer = strings.TrimSuffix(issuer, "/")
	return &zitadelClient{
		issuer:       issuer,
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

func (c *zitadelClient) Exchange(ctx context.Context, pat string) (*TokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:token-exchange")
	form.Set("subject_token", pat)
	form.Set("subject_token_type", "urn:ietf:params:oauth:token-type:access_token")
	form.Set("requested_token_type", "urn:ietf:params:oauth:token-type:access_token")
	form.Set("scope", "openid")
	form.Set("audience", c.clientID)

	tokenEndpoint := c.issuer + "/oauth/v2/token"

	var tokenResp TokenResponse
	resp, err := httpclient.PostForm(
		ctx,
		tokenEndpoint,
		form,
		c.clientID,
		c.clientSecret,
		&tokenResp,
	)
	if err != nil {
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}

	logger.InfoContext(ctx, "token exchange response", slog.String("token_resp", fmt.Sprintf("%+v", tokenResp)))

	if resp.StatusCode() >= http.StatusBadRequest {
		return nil, fmt.Errorf(
			"token exchange failed with status %d: %s",
			resp.StatusCode(),
			string(resp.Body()),
		)
	}

	return &tokenResp, nil
}
