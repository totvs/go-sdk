package identity_test

import (
	"encoding/json"
	"regexp"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/totvs/go-sdk/auth/issuer"
)

func TestIssuerIdentity(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Issuer Identity Suite")
}

var _ = Describe("Test issuer identity", func() {
	i := &issuer.IssuerBase{}
	i.IssuerRegex = regexp.MustCompile(`(?m)^\*\.fluig\.io$`)
	Context("Initialize", func() {
		It("should be a valid issuer instance", func() {
			Expect(i).ToNot(BeNil())
		})
	})
	Context("Valid issuer", func() {
		It("should match issuer", func() {
			Expect(i.MatchIssuer("*.fluig.io")).To(BeTrue())
		})
		It("should not match issuer", func() {
			Expect(i.MatchIssuer("https://admin.rac.dev.totvs.app/totvs.rac")).To(BeFalse())
		})
	})
	Context("Valid claims", func() {
		identityClaims := map[string]interface{}{
			"tenantIdpId": "abcd",
			"companyId":   "abcd",
			"exp":         time.Now().Unix() + 1000,
			"roles":       []string{"admin"},
			"fullName":    "John Doe",
		}
		It("should match claims", func() {
			payload, err := json.Marshal(identityClaims)
			Expect(err).To(BeNil())
			var claims issuer.ClaimsBase
			err = i.ClaimsBase([]byte(payload), &claims)
			Expect(err).To(BeNil())
			Expect(claims.ClaimTenantIdpID() == "abcd").To(BeTrue())
			Expect(claims.ClaimCompanyID() == "abcd").To(BeTrue())
			Expect(claims.ClaimRoles()).To(Equal([]string{"admin"}))
			Expect(claims.ClaimFullName() == "John Doe").To(BeTrue())
		})
	})
})
