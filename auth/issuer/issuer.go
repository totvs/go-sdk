// Package issuer provides interfaces and base implementations for JWT/OIDC token validation.
// It supports multiple issuers (Google, Fluig Identity, TOTVS RAC) with a common interface.
package issuer

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/coreos/go-oidc/v3/oidc"
)

// Issuer represents a JWT token issuer capable of verifying tokens.
type Issuer interface {
	// MatchIssuer returns true if the given issuer string matches this issuer's pattern.
	MatchIssuer(iss string) bool
	// Verify validates the token and returns the parsed IDToken.
	// The context is used for cancellation and timeout control.
	Verify(ctx context.Context, token string) (*oidc.IDToken, error)
	// Claims parses the JWT payload and returns typed claims.
	Claims(payload []byte) (Claims, error)
}

// Claims represents the standard claims extracted from a JWT token.
type Claims interface {
	ClaimRoles() []string
	ClaimFullName() string
	ClaimEmail() string
	ClaimTenantIdpID() string
	ClaimCompanyID() string
	ClaimClientID() string
	ClaimAudience() string
	ClaimIssuer() string
}

// IssuerBase provides common functionality for all issuer implementations.
// Embed this struct in concrete issuer types to inherit MatchIssuer and Verify methods.
type IssuerBase struct {
	IssuerRegex *regexp.Regexp
	JwksURL     string
	Verifier    *oidc.IDTokenVerifier
}

// ClaimsBase provides the base implementation for JWT claims.
// Embed this struct in concrete claims types to inherit getter methods.
type ClaimsBase struct {
	FullName    string   `json:"fullName,omitempty"`
	NotBefore   int64    `json:"nbf,omitempty"`
	ExpiresAt   int64    `json:"exp,omitempty"`
	Issuer      string   `json:"iss,omitempty"`
	Audience    string   `json:"aud,omitempty"`
	Subject     string   `json:"sub,omitempty"`
	IssuedAt    int64    `json:"iat,omitempty"`
	ClientID    string   `json:"client_id,omitempty"`
	TenantIdpID string   `json:"tenantIdpId,omitempty"`
	CompanyID   string   `json:"companyId,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	Email       string   `json:"email"`
}

// MatchIssuer returns true if the issuer string matches this issuer's regex pattern.
func (r IssuerBase) MatchIssuer(iss string) bool {
	return r.IssuerRegex.MatchString(iss)
}

// Verify validates the token using the configured OIDC verifier.
// The context should be used for cancellation and timeout control.
func (r IssuerBase) Verify(ctx context.Context, token string) (*oidc.IDToken, error) {
	return r.Verifier.Verify(ctx, token)
}

// ClaimsBase unmarshals JSON payload into the provided claims struct.
func (r IssuerBase) ClaimsBase(payload []byte, claims any) error {
	if err := json.Unmarshal(payload, &claims); err != nil {
		return fmt.Errorf("JWT: failed to unmarshal claims: %w", err)
	}
	return nil
}

// ClaimRoles returns the user's roles or an empty slice if none.
func (i ClaimsBase) ClaimRoles() []string {
	if i.Roles == nil {
		return []string{}
	}
	return i.Roles
}

// ClaimFullName returns the user's full name or "-" if empty.
func (i ClaimsBase) ClaimFullName() string {
	if i.FullName == "" {
		return "-"
	}
	return i.FullName
}

// ClaimEmail returns the user's email or "-" if empty.
func (i ClaimsBase) ClaimEmail() string {
	if i.Email == "" {
		return "-"
	}
	return i.Email
}

// ClaimTenantIdpID returns the tenant IDP ID or "-" if empty.
func (i ClaimsBase) ClaimTenantIdpID() string {
	if i.TenantIdpID == "" {
		return "-"
	}
	return i.TenantIdpID
}

// ClaimCompanyID returns the company ID or "-" if empty.
func (i ClaimsBase) ClaimCompanyID() string {
	if i.CompanyID == "" {
		return "-"
	}
	return i.CompanyID
}

// ClaimClientID returns the client ID or "-" if empty.
func (i ClaimsBase) ClaimClientID() string {
	if i.ClientID == "" {
		return "-"
	}
	return i.ClientID
}

// ClaimAudience returns the audience or "-" if empty.
func (i ClaimsBase) ClaimAudience() string {
	if i.Audience == "" {
		return "-"
	}
	return i.Audience
}

// ClaimIssuer returns the issuer or "-" if empty.
func (i ClaimsBase) ClaimIssuer() string {
	if i.Issuer == "" {
		return "-"
	}
	return i.Issuer
}
