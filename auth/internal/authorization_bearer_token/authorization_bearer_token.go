// Package authorization_bearer_token provides JWT bearer token validation for HTTP requests.
package authorization_bearer_token

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/totvs/go-sdk/auth/issuer"
)

type AuthorizationBearerToken struct {
	Issuers []issuer.Issuer
}

func (a *AuthorizationBearerToken) IsValidBearerToken(r *http.Request) (issuer.Claims, error) {
	var token, authorization string
	var err error

	authorization = r.Header.Get("Authorization")

	if authorization != "" {
		token, err = a.extractTokenFromBearer(authorization)
		if err != nil {
			return nil, err
		}
	} else {
		token, err = a.extractTokenFromCookie(r)
		if err != nil {
			return nil, err
		}
	}

	if token != "" {

		payload, err := a.parseJWT(token)
		if err != nil {
			return nil, fmt.Errorf("malformed jwt: %v", err.Error())
		}

		i := struct {
			Issuer string `json:"iss,omitempty"`
		}{}
		if err := json.Unmarshal(payload, &i); err != nil {
			return nil, fmt.Errorf("oidc: failed to unmarshal claim issuer only: %v", err)
		}

		iss, err := a.validJWT(r.Context(), i.Issuer, token)
		if err != nil {
			return nil, fmt.Errorf("failed to verify JWT: %w", err)
		}

		return iss.Claims(payload)
	}

	return nil, fmt.Errorf("authorization token not found")
}

func (a *AuthorizationBearerToken) findIssuer(issuerClaim string) (issuer.Issuer, error) {
	for _, i := range a.Issuers {
		if i.MatchIssuer(issuerClaim) {
			return i, nil
		}
	}
	return nil, fmt.Errorf("issuer not found")
}

func (a *AuthorizationBearerToken) validJWT(ctx context.Context, issuerClaim string, rawToken string) (issuer.Issuer, error) {
	issuer, err := a.findIssuer(issuerClaim)
	if err != nil {
		return nil, fmt.Errorf("failed to find issuer: %w", err)
	}

	_, err = issuer.Verify(ctx, rawToken)
	if err != nil {
		return nil, err
	}
	return issuer, nil
}

func (a *AuthorizationBearerToken) parseJWT(jwt string) ([]byte, error) {
	parts := strings.Split(jwt, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("malformed jwt, expected 3 parts got %d", len(parts))
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("malformed jwt payload: %v", err)
	}
	return payload, nil
}

func (a *AuthorizationBearerToken) extractTokenFromBearer(authorization string) (string, error) {
	tokenType, token, err := a.splitAuthHeader(authorization)
	if err != nil {
		return "", err
	}
	if tokenType != "Bearer" {
		return "", fmt.Errorf("invalid authorization header (accepts Bearer only | tokenType: %v)", tokenType)
	}
	return token, nil
}

func (a *AuthorizationBearerToken) splitAuthHeader(header string) (string, string, error) {
	s := strings.Split(header, " ")
	if len(s) != 2 {
		return "", "", fmt.Errorf("authorization header malformed (split size: %v)", len(s))
	}
	return s[0], s[1], nil
}

func (a *AuthorizationBearerToken) extractTokenFromCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie("jwt.token")
	if err != nil {
		if err == http.ErrNoCookie {
			return "", nil
		}

		return "", fmt.Errorf("failed to extract token from cookie: %v", err)
	}
	return cookie.Value, nil
}
