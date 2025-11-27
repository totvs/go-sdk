package issuer_test

import (
	"encoding/json"
	"regexp"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/totvs/go-sdk/issuer"
)

func TestIssuer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Issuer Suite")
}

var _ = Describe("Test issuer", func() {
	i := &issuer.IssuerBase{}
	i.IssuerRegex = regexp.MustCompile(`(?m)^https://.+\.rac\..*totvs\.app/totvs\.rac$`)
	Context("Initialize", func() {
		It("should be a valid issuer instance", func() {
			Expect(i).ToNot(BeNil())
		})
	})
	Context("Valid issuer", func() {
		It("should match issuer", func() {
			Expect(i.MatchIssuer("https://admin.rac.dev.totvs.app/totvs.rac")).To(BeTrue())
		})
		It("should not match issuer", func() {
			Expect(i.MatchIssuer("admin.rac.dev.totvs.com.br")).To(BeFalse())
		})
	})
	Context("Valid claims", func() {
		identityClaims := map[string]interface{}{
			"tenantIdpId": "abcd",
			"companyId":   "abcd",
			"client_id":   "clientid",
			"email":       "name@domain.com",
			"exp":         time.Now().Unix() + 1000,
			"aud":         "aud",
			"iss":         "issuer",
		}
		It("should match claims", func() {
			payload, err := json.Marshal(identityClaims)
			Expect(err).To(BeNil())
			var claims issuer.ClaimsBase
			err = i.ClaimsBase([]byte(payload), &claims)
			Expect(err).To(BeNil())
			Expect(claims.ClaimEmail() == "name@domain.com").To(BeTrue())
			Expect(claims.ClaimTenantIdpID() == "abcd").To(BeTrue())
			Expect(claims.ClaimCompanyID() == "abcd").To(BeTrue())
			Expect(claims.ClaimClientID() == "clientid").To(BeTrue())
			Expect(claims.ClaimAudience() == "aud").To(BeTrue())
			Expect(claims.ClaimIssuer() == "issuer").To(BeTrue())
		})
	})
	Context("Invalid claims", func() {
		identityClaims := map[string]interface{}{
			"exp": time.Now().Unix() + 1000,
		}
		It("should return error when payload is invalid", func() {
			var claims issuer.ClaimsBase
			err := i.ClaimsBase([]byte("abc"), &claims)
			Expect(err).ToNot(BeNil())
		})
		It("should not match claims", func() {
			payload, err := json.Marshal(identityClaims)
			Expect(err).To(BeNil())
			var claims issuer.ClaimsBase
			err = i.ClaimsBase([]byte(payload), &claims)
			Expect(err).To(BeNil())
			Expect(claims.ClaimEmail() == "-").To(BeTrue())
			Expect(claims.ClaimTenantIdpID() == "-").To(BeTrue())
			Expect(claims.ClaimCompanyID() == "-").To(BeTrue())
			Expect(claims.ClaimClientID() == "-").To(BeTrue())
			Expect(claims.ClaimAudience() == "-").To(BeTrue())
			Expect(claims.ClaimIssuer() == "-").To(BeTrue())
		})
	})
})
