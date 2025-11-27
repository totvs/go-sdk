package rac

import (
	"context"
	"regexp"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/totvs/go-sdk/issuer"
)

type racIssuer struct {
	issuer.IssuerBase
}

type racClaims struct {
	issuer.ClaimsBase
	TenantIdpID string `json:"http://www.tnf.com/identity/claims/tenantId"`
}

func NewRac(jwks_url string) issuer.Issuer {
	var r racIssuer
	r.Ctx = context.TODO()
	r.IssuerRegex = regexp.MustCompile(`(?m)^https://.+\.rac\..*totvs\.app/totvs\.rac$`)
	r.Jwks_url = jwks_url
	r.Verifier = oidc.NewVerifier("",
		oidc.NewRemoteKeySet(r.Ctx, r.Jwks_url),
		&oidc.Config{
			InsecureSkipSignatureCheck: false,
			SkipExpiryCheck:            false,
			SkipClientIDCheck:          true,
			SkipIssuerCheck:            true, // Não verifica pois isso é feito via regex
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
