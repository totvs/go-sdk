package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/totvs/go-sdk/auth/internal/issuer"
)

type IssuerClaimsKey string

const ISSUER_CLAIMS_KEY IssuerClaimsKey = "issuer-claims"

// HTTPAuthorizationBearerTokenMiddleware is a middleware that validates the bearer token in the request header and adds the issuer claims to the request context.
func HTTPAuthorizationBearerTokenMiddleware(authMiddleware *issuer.AuthorizationBearerToken) func(http.Handler) http.Handler {
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
