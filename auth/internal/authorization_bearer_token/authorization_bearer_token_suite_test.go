package authorization_bearer_token_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/totvs/go-sdk/auth"
	"github.com/totvs/go-sdk/auth/issuer/google"
	"github.com/totvs/go-sdk/auth/issuer/identity"
	"github.com/totvs/go-sdk/auth/issuer/rac"
)

func TestAuthorizationBearerToken(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Authorization Bearer Token Suite")
}

var _ = Describe("Test package authorization bearer token", func() {
	serveJWKS()

	urlTest := "http://localhost:4444/jwks"

	googleIssuer := google.NewGoogle(urlTest)
	identityIssuer := identity.NewIdentity(urlTest)
	racIssuer := rac.NewRac(urlTest)

	a := auth.NewAuthorizationBearerToken(googleIssuer, identityIssuer, racIssuer)

	urlDefault := &url.URL{
		Scheme: "http",
		Host:   "totvs.app",
		Path:   "/path/to/resource",
	}

	Context("Initialize", func() {
		It("should be a valid authorization bearer token instance", func() {
			Expect(a).ToNot(BeNil())
		})
	})

	Context("Valid bearer token", func() {
		It("Should be an invalid request", func() {
			request := &http.Request{
				Method: "GET",
				URL:    urlDefault,
				Header: map[string][]string{
					"Content-Type": {"application/json"},
				},
			}

			claims, err := a.IsValidBearerToken(request)
			Expect(err).ToNot(BeNil())
			Expect(claims).To(BeNil())
		})

		It("Should be a valid Identity issuer", func() {
			request := &http.Request{
				Method: "GET",
				URL:    urlDefault,
				Header: map[string][]string{
					"Content-Type": {"application/json"},
				},
			}

			claims := jwt.MapClaims{
				"iss":         "*.fluig.io",
				"sub":         "totvs@totvs.com.br",
				"aud":         "fluig_authenticator_resource",
				"exp":         time.Now().UTC().Add(time.Hour).Unix(),
				"iat":         time.Now().UTC().Unix(),
				"email":       "totvs@totvs.com.br",
				"client_id":   "manager",
				"tenantIdpId": "2d4c74cfac2e438b97f110b185530ecb",
				"companyId":   "2d4c74cfac2e438b97f110b185530ecb",
				"roles":       []string{"admin"},
				"fullName":    "John Doe",
			}

			jwt, _ := generateJWT(claims)
			request.Header["Authorization"] = []string{"Bearer " + jwt}

			issuerClaims, err := a.IsValidBearerToken(request)
			if err != nil {
				Fail("Failed to validate bearer token: " + err.Error())
			}

			Expect(issuerClaims.ClaimAudience() == "fluig_authenticator_resource").To(BeTrue())
			Expect(issuerClaims.ClaimIssuer() == "*.fluig.io").To(BeTrue())
			Expect(issuerClaims.ClaimTenantIdpID() == "2d4c74cfac2e438b97f110b185530ecb").To(BeTrue())
			Expect(issuerClaims.ClaimCompanyID() == "2d4c74cfac2e438b97f110b185530ecb").To(BeTrue())
			Expect(issuerClaims.ClaimClientID() == "manager").To(BeTrue())
			Expect(issuerClaims.ClaimEmail() == "totvs@totvs.com.br").To(BeTrue())
			Expect(issuerClaims.ClaimRoles()).To(Equal([]string{"admin"}))
			Expect(issuerClaims.ClaimFullName() == "John Doe").To(BeTrue())
		})

		It("Should be a valid RAC issuer", func() {
			request := &http.Request{
				Method: "GET",
				URL:    urlDefault,
				Header: map[string][]string{
					"Content-Type": {"application/json"},
				},
			}

			claims := jwt.MapClaims{
				"iss":       "https://admin.rac.dev.totvs.app/totvs.rac",
				"sub":       "totvs@totvs.com.br",
				"aud":       "authorization_api",
				"exp":       time.Now().UTC().Add(time.Hour).Unix(),
				"iat":       time.Now().UTC().Unix(),
				"email":     "totvs@totvs.com.br",
				"client_id": "manager",
				"http://www.tnf.com/identity/claims/tenantId": "2d4c74cfac2e438b97f110b185530ecb",
			}

			jwt, _ := generateJWT(claims)
			request.Header["Authorization"] = []string{"Bearer " + jwt}

			issuerClaims, err := a.IsValidBearerToken(request)
			if err != nil {
				Fail("Failed to validate bearer token: " + err.Error())
			}

			Expect(issuerClaims.ClaimAudience() == "authorization_api").To(BeTrue())
			Expect(issuerClaims.ClaimIssuer() == "https://admin.rac.dev.totvs.app/totvs.rac").To(BeTrue())
			Expect(issuerClaims.ClaimTenantIdpID() == "2d4c74cfac2e438b97f110b185530ecb").To(BeTrue())
			Expect(issuerClaims.ClaimCompanyID() == "2d4c74cfac2e438b97f110b185530ecb").To(BeTrue())
			Expect(issuerClaims.ClaimClientID() == "manager").To(BeTrue())
			Expect(issuerClaims.ClaimEmail() == "totvs@totvs.com.br").To(BeTrue())
			Expect(issuerClaims.ClaimRoles()).To(BeEmpty())
		})
	})

	Context("JWT methods", func() {
		It("should returns error when JWT is malformed (expected 3 parts)", func() {
			request := &http.Request{
				Method: "GET",
				URL:    urlDefault,
				Header: map[string][]string{
					"Content-Type":  {"application/json"},
					"Authorization": {"Bearer abc"},
				},
			}

			_, err := a.IsValidBearerToken(request)
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("malformed jwt, expected 3 parts got %d", 1)))
		})
		It("should returns error when JWT is malformed (decode base 64 error)", func() {
			request := &http.Request{
				Method: "GET",
				URL:    urlDefault,
				Header: map[string][]string{
					"Content-Type":  {"application/json"},
					"Authorization": {"Bearer abc.11111.abc"},
				},
			}
			_, err := a.IsValidBearerToken(request)
			Expect(err.Error()).To(ContainSubstring("malformed jwt payload"))
		})
		It("should returns error when JWT issuer not found", func() {
			request := &http.Request{
				Method: "GET",
				URL:    urlDefault,
				Header: map[string][]string{
					"Content-Type":  {"application/json"},
					"Authorization": {"Bearer eyJhbGciOiJSUzI1NiJ9.eyJhdWQiOiJhYmMifQ.YWJj"},
				},
			}
			_, err := a.IsValidBearerToken(request)
			Expect(err.Error()).To(ContainSubstring("issuer not found"))
		})
	})
	Context("Header authorization methods", func() {
		It("Should returns error when authorization is malformed (expected 2 parts)", func() {
			request := &http.Request{
				Method: "GET",
				URL:    urlDefault,
				Header: map[string][]string{
					"Content-Type":  {"application/json"},
					"Authorization": {"YWJj"},
				},
			}
			_, err := a.IsValidBearerToken(request)
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("authorization header malformed (split size: %v)", 1)))
		})
		It("Should returns error when authorization is basic", func() {
			request := &http.Request{
				Method: "GET",
				URL:    urlDefault,
				Header: map[string][]string{
					"Content-Type":  {"application/json"},
					"Authorization": {"Basic " + "YWJj"},
				},
			}
			_, err := a.IsValidBearerToken(request)
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("invalid authorization header (accepts Bearer only | tokenType: %v)", "Basic")))
		})
		It("Should returns error when JWT is invalid", func() {
			request := &http.Request{
				Method: "GET",
				URL:    urlDefault,
				Header: map[string][]string{
					"Content-Type": {"application/json"},
				},
			}

			claims := jwt.MapClaims{
				"iss":         "*.fluig.io",
				"sub":         "totvs@totvs.com.br",
				"aud":         "fluig_authenticator_resource",
				"exp":         time.Now().UTC().Add(time.Hour).Unix(),
				"iat":         time.Now().UTC().Unix(),
				"email":       "totvs@totvs.com.br",
				"client_id":   "manager",
				"tenantIdpId": "2d4c74cfac2e438b97f110b185530ecb",
				"companyId":   "2d4c74cfac2e438b97f110b185530ecb",
			}

			jwt, _ := generateJWT(claims)
			jwt = jwt[:len(jwt)-1]
			request.Header["Authorization"] = []string{"Bearer " + jwt}
			_, err := a.IsValidBearerToken(request)
			Expect(err.Error()).To(ContainSubstring("failed to verify signature: failed to verify id token signature"))
		})
	})

})
