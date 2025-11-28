package issuer_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/totvs/go-sdk/auth"
)

type logObj struct {
	Level string            `json:"level"`
	Value map[string]string `json:"value"`
}

func TestAudit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Audit Suite")
}

var _ = Describe("Test package audit", func() {
	serveJWKS()

	urlTest := "http://localhost:4444/jwks"

	a := auth.NewAuthorizationBearerToken(urlTest, urlTest, urlTest)

	urlDefault := &url.URL{
		Scheme: "http",
		Host:   "totvs.app",
		Path:   "/path/to/resource",
	}

	Context("Initialize", func() {
		It("should be a valid audit instance", func() {
			Expect(a).ToNot(BeNil())
		})
	})

	Context("Generate record of log", func() {
		It("Should be a valid empty and not authenticated request", func() {
			request := &http.Request{
				Method: "GET",
				URL:    urlDefault,
				Header: map[string][]string{
					"Content-Type": {"application/json"},
				},
			}

			logObj := generateRecordForTest(a, request)

			Expect(logObj.Level == "OK").To(BeTrue())
			Expect(logObj.Value["src_audience"] == "-").To(BeTrue())
			Expect(logObj.Value["src_issuer"] == "-").To(BeTrue())
			Expect(logObj.Value["src_tenant_id"] == "-").To(BeTrue())
			Expect(logObj.Value["src_client_id"] == "-").To(BeTrue())
			Expect(logObj.Value["src_email"] == "-").To(BeTrue())
			Expect(logObj.Value["src_url_path"] == "-").To(BeTrue())
			Expect(logObj.Value["src_auth_status"] == "0").To(BeTrue())
			Expect(logObj.Value["src_company_id"] == "-").To(BeTrue())
			Expect(logObj.Value["src_real_ip"] == "-").To(BeTrue())
			Expect(logObj.Value["src_url_host"] == "-").To(BeTrue())
			Expect(logObj.Value["src_namespace"] == "-").To(BeTrue())
			Expect(logObj.Value["src_original_method"] == "GET").To(BeTrue())
			Expect(logObj.Value["src_url_schema"] == "-").To(BeTrue())
			Expect(logObj.Value["src_user_agent"] == "-").To(BeTrue())
		})

		It("Should be a valid simple and not authenticated request", func() {
			request := &http.Request{
				Method: "GET",
				URL:    urlDefault,
				Header: map[string][]string{
					"Content-Type":   {"application/json"},
					"X-Real-Ip":      {"127.0.0.1"},
					"X-Original-Url": {urlDefault.String()},
					"X-Namespace":    {"default"},
					"User-Agent":     {"AppleWebKit/537.36"},
				},
			}

			logObj := generateRecordForTest(a, request)

			Expect(logObj.Level == "OK").To(BeTrue())
			Expect(logObj.Value["src_audience"] == "-").To(BeTrue())
			Expect(logObj.Value["src_issuer"] == "-").To(BeTrue())
			Expect(logObj.Value["src_tenant_id"] == "-").To(BeTrue())
			Expect(logObj.Value["src_client_id"] == "-").To(BeTrue())
			Expect(logObj.Value["src_email"] == "-").To(BeTrue())
			Expect(logObj.Value["src_url_path"] == "/path/to/resource").To(BeTrue())
			Expect(logObj.Value["src_auth_status"] == "0").To(BeTrue())
			Expect(logObj.Value["src_company_id"] == "-").To(BeTrue())
			Expect(logObj.Value["src_real_ip"] == "127.0.0.1").To(BeTrue())
			Expect(logObj.Value["src_url_host"] == "totvs.app").To(BeTrue())
			Expect(logObj.Value["src_namespace"] == "default").To(BeTrue())
			Expect(logObj.Value["src_original_method"] == "GET").To(BeTrue())
			Expect(logObj.Value["src_url_schema"] == "http").To(BeTrue())
			Expect(logObj.Value["src_user_agent"] == "AppleWebKit/537.36").To(BeTrue())
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
				"sub":         "felipe.conti@totvs.com.br",
				"aud":         "fluig_authenticator_resource",
				"exp":         time.Now().UTC().Add(time.Hour).Unix(),
				"iat":         time.Now().UTC().Unix(),
				"email":       "felipe.conti@totvs.com.br",
				"client_id":   "manager",
				"tenantIdpId": "2d4c74cfac2e438b97f110b185530ecb",
				"companyId":   "2d4c74cfac2e438b97f110b185530ecb",
			}

			jwt, _ := generateJWT(claims)
			request.Header["Authorization"] = []string{"Bearer " + jwt}

			logObj := generateRecordForTest(a, request)

			Expect(logObj.Level == "OK").To(BeTrue())
			Expect(logObj.Value["src_auth_status"] == "1").To(BeTrue())
			Expect(logObj.Value["src_audience"] == "fluig_authenticator_resource").To(BeTrue())
			Expect(logObj.Value["src_company_id"] == "2d4c74cfac2e438b97f110b185530ecb").To(BeTrue())
			Expect(logObj.Value["src_tenant_id"] == "2d4c74cfac2e438b97f110b185530ecb").To(BeTrue())
			Expect(logObj.Value["src_client_id"] == "manager").To(BeTrue())
			Expect(logObj.Value["src_email"] == "felipe.conti@totvs.com.br").To(BeTrue())
			Expect(logObj.Value["src_issuer"] == "*.fluig.io").To(BeTrue())
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
				"sub":       "felipe.conti@totvs.com.br",
				"aud":       "authorization_api",
				"exp":       time.Now().UTC().Add(time.Hour).Unix(),
				"iat":       time.Now().UTC().Unix(),
				"email":     "felipe.conti@totvs.com.br",
				"client_id": "manager",
				"http://www.tnf.com/identity/claims/tenantId": "2d4c74cfac2e438b97f110b185530ecb",
			}

			jwt, _ := generateJWT(claims)
			request.Header["Authorization"] = []string{"Bearer " + jwt}

			logObj := generateRecordForTest(a, request)

			Expect(logObj.Level == "OK").To(BeTrue())
			Expect(logObj.Value["src_auth_status"] == "1").To(BeTrue())
			Expect(logObj.Value["src_audience"] == "authorization_api").To(BeTrue())
			Expect(logObj.Value["src_company_id"] == "2d4c74cfac2e438b97f110b185530ecb").To(BeTrue())
			Expect(logObj.Value["src_tenant_id"] == "2d4c74cfac2e438b97f110b185530ecb").To(BeTrue())
			Expect(logObj.Value["src_client_id"] == "manager").To(BeTrue())
			Expect(logObj.Value["src_email"] == "felipe.conti@totvs.com.br").To(BeTrue())
			Expect(logObj.Value["src_issuer"] == "https://admin.rac.dev.totvs.app/totvs.rac").To(BeTrue())
		})
	})

	Context("Read App Name from HTTP header User Agent", func() {
		It("should extract correct app name from common user agents ", func() {
			run_test("./common/user_agents/desktop.json")
			run_test("./common/user_agents/mobile.json")
			run_test("./common/user_agents/other.json")
		})
	})

	Context("Generate record of log with specifics url paths", func() {
		It("Should be a valid url request", func() {
			request := &http.Request{
				Method: "GET",
				URL:    &url.URL{},
				Header: map[string][]string{
					"Content-Type": {"application/json"},
				},
			}

			file, err := os.Open("./common/urls/urls.txt")
			if err != nil {
				fmt.Println("Erro ao abrir o arquivo:", err)
				return
			}
			defer file.Close()

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				pathLine := scanner.Text()
				request.Header["X-Original-Url"] = []string{pathLine}
				logObj := generateRecordForTest(a, request)
				Expect(logObj.Value["src_url_path"] == pathLine).To(BeTrue())
			}

			if err := scanner.Err(); err != nil {
				fmt.Println("Erro ao ler o arquivo:", err)
			}
		})
	})
})

type data struct {
	UA  string `json:"ua"`
	App string `json:"app"`
}

func run_test(filepath string) {
	file, err := os.Open(filepath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}

	fileSize := fileInfo.Size()
	buffer := make([]byte, fileSize)
	_, err = file.Read(buffer)
	if err != nil {
		log.Fatal(err)
	}

	var list []data
	err = json.Unmarshal(buffer, &list)
	if err != nil {
		log.Fatal(err)
	}
}
