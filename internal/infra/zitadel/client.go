package zitadel

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	httpclient "github.com/astro-web3/oauth2-token-exchange/pkg/http"
	"github.com/astro-web3/oauth2-token-exchange/pkg/logger"
)

type UserInfo struct {
	Sub      string `json:"sub"`
	Username string `json:"preferred_username"`
	Email    string `json:"email"`
	Name     string `json:"name"`
}

type MachineUser struct {
	ID          string
	Username    string
	Name        string
	Description string
}

type PersonalAccessToken struct {
	ID             string
	UserID         string
	ExpirationDate time.Time
	CreatedAt      time.Time
}

type TokenExchanger interface {
	Exchange(ctx context.Context, pat string) (*TokenResponse, error)
	ExchangeWithActor(ctx context.Context, subjectToken, subjectTokenType, actorToken string) (*TokenResponse, error)
}

type UserInfoGetter interface {
	GetUserInfo(ctx context.Context, pat string) (*UserInfo, error)
}

type MachineUserManager interface {
	GetMachineUserByUsername(ctx context.Context, adminPAT, username string) (*MachineUser, error)
	CreateMachineUser(ctx context.Context, adminPAT, username, name, description string) (*MachineUser, error)
}

type PATManager interface {
	AddPersonalAccessToken(ctx context.Context, adminPAT, userID string, expirationDate time.Time) (*PersonalAccessToken, string, error)
	ListPersonalAccessTokens(ctx context.Context, adminPAT, userID string) ([]*PersonalAccessToken, error)
	RemovePersonalAccessToken(ctx context.Context, adminPAT, userID, patID string) error
}

type Client interface {
	TokenExchanger
	UserInfoGetter
	MachineUserManager
	PATManager
}

type zitadelClient struct {
	issuer         string
	clientID       string
	clientSecret   string
	organizationID string
}

