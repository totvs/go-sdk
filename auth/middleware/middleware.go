package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/totvs/go-sdk/auth/internal/issuer"
	"github.com/totvs/go-sdk/auth/internal/issuer/google"
	"github.com/totvs/go-sdk/auth/internal/issuer/identity"
	"github.com/totvs/go-sdk/auth/internal/issuer/rac"
)

type IssuerClaimsKey string

const ISSUER_CLAIMS_KEY IssuerClaimsKey = "issuer-claims"

type authorizationBearerTokenMiddleware struct {
	Issuers []issuer.Issuer
}

func NewAuthorizationBearerTokenMiddleware(jwksIdentity, jwksRac, jwksGoogle string) *authorizationBearerTokenMiddleware {
	return &authorizationBearerTokenMiddleware{
		Issuers: []issuer.Issuer{
			identity.NewIdentity(jwksIdentity),
			rac.NewRac(jwksRac),
			google.NewGoogle(jwksGoogle),
		},
	}
}

func (a *authorizationBearerTokenMiddleware) ValidBearerToken(r *http.Request) (issuer.Claims, error) {
	authorization := r.Header.Get("Authorization")

	if authorization != "" {
		token, err := a.extractTokenFromBearer(authorization)
		if err != nil {
			return nil, err
		}

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

		issuer, err := a.validJWT(i.Issuer, token)
		if err != nil {
			return nil, fmt.Errorf("failed to verify JWT: %v", err.Error())
		}

		return issuer.Claims(payload)
	}

	return nil, fmt.Errorf("header authorization not found")
}

func (a *authorizationBearerTokenMiddleware) findIssuer(issuerClaim string) (issuer.Issuer, error) {
	for _, i := range a.Issuers {
		if i.MatchIssuer(issuerClaim) {
			return i, nil
		}
	}
	return nil, fmt.Errorf("issuer not found")
}

func (a *authorizationBearerTokenMiddleware) validJWT(issuerClaim string, rawToken string) (issuer.Issuer, error) {
	issuer, err := a.findIssuer(issuerClaim)
	if err != nil {
		return nil, fmt.Errorf("failed to find issuer: %v", err)
	}

	_, err = issuer.Verify(rawToken)
	if err != nil {
		return nil, err
	}
	return issuer, nil
}

func (a *authorizationBearerTokenMiddleware) parseJWT(jwt string) ([]byte, error) {
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

func (a *authorizationBearerTokenMiddleware) extractTokenFromBearer(authorization string) (string, error) {
	tokenType, token, err := a.splitAuthHeader(authorization)
	if err != nil {
		return "", err
	}
	if tokenType != "Bearer" {
		return "", fmt.Errorf("invalid authorization header (accepts Bearer only | tokenType: %v)", tokenType)
	}
	return token, nil
}

func (a *authorizationBearerTokenMiddleware) splitAuthHeader(header string) (string, string, error) {
	s := strings.Split(header, " ")
	if len(s) != 2 {
		return "", "", fmt.Errorf("authorization header malformed (split size: %v)", len(s))
	}
	return s[0], s[1], nil
}

func HTTPAuthorizationBearerTokenMiddleware(authMiddleware authorizationBearerTokenMiddleware) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := authMiddleware.ValidBearerToken(r)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				fmt.Fprintf(w, "{\"error\": \"%v\"}", err.Error())
				return
			}

			ctx := context.WithValue(r.Context(), ISSUER_CLAIMS_KEY, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetIssuerClaimsFromContext(ctx context.Context) issuer.Claims {
	claims := ctx.Value(ISSUER_CLAIMS_KEY).(issuer.Claims)
	return claims
}
