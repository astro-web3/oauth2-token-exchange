package zitadel

import (
	"encoding/json"
	"time"
)

type UserType string

const (
	UserTypeUnspecified UserType = "TYPE_UNSPECIFIED"
	UserTypeMachine     UserType = "TYPE_MACHINE"
	UserTypeHuman       UserType = "TYPE_HUMAN"
)

// RFC3339Time is a time.Time wrapper that marshals/unmarshals as RFC3339 string
type RFC3339Time struct {
	time.Time
}

func (t RFC3339Time) MarshalJSON() ([]byte, error) {
	if t.Time.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.Time.Format(time.RFC3339))
}

func (t *RFC3339Time) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == "" || s == "null" {
		t.Time = time.Time{}
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}
	t.Time = parsed
	return nil
}

type TokenExchangeRequest struct {
	GrantType          string  `json:"grant_type"`
	SubjectToken       string  `json:"subject_token"`
	SubjectTokenType   string  `json:"subject_token_type"`
	ActorToken         *string `json:"actor_token,omitempty"`
	ActorTokenType     *string `json:"actor_token_type,omitempty"`
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

// ListUsersRequest represents the request for ListUsers API
type ListUsersRequest struct {
	Query         *ListQuery     `json:"query,omitempty"`
	SortingColumn string         `json:"sortingColumn,omitempty"`
	Queries       []*SearchQuery `json:"queries,omitempty"`
}

// ListQuery represents pagination and sorting parameters
type ListQuery struct {
	Offset uint64 `json:"offset,omitempty"`
	Limit  uint32 `json:"limit,omitempty"`
	Asc    bool   `json:"asc,omitempty"`
}

// SearchQuery represents a search query (oneof type)
type SearchQuery struct {
	UserNameQuery       *UserNameQuery       `json:"userNameQuery,omitempty"`
	TypeQuery           *TypeQuery           `json:"typeQuery,omitempty"`
	OrganizationIDQuery *OrganizationIDQuery `json:"organizationIdQuery,omitempty"`
}

// TypeQuery represents a type search query
type TypeQuery struct {
	Type UserType `json:"type"`
}

// UserNameQuery represents a username search query
type UserNameQuery struct {
	UserName string `json:"userName"`
	Method   string `json:"method"`
}

// OrganizationIDQuery represents an organization ID query
type OrganizationIDQuery struct {
	OrganizationID string `json:"organizationId"`
}

// ListUsersResponse represents the response from ListUsers API
type ListUsersResponse struct {
	Details       *ListDetails `json:"details,omitempty"`
	SortingColumn string       `json:"sortingColumn,omitempty"`
	Result        []*User      `json:"result,omitempty"`
}

// ListDetails represents list pagination details
type ListDetails struct {
	TotalResult       string       `json:"totalResult,omitempty"`
	ProcessedSequence uint64       `json:"processedSequence,omitempty"`
	Timestamp         *RFC3339Time `json:"timestamp,omitempty"`
}

// User represents a user in ZITADEL
type User struct {
	UserID             string               `json:"userId,omitempty"`
	State              string               `json:"state,omitempty"`
	Username           string               `json:"username,omitempty"`
	LoginNames         []string             `json:"loginNames,omitempty"`
	PreferredLoginName string               `json:"preferredLoginName,omitempty"`
	Machine            *MachineUserResponse `json:"machine,omitempty"`
}

// MachineUserResponse represents a machine user in API response
type MachineUserResponse struct {
	Name            string `json:"name,omitempty"`
	Description     string `json:"description,omitempty"`
	HasSecret       bool   `json:"hasSecret,omitempty"`
	AccessTokenType string `json:"accessTokenType,omitempty"`
}

// CreateUserRequest represents the request for CreateUser API
type CreateUserRequest struct {
	OrganizationID string                    `json:"organizationId"`
	UserID         *string                   `json:"userId,omitempty"`
	Username       *string                   `json:"username,omitempty"`
	Machine        *CreateUserRequestMachine `json:"machine,omitempty"`
}

// CreateUserRequestMachine represents machine user creation data
type CreateUserRequestMachine struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// CreateUserResponse represents the response from CreateUser API
type CreateUserResponse struct {
	ID           string       `json:"id"`
	CreationDate *RFC3339Time `json:"creationDate,omitempty"`
	EmailCode    *string      `json:"emailCode,omitempty"`
	PhoneCode    *string      `json:"phoneCode,omitempty"`
}

// AddPersonalAccessTokenRequest represents the request for AddPersonalAccessToken API
type AddPersonalAccessTokenRequest struct {
	UserID         string       `json:"userId"`
	ExpirationDate *RFC3339Time `json:"expirationDate"`
}

// AddPersonalAccessTokenResponse represents the response from AddPersonalAccessToken API
type AddPersonalAccessTokenResponse struct {
	CreationDate *RFC3339Time `json:"creationDate,omitempty"`
	TokenID      string       `json:"tokenId"`
	Token        string       `json:"token"`
}

// ListPersonalAccessTokensRequest represents the request for ListPersonalAccessTokens API
type ListPersonalAccessTokensRequest struct {
	Pagination    *PaginationRequest                  `json:"pagination,omitempty"`
	SortingColumn string                              `json:"sortingColumn,omitempty"`
	Filters       []*PersonalAccessTokensSearchFilter `json:"filters,omitempty"`
}

// PaginationRequest represents pagination parameters
type PaginationRequest struct {
	Offset uint64 `json:"offset,omitempty"`
	Limit  uint32 `json:"limit,omitempty"`
	Asc    bool   `json:"asc,omitempty"`
}

// PersonalAccessTokensSearchFilter represents a search filter (oneof type)
type PersonalAccessTokensSearchFilter struct {
	UserIDFilter *IDFilter `json:"userIdFilter,omitempty"`
}

// IDFilter represents an ID filter
type IDFilter struct {
	ID string `json:"id"`
}

// ListPersonalAccessTokensResponse represents the response from ListPersonalAccessTokens API
type ListPersonalAccessTokensResponse struct {
	Pagination *PaginationResponse            `json:"pagination,omitempty"`
	Result     []*PersonalAccessTokenResponse `json:"result,omitempty"`
}

// PaginationResponse represents pagination response
type PaginationResponse struct {
	TotalResult  uint64 `json:"totalResult,omitempty"`
	AppliedLimit uint64 `json:"appliedLimit,omitempty"`
}

// PersonalAccessTokenResponse represents a personal access token in API response
type PersonalAccessTokenResponse struct {
	CreationDate   *RFC3339Time `json:"creationDate,omitempty"`
	ChangeDate     *RFC3339Time `json:"changeDate,omitempty"`
	ID             string       `json:"id"`
	UserID         string       `json:"userId"`
	OrganizationID string       `json:"organizationId,omitempty"`
	ExpirationDate *RFC3339Time `json:"expirationDate,omitempty"`
}

// RemovePersonalAccessTokenRequest represents the request for RemovePersonalAccessToken API
type RemovePersonalAccessTokenRequest struct {
	UserID  string `json:"userId"`
	TokenID string `json:"tokenId"`
}

// RemovePersonalAccessTokenResponse represents the response from RemovePersonalAccessToken API
type RemovePersonalAccessTokenResponse struct {
	DeletionDate *RFC3339Time `json:"deletionDate,omitempty"`
}
