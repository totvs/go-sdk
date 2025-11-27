package identity_test

import (
	"encoding/json"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/totvs/go-sdk/issuer/identity"
)

func TestIssuerIdentity(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Issuer Identity Suite")
}

var _ = Describe("Test issuer identity", func() {
	issuerIdentity := identity.NewIdentity("")
	Context("Initialize", func() {
		It("should be a valid Identity issuer instance", func() {
			Expect(issuerIdentity).ToNot(BeNil())
		})
	})
	Context("Valid claims", func() {
		identityClaims := map[string]interface{}{
			"tenantIdpId": "abcd",
			"companyId":   "abcd",
			"exp":         time.Now().Unix() + 1000,
		}
		It("should match claims", func() {
			payload, err := json.Marshal(identityClaims)
			Expect(err).To(BeNil())
			claims, err := issuerIdentity.Claims([]byte(payload))
			Expect(err).To(BeNil())
			Expect(claims.ClaimTenantIdpID() == "abcd").To(BeTrue())
			Expect(claims.ClaimCompanyID() == "abcd").To(BeTrue())
		})
	})
})
