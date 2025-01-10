// Package encrypt includes middleware for decrypting request body if privateKey is set with parameters
package encrypt

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/esafronov/yp-metrics/internal/logger"
	"go.uber.org/zap"
)

var once sync.Once

var privateKeyRsa *rsa.PrivateKey //rsa private key

// DecryptingMiddleware server middleware for decrypting messages from agent
func DecryptingMiddleware(privateKey string) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if privateKey != "" {
				var err error
				//read pem file with secret key once
				once.Do(func() {
					var privateKeyPEM []byte
					privateKeyPEM, err = os.ReadFile(privateKey)
					if err != nil {
						return
					}
					privateKeyBlock, _ := pem.Decode(privateKeyPEM)
					privateKeyRsa, err = x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
					if err != nil {
						return
					}
				})
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					logger.Log.Info("read body", zap.Error(err))
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				decodedBody, err := base64.StdEncoding.DecodeString(string(body))
				if err != nil {
					logger.Log.Info("decode body from base64", zap.Error(err))
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				decryptedBody, err := rsa.DecryptPKCS1v15(rand.Reader, privateKeyRsa, decodedBody)
				if err != nil {
					logger.Log.Info("decrypt body", zap.Error(err))
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				// we need leave body unread for next middleware, so we recreate it
				r.Body = io.NopCloser(bytes.NewBuffer(decryptedBody))
			}

			// передаём управление хендлеру
			h.ServeHTTP(w, r)
		})
	}
}

func EncryptBody(body []byte, publicKey string) (encryptedBody []byte, err error) {
	var publicKeyPEM []byte
	publicKeyPEM, err = os.ReadFile(publicKey)
	if err != nil {
		return
	}
	publicKeyBlock, _ := pem.Decode(publicKeyPEM)
	var pk any
	pk, err = x509.ParsePKIXPublicKey(publicKeyBlock.Bytes)
	if err != nil {
		return
	}
	publicKeyRsa := pk.(*rsa.PublicKey)
	encryptedBytes, err := rsa.EncryptPKCS1v15(rand.Reader, publicKeyRsa, body)
	enryptedStr := base64.StdEncoding.EncodeToString(encryptedBytes)
	encryptedBody = []byte(enryptedStr)
	return
}
