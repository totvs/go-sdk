// Package oauth2 provides a small, transport-agnostic OAuth 2.0 token client.
// HTTP handlers, cookies, sessions, and application routes belong to consumers.
package oauth2

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultHTTPTimeout     = 30 * time.Second
	defaultMaxResponseSize = int64(1 << 20)
)

var (
	ErrInvalidConfig  = errors.New("oauth2: invalid configuration")
	ErrInvalidRequest = errors.New("oauth2: invalid token request")
)

// Config contains the confidential-client settings required by a token
// endpoint. ClientSecret is never included in returned errors.
type Config struct {
	TokenEndpoint        string
	ClientID             string
	ClientSecret         string
	ClientAuthentication ClientAuthenticationMethod
}

// ClientAuthenticationMethod selects how a confidential client authenticates
// at the token endpoint.
type ClientAuthenticationMethod string

const (
	ClientSecretBasic ClientAuthenticationMethod = "client_secret_basic"
	ClientSecretPost  ClientAuthenticationMethod = "client_secret_post"
)

// AuthorizationCodeRequest is the input for the authorization-code grant.
// When provided, RedirectURI must be exactly the URI used by the authorization
// request.
type AuthorizationCodeRequest struct {
	Code         string
	RedirectURI  string
	CodeVerifier string
}

