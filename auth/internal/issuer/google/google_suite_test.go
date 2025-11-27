package google_test

import (
	"encoding/json"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/totvs/go-sdk/auth/internal/issuer/google"
)

func TestIssuerGoogle(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Issuer Google Suite")
}

var _ = Describe("Test issuer Google", func() {
	issuerGoogle := google.NewGoogle("")
	Context("Initialize", func() {
		It("should be a valid Google issuer instance", func() {
			Expect(issuerGoogle).ToNot(BeNil())
		})
	})
	Context("Valid claims", func() {
		identityClaims := map[string]interface{}{
			"email": "name@domain.com",
			"exp":   time.Now().Unix() + 1000,
		}
		It("should match claims", func() {
			payload, err := json.Marshal(identityClaims)
			Expect(err).To(BeNil())
			claims, err := issuerGoogle.Claims([]byte(payload))
			Expect(err).To(BeNil())
			Expect(claims.ClaimEmail() == "name@domain.com").To(BeTrue())
			Expect(claims.ClaimTenantIdpID() == "-").To(BeTrue())
			Expect(claims.ClaimCompanyID() == "-").To(BeTrue())
		})
	})
})