func NewClient(issuer, clientID, clientSecret, organizationID string) Client {
	issuer = strings.TrimSuffix(issuer, "/")
	return &zitadelClient{
		issuer:         issuer,
		clientID:       clientID,
		clientSecret:   clientSecret,
		organizationID: organizationID,
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
		logger.ErrorContext(ctx, "Token exchange request failed",
			slog.String("endpoint", tokenEndpoint),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}

	if resp.StatusCode() >= http.StatusBadRequest {
		bodyStr := string(resp.Body())
		logger.ErrorContext(ctx, "Token exchange failed",
			slog.String("endpoint", tokenEndpoint),
			slog.Int("status_code", resp.StatusCode()),
			slog.String("response_body", bodyStr),
		)
		return nil, fmt.Errorf(
			"token exchange failed with status %d: %s",
			resp.StatusCode(),
			bodyStr,
		)
	}

	return &tokenResp, nil
}

func (c *zitadelClient) ExchangeWithActor(ctx context.Context, subjectToken, subjectTokenType, actorToken string) (*TokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:token-exchange")
	form.Set("subject_token", subjectToken)
	form.Set("subject_token_type", subjectTokenType)
	form.Set("actor_token", actorToken)
	form.Set("actor_token_type", "urn:ietf:params:oauth:token-type:access_token")
	form.Set("requested_token_type", "urn:ietf:params:oauth:token-type:jwt")
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
		logger.ErrorContext(ctx, "Token exchange with actor request failed",
			slog.String("endpoint", tokenEndpoint),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("token exchange with actor failed: %w", err)
	}

	if resp.StatusCode() >= http.StatusBadRequest {
		bodyStr := string(resp.Body())
		logger.ErrorContext(ctx, "Token exchange with actor failed",
			slog.String("endpoint", tokenEndpoint),
			slog.Int("status_code", resp.StatusCode()),
			slog.String("response_body", bodyStr),
		)
		return nil, fmt.Errorf(
			"token exchange with actor failed with status %d: %s",
			resp.StatusCode(),
			bodyStr,
		)
	}

	return &tokenResp, nil
}

func (c *zitadelClient) GetUserInfo(ctx context.Context, pat string) (*UserInfo, error) {
	userInfoEndpoint := c.issuer + "/oidc/v1/userinfo"

	var userInfo UserInfo
	resp, err := httpclient.GetJSON(ctx, userInfoEndpoint, pat, &userInfo)
	if err != nil {
		logger.ErrorContext(ctx, "Get userinfo request failed",
			slog.String("endpoint", userInfoEndpoint),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("get userinfo failed: %w", err)
	}

	if resp.StatusCode() >= http.StatusBadRequest {
		bodyStr := string(resp.Body())
		logger.ErrorContext(ctx, "Get userinfo failed",
			slog.String("endpoint", userInfoEndpoint),
			slog.Int("status_code", resp.StatusCode()),
			slog.String("response_body", bodyStr),
		)
		return nil, fmt.Errorf(
			"get userinfo failed with status %d: %s",
			resp.StatusCode(),
			bodyStr,
		)
	}

	return &userInfo, nil
}

func (c *zitadelClient) GetMachineUserByUsername(ctx context.Context, adminPAT, username string) (*MachineUser, error) {
	searchEndpoint := c.issuer + "/v2/users"

	reqBody := &ListUsersRequest{
		Query: &ListQuery{
			Limit: 1,
		},
		Queries: []*SearchQuery{
			{
				UserNameQuery: &UserNameQuery{
					UserName: username,
					Method:   "TEXT_QUERY_METHOD_EQUALS",
				},
			},
			{
				TypeQuery: &TypeQuery{
					Type: UserTypeMachine,
				},
			},
			{
				OrganizationIDQuery: &OrganizationIDQuery{
					OrganizationID: c.organizationID,
				},
			},
		},
	}

	var result ListUsersResponse

	resp, err := httpclient.PostJSON(ctx, searchEndpoint, adminPAT, reqBody, &result)

	if err != nil {
		logger.ErrorContext(ctx, "Get machine user by username request failed",
			slog.String("endpoint", searchEndpoint),
			slog.String("username", username),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("get machine user by username failed: %w", err)
	}

	if resp.StatusCode() >= http.StatusBadRequest {
		if resp.StatusCode() == http.StatusNotFound {
			logger.DebugContext(ctx, "User not found",
				slog.String("endpoint", searchEndpoint),
				slog.String("username", username),
			)
			return nil, nil
		}
		bodyStr := string(resp.Body())
		logger.ErrorContext(ctx, "Get machine user by username failed",
			slog.String("endpoint", searchEndpoint),
			slog.String("username", username),
			slog.Int("status_code", resp.StatusCode()),
			slog.String("response_body", bodyStr),
		)
		return nil, fmt.Errorf(
			"get machine user by username failed with status %d: %s",
			resp.StatusCode(),
			bodyStr,
		)
	}

	if len(result.Result) == 0 {
		logger.DebugContext(ctx, "Machine user not found (empty result)",
			slog.String("endpoint", searchEndpoint),
			slog.String("username", username),
		)
		return nil, nil
	}

	user := result.Result[0]
	if user.UserID == "" {
		bodyStr := string(resp.Body())
		logger.ErrorContext(ctx, "Get machine user by username returned empty user ID",
			slog.String("endpoint", searchEndpoint),
			slog.String("username", username),
			slog.String("response_body", bodyStr),
			slog.Any("parsed_result", result),
		)
		return nil, fmt.Errorf("get machine user by username returned empty user ID, response body: %s, parsed result: %+v", bodyStr, result)
	}

	if user.Machine == nil {
		bodyStr := string(resp.Body())
		logger.ErrorContext(ctx, "Get machine user by username returned non-machine user",
			slog.String("endpoint", searchEndpoint),
			slog.String("username", username),
			slog.String("response_body", bodyStr),
		)
		return nil, fmt.Errorf("machine user %s is not a machine user", username)
	}

	return &MachineUser{
		ID:          user.UserID,
		Username:    user.Username,
		Name:        user.Machine.Name,
		Description: user.Machine.Description,
	}, nil
}

func (c *zitadelClient) CreateMachineUser(ctx context.Context, adminPAT, username, name, description string) (*MachineUser, error) {
	createEndpoint := c.issuer + "/v2/users/new"

	usernamePtr := &username
	reqBody := &CreateUserRequest{
		OrganizationID: c.organizationID,
		Username:       usernamePtr,
		Machine: &CreateUserRequestMachine{
			Name:        name,
			Description: description,
		},
	}

	var result CreateUserResponse

	resp, err := httpclient.PostJSON(ctx, createEndpoint, adminPAT, reqBody, &result)
	if err != nil {
		logger.ErrorContext(ctx, "Create machine user request failed",
			slog.String("endpoint", createEndpoint),
			slog.String("username", username),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("create machine user failed: %w", err)
	}

	if resp.StatusCode() >= http.StatusBadRequest {
		if resp.StatusCode() == http.StatusConflict {
			logger.WarnContext(ctx, "Machine user already exists, retrieving existing user",
				slog.String("endpoint", createEndpoint),
				slog.String("username", username),
			)
			machineUser, getErr := c.GetMachineUserByUsername(ctx, adminPAT, username)
			if getErr != nil {
				logger.ErrorContext(ctx, "Failed to retrieve existing machine user",
					slog.String("username", username),
					slog.String("error", getErr.Error()),
				)
				return nil, fmt.Errorf("user already exists but could not be retrieved: %w", getErr)
			}
			if machineUser == nil {
				logger.ErrorContext(ctx, "Existing machine user not found",
					slog.String("username", username),
				)
				return nil, fmt.Errorf("user already exists but could not be retrieved")
			}
			return machineUser, nil
		}
		bodyStr := string(resp.Body())
		logger.ErrorContext(ctx, "Create machine user failed",
			slog.String("endpoint", createEndpoint),
			slog.String("username", username),
			slog.Int("status_code", resp.StatusCode()),
			slog.String("response_body", bodyStr),
		)
		return nil, fmt.Errorf(
			"create machine user failed with status %d: %s",
			resp.StatusCode(),
			bodyStr,
		)
	}

	if result.ID == "" {
		bodyStr := string(resp.Body())
		logger.ErrorContext(ctx, "Create machine user returned empty user ID",
			slog.String("endpoint", createEndpoint),
			slog.String("username", username),
			slog.String("response_body", bodyStr),
			slog.Any("parsed_result", result),
		)
		return nil, fmt.Errorf("create machine user returned empty user ID, response body: %s, parsed result: %+v", bodyStr, result)
	}

	return &MachineUser{
		ID:          result.ID,
		Username:    username,
		Name:        name,
		Description: description,
	}, nil
}

func (c *zitadelClient) AddPersonalAccessToken(ctx context.Context, adminPAT, userID string, expirationDate time.Time) (*PersonalAccessToken, string, error) {
	createEndpoint := fmt.Sprintf("%s/v2/users/%s/pats", c.issuer, userID)

	reqBody := &AddPersonalAccessTokenRequest{
		UserID:         userID,
		ExpirationDate: &RFC3339Time{Time: expirationDate},
	}

	var result AddPersonalAccessTokenResponse

	resp, err := httpclient.PostJSON(ctx, createEndpoint, adminPAT, reqBody, &result)
	if err != nil {
		logger.ErrorContext(ctx, "Add personal access token request failed",
			slog.String("endpoint", createEndpoint),
			slog.String("user_id", userID),
			slog.String("error", err.Error()),
		)
		return nil, "", fmt.Errorf("add personal access token failed: %w", err)
	}

	if resp.StatusCode() >= http.StatusBadRequest {
		bodyStr := string(resp.Body())
		logger.ErrorContext(ctx, "Add personal access token failed",
			slog.String("endpoint", createEndpoint),
			slog.String("user_id", userID),
			slog.Int("status_code", resp.StatusCode()),
			slog.String("response_body", bodyStr),
		)
		return nil, "", fmt.Errorf(
			"add personal access token failed with status %d: %s",
			resp.StatusCode(),
			bodyStr,
		)
	}

	bodyStr := string(resp.Body())
	logger.DebugContext(ctx, "Add personal access token response",
		slog.String("response_body", bodyStr),
		slog.Any("parsed_result", result),
	)

	if result.TokenID == "" {
		logger.WarnContext(ctx, "TokenID is empty in response",
			slog.String("response_body", bodyStr),
			slog.Any("parsed_result", result),
		)
	}

	var parsedCreatedAt time.Time
	if result.CreationDate != nil {
		parsedCreatedAt = result.CreationDate.Time
	}

	return &PersonalAccessToken{
		ID:             result.TokenID,
		UserID:         userID,
		ExpirationDate: expirationDate,
		CreatedAt:      parsedCreatedAt,
	}, result.Token, nil
}

func (c *zitadelClient) ListPersonalAccessTokens(ctx context.Context, adminPAT, userID string) ([]*PersonalAccessToken, error) {
	listEndpoint := c.issuer + "/v2/users/pats/search"

	reqBody := &ListPersonalAccessTokensRequest{
		Pagination: &PaginationRequest{
			Limit: 100,
		},
		Filters: []*PersonalAccessTokensSearchFilter{
			{
				UserIDFilter: &IDFilter{
					ID: userID,
				},
			},
		},
	}

	var result ListPersonalAccessTokensResponse

	resp, err := httpclient.PostJSON(ctx, listEndpoint, adminPAT, reqBody, &result)
	if err != nil {
		logger.ErrorContext(ctx, "List personal access tokens request failed",
			slog.String("endpoint", listEndpoint),
			slog.String("user_id", userID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("list personal access tokens failed: %w", err)
	}

	if resp.StatusCode() >= http.StatusBadRequest {
		bodyStr := string(resp.Body())
		logger.ErrorContext(ctx, "List personal access tokens failed",
			slog.String("endpoint", listEndpoint),
			slog.String("user_id", userID),
			slog.Int("status_code", resp.StatusCode()),
			slog.String("response_body", bodyStr),
		)
		return nil, fmt.Errorf(
			"list personal access tokens failed with status %d: %s",
			resp.StatusCode(),
			bodyStr,
		)
	}

	pats := make([]*PersonalAccessToken, 0, len(result.Result))
	for _, pat := range result.Result {
		patUserID := pat.UserID
		if patUserID == "" {
			patUserID = userID
		}

		var expirationDate time.Time
		if pat.ExpirationDate != nil {
			expirationDate = pat.ExpirationDate.Time
		}

		var createdAt time.Time
		if pat.CreationDate != nil {
			createdAt = pat.CreationDate.Time
		}

		pats = append(pats, &PersonalAccessToken{
			ID:             pat.ID,
			UserID:         patUserID,
			ExpirationDate: expirationDate,
			CreatedAt:      createdAt,
		})
	}

	return pats, nil
}

func (c *zitadelClient) RemovePersonalAccessToken(ctx context.Context, adminPAT, userID, patID string) error {
	deleteEndpoint := fmt.Sprintf("%s/v2/users/%s/pats/%s", c.issuer, userID, patID)

	var result RemovePersonalAccessTokenResponse

	resp, err := httpclient.DeleteJSON(ctx, deleteEndpoint, adminPAT, &result)
	if err != nil {
		logger.ErrorContext(ctx, "Remove personal access token request failed",
			slog.String("endpoint", deleteEndpoint),
			slog.String("user_id", userID),
			slog.String("pat_id", patID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("remove personal access token failed: %w", err)
	}

	if resp.StatusCode() >= http.StatusBadRequest {
		bodyStr := string(resp.Body())
		logger.ErrorContext(ctx, "Remove personal access token failed",
			slog.String("endpoint", deleteEndpoint),
			slog.String("user_id", userID),
			slog.String("pat_id", patID),
			slog.Int("status_code", resp.StatusCode()),
			slog.String("response_body", bodyStr),
		)
		return fmt.Errorf(
			"remove personal access token failed with status %d: %s",
			resp.StatusCode(),
			bodyStr,
		)
	}

	return nil
}
