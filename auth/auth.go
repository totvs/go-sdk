package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"

	"github.com/totvs/go-sdk/auth/internal/authorization_bearer_token"
	"github.com/totvs/go-sdk/auth/issuer"
)

type IssuerClaimsKey string

const ISSUER_CLAIMS_KEY IssuerClaimsKey = "issuer-claims"

// HTTPAuthorizationBearerTokenMiddleware is a middleware that validates the bearer token in the request header and adds the issuer claims to the request context.
func HTTPAuthorizationBearerTokenMiddleware(authorizationBearerToken *authorization_bearer_token.AuthorizationBearerToken) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := authorizationBearerToken.IsValidBearerToken(r)
			if err != nil {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{
					"error": err.Error(),
				})
				return
			}

			ctx := context.WithValue(r.Context(), ISSUER_CLAIMS_KEY, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// NewAuthorizationBearerToken creates a new AuthorizationBearerToken with the given issuers.
func NewAuthorizationBearerToken(issuers ...issuer.Issuer) *authorization_bearer_token.AuthorizationBearerToken {
	return &authorization_bearer_token.AuthorizationBearerToken{
		Issuers: issuers,
	}
}

// GetIssuerClaimsFromContext is a convenience function that returns the issuer claims from the request context.
func GetIssuerClaimsFromContext(ctx context.Context) issuer.Claims {
	claims, ok := ctx.Value(ISSUER_CLAIMS_KEY).(issuer.Claims)
	if !ok {
		return nil
	}
	return claims
}

// HasRole checks if the user has the given role.
func HasRole(ctx context.Context, role string) bool {
	claims := GetIssuerClaimsFromContext(ctx)

	if claims == nil {
		return false
	}

	return slices.Contains(claims.ClaimRoles(), role)
}
