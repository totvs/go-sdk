package authorization_bearer_token_test

import (
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

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
