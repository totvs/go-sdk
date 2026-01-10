package identity

import (
	"context"
	"regexp"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/totvs/go-sdk/auth/issuer"
)

type identityIssuer struct {
	issuer.IssuerBase
}

type identityClaims struct {
	issuer.ClaimsBase
}

// NewIdentity creates a new Fluig Identity issuer that validates tokens against the provided JWKS URL.
func NewIdentity(jwksURL string) issuer.Issuer {
	var i identityIssuer
	i.IssuerRegex = regexp.MustCompile(`(?m)^\*\.fluig\.io$`)
	i.JwksURL = jwksURL
	i.Verifier = oidc.NewVerifier("",
		oidc.NewRemoteKeySet(context.Background(), jwksURL),
		&oidc.Config{
			InsecureSkipSignatureCheck: false,
			SkipExpiryCheck:            false,
			SkipClientIDCheck:          true,
			SkipIssuerCheck:            true, // Issuer is validated via regex
		})

	return &i
}

func (r identityIssuer) Claims(payload []byte) (issuer.Claims, error) {
	var claims identityClaims
	err := r.IssuerBase.ClaimsBase(payload, &claims)
	return claims, err
}
