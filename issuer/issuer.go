package issuer

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/coreos/go-oidc/v3/oidc"
)

type Issuer interface {
	MatchIssuer(string) bool
	Verify(string) (*oidc.IDToken, error)
	Claims([]byte) (Claims, error)
}

type Claims interface {
	ClaimEmail() string
	ClaimTenantIdpID() string
	ClaimCompanyID() string
	ClaimClientID() string
	ClaimAudience() string
	ClaimIssuer() string
}

type IssuerBase struct {
	Ctx         context.Context
	IssuerRegex *regexp.Regexp
	Jwks_url    string
	Verifier    *oidc.IDTokenVerifier
}

type ClaimsBase struct {
	NotBefore   int64  `json:"nbf,omitempty"`
	ExpiresAt   int64  `json:"exp,omitempty"`
	Issuer      string `json:"iss,omitempty"`
	Audience    string `json:"aud,omitempty"`
	Subject     string `json:"sub,omitempty"`
	IssuedAt    int64  `json:"iat,omitempty"`
	ClientID    string `json:"client_id,omitempty"`
	TenantIdpID string `json:"tenantIdpId,omitempty"`
	CompanyID   string `json:"companyId,omitempty"`
	Email       string `json:"email"`
}

func (r IssuerBase) MatchIssuer(iss string) bool {
	return r.IssuerRegex.MatchString(iss)
}

func (r IssuerBase) Verify(token string) (*oidc.IDToken, error) {
	return r.Verifier.Verify(r.Ctx, token)
}

func (r IssuerBase) ClaimsBase(payload []byte, claims any) error {
	if err := json.Unmarshal(payload, &claims); err != nil {
		return fmt.Errorf("JWT: failed to unmarshal claims: %v", err)
	}
	return nil
}

func (i ClaimsBase) ClaimEmail() string {
	if i.Email == "" {
		return "-"
	}
	return i.Email
}

func (i ClaimsBase) ClaimTenantIdpID() string {
	if i.TenantIdpID == "" {
		return "-"
	}
	return i.TenantIdpID
}

func (i ClaimsBase) ClaimCompanyID() string {
	if i.CompanyID == "" {
		return "-"
	}
	return i.CompanyID
}

func (i ClaimsBase) ClaimClientID() string {
	if i.ClientID == "" {
		return "-"
	}
	return i.ClientID
}

func (i ClaimsBase) ClaimAudience() string {
	if i.Audience == "" {
		return "-"
	}
	return i.Audience
}

func (i ClaimsBase) ClaimIssuer() string {
	if i.Issuer == "" {
		return "-"
	}
	return i.Issuer
}
