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

func NewIdentity(jwksURL string) issuer.Issuer {
	var i identityIssuer
	i.Ctx = context.Background()
	i.IssuerRegex = regexp.MustCompile(`(?m)^\*\.fluig\.io$`)
	i.Jwks_url = jwksURL
	i.Verifier = oidc.NewVerifier("",
		oidc.NewRemoteKeySet(i.Ctx, i.Jwks_url),
		&oidc.Config{
			InsecureSkipSignatureCheck: false,
			SkipExpiryCheck:            false,
			SkipClientIDCheck:          true,
			SkipIssuerCheck:            true, // Não verifica pois isso é feito via regex
		})

	return &i
}

func (r identityIssuer) Claims(payload []byte) (issuer.Claims, error) {
	var claims identityClaims
	err := r.IssuerBase.ClaimsBase(payload, &claims)
	return claims, err
}
