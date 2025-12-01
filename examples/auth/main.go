package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/totvs/go-sdk/auth"
	"github.com/totvs/go-sdk/auth/issuer/google"
	"github.com/totvs/go-sdk/auth/issuer/identity"
	"github.com/totvs/go-sdk/auth/issuer/rac"
)

const jwksURL = "http://localhost:4444/jwks"

// httpServer demonstra uso do middleware HTTPAuthorizationBearerTokenMiddleware em um servidor HTTP.
func httpServer() {
	googleIssuer := google.NewGoogle(jwksURL)
	identityIssuer := identity.NewIdentity(jwksURL)
	racIssuer := rac.NewRac(jwksURL)

	authBearerToken := auth.NewAuthorizationBearerToken(googleIssuer, identityIssuer, racIssuer)

	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		claims := auth.GetIssuerClaimsFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Hello, %s! You are %v and your email is %s", claims.ClaimFullName(), claims.ClaimRoles(), claims.ClaimEmail())
	})

	mux.HandleFunc("/role", func(w http.ResponseWriter, r *http.Request) {
		if !auth.HasRole(r.Context(), "admin") {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintf(w, "Forbidden")
		}

	})

	handler := auth.HTTPAuthorizationBearerTokenMiddleware(authBearerToken)(mux)

	go func() {
		err := http.ListenAndServe(":8080", handler)
		if err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)
}

// exampleGetIssuerClaimsFromContext demonstra como obter os claims do issuer a partir do contexto da requisição.
func exampleGetIssuerClaimsFromContext() {
	claims := jwt.MapClaims{
		"iss":       "*.fluig.io",
		"sub":       "totvs@totvs.com.br",
		"aud":       "fluig_authenticator_resource",
		"exp":       time.Now().Add(time.Hour).Unix(),
		"iat":       time.Now().Unix(),
		"email":     "totvs@totvs.com.br",
		"client_id": "manager",
		"roles":     []string{"admin"},
		"fullName":  "John Doe",
	}

	jwtToken, err := generateJWT(claims)
	if err != nil {
		log.Fatalf("Failed to generate JWT: %v", err)
	}

	req, err := http.NewRequest("GET", "http://localhost:8080/user", nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+jwtToken)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
		return
	}
	fmt.Println(string(body))
}

// exampleHasRole demonstra como verificar se o usuário tem um papel específico.
func exampleHasRole() {
	claims := jwt.MapClaims{
		"iss":       "*.fluig.io",
		"sub":       "totvs@totvs.com.br",
		"aud":       "fluig_authenticator_resource",
		"exp":       time.Now().Add(time.Hour).Unix(),
		"iat":       time.Now().Unix(),
		"email":     "totvs@totvs.com.br",
		"client_id": "manager",
		"roles":     []string{"user"},
		"fullName":  "John Doe",
	}

	jwtToken, err := generateJWT(claims)
	if err != nil {
		log.Fatalf("Failed to generate JWT: %v", err)
	}

	req, err := http.NewRequest("GET", "http://localhost:8080/role", nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+jwtToken)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
		return
	}
	fmt.Printf("Status: %d, Body: %s\n", resp.StatusCode, string(body))
}

// serveJWKS serve o JWKS para o servidor de autenticação para fins de demonstração.
func serveJWKS() {
	key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(publicKey))
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"keys": [
				{
					"kid": "key-id",
					"e": "AQAB",
					"kty": "RSA",
					"alg": "RS256",
					"n": "` + base64.RawURLEncoding.EncodeToString(key.N.Bytes()) + `"
				}
			]
		}`))
	})
	go http.ListenAndServe(":4444", nil)
}

// generateJWT gera um JWT com as claims fornecidas para fins de demonstração.
func generateJWT(claims map[string]interface{}) (string, error) {
	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(privateKey))
	if err != nil {
		return "", fmt.Errorf("create: parse key: %w", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims(claims))
	token.Header["kid"] = "key-id"
	jwt, err := token.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("create: sign token: %w", err)
	}

	return jwt, nil
}

func main() {
	serveJWKS()
	httpServer()
	exampleGetIssuerClaimsFromContext()
	exampleHasRole()
}

// privateKey e publicKey são chaves de exemplo apenas para fins de demonstração.
// Em produção, nunca inclua chaves privadas no código.
var (
	privateKey = `-----BEGIN PRIVATE KEY-----
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
tHJ70N/bUoCN5r9KNSz/nR0NJUDA
-----END PRIVATE KEY-----`
	publicKey = `-----BEGIN PUBLIC KEY-----
MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAt5AEZGiqAM7zZ2Oc0aTK
MCdEZvRyaMw6bUH2BMGB4xZJtlnQBjpsDw5fRIYpS+CniCZQEOxi7BYJKmzEnwp6
5njdYE9GVYzT5o4C7TOFgIikRuce1LYqX65K2kMcJVSR6Ly9n6UIHqqrMECkPEdH
QW+GGl2xN4k9kdm4IqRpPSShVvbPMBhCUXPeS5AII0zK6jzgPqmpZRDlaBz7HzYS
krGH891Kn+h39rlhBGdKG1KZezxUAeQoD/61CwDArlM6eAMp7cVopKkGC2+CySXg
AkMnkP1/OewqPObdM112RGvwNp22tYB1mWL6AuQrnEEmy7npBMTu6+h0PZxJJ9y+
9oD1/O1KqNoD7j9kA7SIvWApYAp5tWRkrxUFHseSYjztHKbcmuxIutZtdF2TU8mB
6mVSr/DCH7QZP7Iz3rad+2UXI3tTz+btGEIBiP/w0ONSQQEZydL/wUTkwW5P/Nv1
EG/eCWrqj4Gn7pzA/nFRfxK7mTA/6UlXYo0uuV/3TCXXve5qeia8K6pFXRfvtWqK
FXoNBIYKsPEzD/DVYINhzf9TsERu2FwQhtfyBLnznN9QkdUZODrCfbVEfgrud3zm
l3UFbeiaMb8tTB2ztQBvbhuWJSJx+tRwoMzaVrXzIfZ+zcbvG8jn6zHhIA3WbvyX
xaZQTOTGfc9sQDAo/7OjAZUCAwEAAQ==
-----END PUBLIC KEY-----`
)
