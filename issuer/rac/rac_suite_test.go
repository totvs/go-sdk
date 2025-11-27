package rac_test

import (
	"encoding/json"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/totvs/go-sdk/issuer/rac"
)

func TestIssuerRAC(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Issuer RAC Suite")
}

var _ = Describe("Test issuer rac", func() {
	issuerRAC := rac.NewRac("")
	Context("Initialize", func() {
		It("should be a valid rac issuer instance", func() {
			Expect(issuerRAC).ToNot(BeNil())
		})
	})
	Context("Valid claims", func() {
		racClaims := map[string]interface{}{
			"http://www.tnf.com/identity/claims/tenantId": "abcd",
			"exp": time.Now().Unix() + 1000,
		}
		It("should match claims", func() {
			payload, err := json.Marshal(racClaims)
			Expect(err).To(BeNil())
			claims, err := issuerRAC.Claims([]byte(payload))
			Expect(err).To(BeNil())
			Expect(claims.ClaimTenantIdpID() == "abcd").To(BeTrue())
			Expect(claims.ClaimCompanyID() == "abcd").To(BeTrue())
		})
	})
	Context("Invalid claims", func() {
		racClaims := map[string]interface{}{
			"exp": time.Now().Unix() + 1000,
		}
		It("should return error when payload is invalid", func() {
			_, err := issuerRAC.Claims([]byte("abc"))
			Expect(err).ToNot(BeNil())
		})
		It("should not match claims", func() {
			payload, err := json.Marshal(racClaims)
			Expect(err).To(BeNil())
			claims, err := issuerRAC.Claims([]byte(payload))
			Expect(err).To(BeNil())
			Expect(claims.ClaimTenantIdpID() == "-").To(BeTrue())
			Expect(claims.ClaimCompanyID() == "-").To(BeTrue())
		})
	})
})
