package rac

import (
	"context"
	"regexp"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/totvs/go-sdk/auth/issuer"
)

type racIssuer struct {
	issuer.IssuerBase
}

type racClaims struct {
	issuer.ClaimsBase
	TenantIdpID string `json:"http://www.tnf.com/identity/claims/tenantId"`
}

// NewRac creates a new TOTVS RAC issuer that validates tokens against the provided JWKS URL.
func NewRac(jwksURL string) issuer.Issuer {
	var r racIssuer
	r.IssuerRegex = regexp.MustCompile(`(?m)^https://.+\.rac\..*totvs\.app/totvs\.rac$`)
	r.JwksURL = jwksURL
	r.Verifier = oidc.NewVerifier("",
		oidc.NewRemoteKeySet(context.Background(), jwksURL),
		&oidc.Config{
			InsecureSkipSignatureCheck: false,
			SkipExpiryCheck:            false,
			SkipClientIDCheck:          true,
			SkipIssuerCheck:            true, // Issuer is validated via regex
		})

	return &r
}

func (r racIssuer) Claims(payload []byte) (issuer.Claims, error) {
	var claims racClaims
	err := r.IssuerBase.ClaimsBase(payload, &claims)
	return claims, err
}

func (i racClaims) ClaimTenantIdpID() string {
	if i.TenantIdpID == "" {
		return "-"
	}
	return i.TenantIdpID
}

func (i racClaims) ClaimCompanyID() string {
	if i.TenantIdpID == "" {
		return "-"
	}
	return i.TenantIdpID
}