// Tokens is the normalized token-endpoint response.
type Tokens struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	IDToken      string    `json:"id_token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
}

// EndpointError describes a rejected upstream token request without exposing
// the provider response body or OAuth tokens.
type EndpointError struct {
	StatusCode int
	Code       string
}

func (e *EndpointError) Error() string {
	if e.Code == "" {
		return fmt.Sprintf("oauth2: token endpoint returned status %d", e.StatusCode)
	}
	return fmt.Sprintf("oauth2: token endpoint returned status %d (%s)", e.StatusCode, e.Code)
}

// Client exchanges authorization codes and refresh tokens for a confidential
// OAuth client using client_secret_basic authentication.
type Client struct {
	config                   Config
	httpClient               *http.Client
	now                      func() time.Time
	maxResponseBytes         int64
	refreshCredentialsInBody bool
}

// Option customizes a Client.
type Option func(*Client)

// WithHTTPClient injects the HTTP client used for token requests.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(client *Client) {
		if httpClient != nil {
			client.httpClient = httpClient
		}
	}
}

// WithMaxResponseBytes changes the token-response size limit when positive.
func WithMaxResponseBytes(maxBytes int64) Option {
	return func(client *Client) {
		if maxBytes > 0 {
			client.maxResponseBytes = maxBytes
		}
	}
}

// WithRefreshClientCredentialsInBody includes client_id and client_secret in
// refresh request bodies while retaining client_secret_basic authentication.
func WithRefreshClientCredentialsInBody() Option {
	return func(client *Client) {
		client.refreshCredentialsInBody = true
	}
}

// NewClient validates config and creates an OAuth token client.
func NewClient(config Config, options ...Option) (*Client, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	client := &Client{
		config:           withConfigDefaults(config),
		httpClient:       &http.Client{Timeout: defaultHTTPTimeout},
		now:              time.Now,
		maxResponseBytes: defaultMaxResponseSize,
	}
	for _, option := range options {
		option(client)
	}
	return client, nil
}

// ExchangeAuthorizationCode exchanges an authorization code for a token
// bundle. PKCE is supported when CodeVerifier is provided.
func (c *Client) ExchangeAuthorizationCode(ctx context.Context, input AuthorizationCodeRequest) (Tokens, error) {
	if strings.TrimSpace(input.Code) == "" {
		return Tokens{}, fmt.Errorf("%w: code is required", ErrInvalidRequest)
	}

	form := url.Values{
		"grant_type": {"authorization_code"},
		"code":       {input.Code},
	}
	if strings.TrimSpace(input.RedirectURI) != "" {
		if err := validateHTTPURL(input.RedirectURI); err != nil {
			return Tokens{}, fmt.Errorf("%w: redirect URI: %v", ErrInvalidRequest, err)
		}
		form.Set("redirect_uri", input.RedirectURI)
	}
	if input.CodeVerifier != "" {
		form.Set("code_verifier", input.CodeVerifier)
	}
	return c.requestTokens(ctx, form)
}

// Refresh exchanges a refresh token for a new token bundle. Providers may
// rotate refresh tokens; when they do not, the previous value is preserved.
func (c *Client) Refresh(ctx context.Context, refreshToken string) (Tokens, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return Tokens{}, fmt.Errorf("%w: refresh token is required", ErrInvalidRequest)
	}

	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}
	if c.config.ClientAuthentication == ClientSecretBasic && c.refreshCredentialsInBody {
		form.Set("client_id", c.config.ClientID)
		form.Set("client_secret", c.config.ClientSecret)
	}
	tokens, err := c.requestTokens(ctx, form)
	if err != nil {
		return Tokens{}, err
	}
	if tokens.RefreshToken == "" {
		tokens.RefreshToken = refreshToken
	}
	return tokens, nil
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int64  `json:"expires_in"`
}

type endpointErrorResponse struct {
	Code string `json:"error"`
}

func (c *Client) requestTokens(ctx context.Context, form url.Values) (Tokens, error) {
	if c.config.ClientAuthentication == ClientSecretPost {
		form.Set("client_id", c.config.ClientID)
		form.Set("client_secret", c.config.ClientSecret)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return Tokens{}, fmt.Errorf("oauth2: create token request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if c.config.ClientAuthentication == ClientSecretBasic {
		req.Header.Set("Authorization", basicAuthHeader(c.config.ClientID, c.config.ClientSecret))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Tokens{}, fmt.Errorf("oauth2: call token endpoint: %w", err)
	}
	defer resp.Body.Close()

	body, err := readBounded(resp.Body, c.maxResponseBytes)
	if err != nil {
		return Tokens{}, fmt.Errorf("oauth2: read token response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		var upstream endpointErrorResponse
		_ = json.Unmarshal(body, &upstream)
		return Tokens{}, &EndpointError{StatusCode: resp.StatusCode, Code: sanitizeErrorCode(upstream.Code)}
	}

	var upstream tokenResponse
	if err := json.Unmarshal(body, &upstream); err != nil {
		return Tokens{}, errors.New("oauth2: token endpoint returned invalid JSON")
	}
	if strings.TrimSpace(upstream.AccessToken) == "" {
		return Tokens{}, errors.New("oauth2: token endpoint response has no access token")
	}

	tokens := Tokens{
		AccessToken:  upstream.AccessToken,
		RefreshToken: upstream.RefreshToken,
		IDToken:      upstream.IDToken,
		TokenType:    upstream.TokenType,
		Scope:        upstream.Scope,
	}
	if upstream.ExpiresIn > 0 {
		tokens.ExpiresAt = c.now().Add(time.Duration(upstream.ExpiresIn) * time.Second)
	}
	return tokens, nil
}

func validateConfig(config Config) error {
	if strings.TrimSpace(config.TokenEndpoint) == "" || strings.TrimSpace(config.ClientID) == "" || config.ClientSecret == "" {
		return fmt.Errorf("%w: token endpoint, client ID, and client secret are required", ErrInvalidConfig)
	}
	if err := validateHTTPURL(config.TokenEndpoint); err != nil {
		return fmt.Errorf("%w: token endpoint: %v", ErrInvalidConfig, err)
	}
	method := withConfigDefaults(config).ClientAuthentication
	if method != ClientSecretBasic && method != ClientSecretPost {
		return fmt.Errorf("%w: unsupported client authentication method %q", ErrInvalidConfig, method)
	}
	return nil
}

func withConfigDefaults(config Config) Config {
	if config.ClientAuthentication == "" {
		config.ClientAuthentication = ClientSecretBasic
	}
	return config
}

func validateHTTPURL(raw string) error {
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return errors.New("must be an absolute HTTP(S) URL")
	}
	return nil
}

func sanitizeErrorCode(raw string) string {
	if raw == "" || len(raw) > 64 {
		return ""
	}
	for _, char := range raw {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') &&
			(char < '0' || char > '9') && char != '_' && char != '-' && char != '.' {
			return ""
		}
	}
	return raw
}

func basicAuthHeader(clientID, clientSecret string) string {
	credentials := url.QueryEscape(clientID) + ":" + url.QueryEscape(clientSecret)
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(credentials))
}

func readBounded(reader io.Reader, maxBytes int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxBytes {
		return nil, errors.New("response exceeds configured size limit")
	}
	return body, nil
}
