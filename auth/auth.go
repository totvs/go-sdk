package auth

import (
	"context"
	"slices"

	"github.com/totvs/go-sdk/auth/internal/issuer"
	"github.com/totvs/go-sdk/auth/internal/issuer/google"
	"github.com/totvs/go-sdk/auth/internal/issuer/identity"
	"github.com/totvs/go-sdk/auth/internal/issuer/rac"
	"github.com/totvs/go-sdk/auth/middleware"
)

// NewAuthorizationBearerToken creates a new AuthorizationBearerToken with the given JWKS URLs for the identity, rac, and google issuers.
func NewAuthorizationBearerToken(jwksIdentity, jwksRac, jwksGoogle string) *issuer.AuthorizationBearerToken {
	return &issuer.AuthorizationBearerToken{
		Issuers: []issuer.Issuer{
			identity.NewIdentity(jwksIdentity),
			rac.NewRac(jwksRac),
			google.NewGoogle(jwksGoogle),
		},
	}
}

// GetIssuerClaimsFromContext is a convenience function that returns the issuer claims from the request context.
func GetIssuerClaimsFromContext(ctx context.Context) issuer.Claims {
	claims, ok := ctx.Value(middleware.ISSUER_CLAIMS_KEY).(issuer.Claims)
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
