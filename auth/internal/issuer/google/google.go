package google

import (
	"context"
	"regexp"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/totvs/go-sdk/auth/internal/issuer"
)

type googleIssuer struct {
	issuer.IssuerBase
}

type googleClaims struct {
	issuer.ClaimsBase
}

func NewGoogle(jwks_url string) issuer.Issuer {
	var g googleIssuer
	g.Ctx = context.TODO()
	g.IssuerRegex = regexp.MustCompile(`(?m)^https://accounts\.google\.com$`)
	g.Jwks_url = jwks_url
	g.Verifier = oidc.NewVerifier("",
		oidc.NewRemoteKeySet(g.Ctx, g.Jwks_url),
		&oidc.Config{
			InsecureSkipSignatureCheck: false,
			SkipExpiryCheck:            false,
			SkipClientIDCheck:          true,
			SkipIssuerCheck:            true, // Não verifica pois isso é feito via regex
		})

	return &g
}

func (r googleIssuer) Claims(payload []byte) (issuer.Claims, error) {
	var claims googleClaims
	err := r.IssuerBase.ClaimsBase(payload, &claims)
	return claims, err
}
