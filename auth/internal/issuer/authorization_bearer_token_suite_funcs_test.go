package issuer_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"syscall"

	"github.com/golang-jwt/jwt/v5"
	"github.com/totvs/go-sdk/auth/internal/issuer"
)

func createPipe() (r *os.File, w *os.File) {
	pr, pw, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	return pr, pw
}

func createFIFO(fifoPath string) {
	if err := syscall.Mkfifo(fifoPath, uint32(0777)); err != nil {
		if !os.IsExist(err) { // Ignore error if FIFO already exists
			panic(err)
		}
	}
}

func initializeLog() {
	fifoPath := "/tmp/logs.log"
	createFIFO(fifoPath)
}

func lastReadStringOfBuffer(buf *bytes.Buffer) string {
	var lastLine string
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			if line != "" {
				lastLine = line
			}
			break
		}
		if line != "" {
			lastLine = line
		}
	}
	return lastLine
}

func generateRecordForTest(a *issuer.AuthorizationBearerToken, req *http.Request) logObj {
	pr, pw := createPipe()
	defer pr.Close()

	std := os.Stderr
	os.Stderr = pw

	initializeLog()
	a.ValidBearerToken(req)

	pw.Close()
	os.Stderr = std

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(pr)
	output := lastReadStringOfBuffer(&buf)

	var logObj = struct {
		Level string            `json:"level"`
		Value map[string]string `json:"value"`
	}{}
	json.Unmarshal([]byte(output), &logObj)

	return logObj
}

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
