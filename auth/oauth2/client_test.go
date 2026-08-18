package oauth2_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	sdkoauth "github.com/totvs/go-sdk/auth/oauth2"
)

func TestExchangeAuthorizationCode(t *testing.T) {
	var seenForm url.Values
	var seenAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuthorization = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		seenForm, _ = url.ParseQuery(string(body))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "access-token",
			"refresh_token": "refresh-token",
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	}))
	defer server.Close()

	client := newClient(t, server.URL)
	tokens, err := client.ExchangeAuthorizationCode(context.Background(), sdkoauth.AuthorizationCodeRequest{
		Code:         "authorization-code",
		RedirectURI:  "https://smart-harness.example/oauth/login",
		CodeVerifier: "verifier",
	})
	if err != nil {
		t.Fatalf("exchange authorization code: %v", err)
	}

	if tokens.AccessToken != "access-token" || tokens.RefreshToken != "refresh-token" {
		t.Fatalf("unexpected tokens: %#v", tokens)
	}
	if seenForm.Get("grant_type") != "authorization_code" || seenForm.Get("code") != "authorization-code" {
		t.Fatalf("unexpected form: %v", seenForm)
	}
	if seenForm.Get("redirect_uri") != "https://smart-harness.example/oauth/login" || seenForm.Get("code_verifier") != "verifier" {
		t.Fatalf("missing redirect URI or PKCE verifier: %v", seenForm)
	}
	if seenForm.Has("client_id") || seenForm.Has("client_secret") {
		t.Fatalf("basic credentials must not be sent in form: %v", seenForm)
	}
	if got := decodeBasicAuth(t, seenAuthorization); got != "client+id:client+secret" {
		t.Fatalf("basic credentials = %q", got)
	}
}

func TestExchangeAuthorizationCodeWithoutRedirectURIUsesMinimalBasicBody(t *testing.T) {
	var seenBody string
	var seenAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuthorization = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		seenBody = string(body)
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "access-token"})
	}))
	defer server.Close()

	client := newClient(t, server.URL)
	if _, err := client.ExchangeAuthorizationCode(context.Background(), sdkoauth.AuthorizationCodeRequest{Code: "authorization-code"}); err != nil {
		t.Fatalf("exchange authorization code: %v", err)
	}
	if seenBody != "code=authorization-code&grant_type=authorization_code" {
		t.Fatalf("request body = %q", seenBody)
	}
	if got := decodeBasicAuth(t, seenAuthorization); got != "client+id:client+secret" {
		t.Fatalf("basic credentials = %q", got)
	}
}

func TestClientSecretPostSendsCredentialsInForm(t *testing.T) {
	var seenForm url.Values
	var seenAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuthorization = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		seenForm, _ = url.ParseQuery(string(body))
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "access-token"})
	}))
	defer server.Close()

	client, err := sdkoauth.NewClient(sdkoauth.Config{
		TokenEndpoint:        server.URL,
		ClientID:             "post-client",
		ClientSecret:         "post-secret",
		ClientAuthentication: sdkoauth.ClientSecretPost,
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if _, err := client.Refresh(context.Background(), "refresh-token"); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if seenAuthorization != "" {
		t.Fatalf("unexpected Authorization header: %q", seenAuthorization)
	}
	if seenForm.Get("client_id") != "post-client" || seenForm.Get("client_secret") != "post-secret" {
		t.Fatalf("credentials missing from form: %v", seenForm)
	}
}

func TestRefreshPreservesUnrotatedRefreshToken(t *testing.T) {
	var seenBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		seenBody = string(body)
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "new-access-token"})
	}))
	defer server.Close()

	tokens, err := newClient(t, server.URL).Refresh(context.Background(), "current-refresh-token")
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if tokens.RefreshToken != "current-refresh-token" {
		t.Fatalf("refresh token = %q", tokens.RefreshToken)
	}
	if seenBody != "grant_type=refresh_token&refresh_token=current-refresh-token" {
		t.Fatalf("request body = %q", seenBody)
	}
}

func TestRefreshCanSendBasicCredentialsInBody(t *testing.T) {
	var seenBody string
	var seenAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuthorization = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		seenBody = string(body)
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "new-access-token"})
	}))
	defer server.Close()

	client, err := sdkoauth.NewClient(
		sdkoauth.Config{TokenEndpoint: server.URL, ClientID: "client id", ClientSecret: "client secret"},
		sdkoauth.WithRefreshClientCredentialsInBody(),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if _, err := client.Refresh(context.Background(), "refresh-token"); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if seenBody != "client_id=client+id&client_secret=client+secret&grant_type=refresh_token&refresh_token=refresh-token" {
		t.Fatalf("request body = %q", seenBody)
	}
	if got := decodeBasicAuth(t, seenAuthorization); got != "client+id:client+secret" {
		t.Fatalf("basic credentials = %q", got)
	}
}

func TestEndpointErrorDoesNotLeakResponseDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"secret upstream detail"}`))
	}))
	defer server.Close()

	_, err := newClient(t, server.URL).Refresh(context.Background(), "rejected-refresh-token")
	if err == nil {
		t.Fatal("expected refresh failure")
	}
	var endpointError *sdkoauth.EndpointError
	if !errors.As(err, &endpointError) || endpointError.StatusCode != http.StatusUnauthorized || endpointError.Code != "invalid_grant" {
		t.Fatalf("unexpected error: %T %v", err, err)
	}
	if strings.Contains(err.Error(), "secret upstream detail") || strings.Contains(err.Error(), "rejected-refresh-token") {
		t.Fatalf("error leaked sensitive data: %v", err)
	}
}

func TestEndpointErrorRejectsUnsafeProviderCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"token value must not propagate"}`))
	}))
	defer server.Close()

	_, err := newClient(t, server.URL).Refresh(context.Background(), "refresh-token")
	var endpointError *sdkoauth.EndpointError
	if !errors.As(err, &endpointError) || endpointError.Code != "" {
		t.Fatalf("unsafe provider code was not discarded: %v", err)
	}
}

func TestResponseSizeIsBounded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"too-large"}`))
	}))
	defer server.Close()

	client, err := sdkoauth.NewClient(
		sdkoauth.Config{TokenEndpoint: server.URL, ClientID: "client", ClientSecret: "secret"},
		sdkoauth.WithMaxResponseBytes(8),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if _, err := client.Refresh(context.Background(), "refresh-token"); err == nil || !strings.Contains(err.Error(), "size limit") {
		t.Fatalf("expected bounded response error, got %v", err)
	}
}

func TestNewClientRejectsUnsafeConfiguration(t *testing.T) {
	tests := []sdkoauth.Config{
		{},
		{TokenEndpoint: "file:///tmp/token", ClientID: "client", ClientSecret: "secret"},
		{TokenEndpoint: "https://user:password@identity.example/token", ClientID: "client", ClientSecret: "secret"},
		{TokenEndpoint: "https://identity.example/token", ClientSecret: "secret"},
		{TokenEndpoint: "https://identity.example/token", ClientID: "client", ClientSecret: "secret", ClientAuthentication: "private_key_jwt"},
	}
	for _, config := range tests {
		if _, err := sdkoauth.NewClient(config); !errors.Is(err, sdkoauth.ErrInvalidConfig) {
			t.Fatalf("config %#v: expected invalid config, got %v", config, err)
		}
	}
}

func TestRequestsRequireGrantInputs(t *testing.T) {
	client := newClient(t, "https://identity.example/token")
	if _, err := client.ExchangeAuthorizationCode(context.Background(), sdkoauth.AuthorizationCodeRequest{}); !errors.Is(err, sdkoauth.ErrInvalidRequest) {
		t.Fatalf("expected invalid authorization request, got %v", err)
	}
	if _, err := client.Refresh(context.Background(), ""); !errors.Is(err, sdkoauth.ErrInvalidRequest) {
		t.Fatalf("expected invalid refresh request, got %v", err)
	}
}

func TestExchangeAuthorizationCodeRejectsInvalidRedirectURI(t *testing.T) {
	client := newClient(t, "https://identity.example/token")
	_, err := client.ExchangeAuthorizationCode(context.Background(), sdkoauth.AuthorizationCodeRequest{
		Code:        "authorization-code",
		RedirectURI: "file:///tmp/callback",
	})
	if !errors.Is(err, sdkoauth.ErrInvalidRequest) {
		t.Fatalf("expected invalid authorization request, got %v", err)
	}
}

func newClient(t *testing.T, endpoint string) *sdkoauth.Client {
	t.Helper()
	client, err := sdkoauth.NewClient(sdkoauth.Config{
		TokenEndpoint: endpoint,
		ClientID:      "client id",
		ClientSecret:  "client secret",
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	return client
}

func decodeBasicAuth(t *testing.T, header string) string {
	t.Helper()
	encoded := strings.TrimPrefix(header, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode basic auth: %v", err)
	}
	return string(decoded)
}
