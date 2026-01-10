package google

import (
	"context"
	"regexp"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/totvs/go-sdk/auth/issuer"
)

type googleIssuer struct {
	issuer.IssuerBase
}

type googleClaims struct {
	issuer.ClaimsBase
}

// NewGoogle creates a new Google OAuth issuer that validates tokens against the provided JWKS URL.
func NewGoogle(jwksURL string) issuer.Issuer {
	var g googleIssuer
	g.IssuerRegex = regexp.MustCompile(`(?m)^https://accounts\.google\.com$`)
	g.JwksURL = jwksURL
	g.Verifier = oidc.NewVerifier("",
		oidc.NewRemoteKeySet(context.Background(), jwksURL),
		&oidc.Config{
			InsecureSkipSignatureCheck: false,
			SkipExpiryCheck:            false,
			SkipClientIDCheck:          true,
			SkipIssuerCheck:            true, // Issuer is validated via regex
		})

	return &g
}

func (r googleIssuer) Claims(payload []byte) (issuer.Claims, error) {
	var claims googleClaims
	err := r.IssuerBase.ClaimsBase(payload, &claims)
	return claims, err
}
