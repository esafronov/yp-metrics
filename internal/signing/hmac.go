// Package signing includes http server middleware for request signature validation
package signing

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"

	pb "github.com/esafronov/yp-metrics/internal/grpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const HeaderSignatureKey = "HashSHA256"

// Sign get hmac signature in hexadecimal format for body
func Sign(body []byte, key string) (signature string, err error) {
	h := hmac.New(sha256.New, []byte(key))
	if _, err = h.Write(body); err != nil {
		return
	}
	signature = hex.EncodeToString(h.Sum(nil))
	return
}

// IsValid check hmac signature for body using key
func IsValid(signature string, body []byte, key string) bool {
	h := hmac.New(sha256.New, []byte(key))
	if _, err := h.Write(body); err != nil {
		return false
	}
	trueSignature := h.Sum(nil)
	//decode from hexadecimal format into slice of bytes
	checkSignature, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	return hmac.Equal(checkSignature, trueSignature)
}

// ValidateSignature server middleware for checking signature in request using secretKey
func ValidateSignature(secretKey string) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sw := w
			//if secret key is not empty, we make validation of signature
			if secretKey != "" {
				signature := r.Header.Get(HeaderSignatureKey)
				if signature == "" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				//we need leave body unread for next middleware, so we recreate it
				r.Body = io.NopCloser(bytes.NewBuffer(body))
				//check request signature is valid
				if !IsValid(signature, body, secretKey) {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				//replace http.ResponseWriter on SingingResponseWriter
				sw = NewSigningResponseWriter(w, secretKey)
			}
			h.ServeHTTP(sw, r)
		})
	}
}

func NewSigningResponseWriter(w http.ResponseWriter, key string) *SigningResponseWriter {
	return &SigningResponseWriter{
		rw:        w,
		secretKey: key,
		body:      &bytes.Buffer{},
	}
}

type SigningResponseWriter struct {
	rw        http.ResponseWriter
	body      *bytes.Buffer
	secretKey string
}

func (w *SigningResponseWriter) Write(b []byte) (int, error) {
	len, err := w.rw.Write(b)
	if err != nil {
		return len, err
	}
	len, err = w.body.Write(b)
	return len, err
}

func (w *SigningResponseWriter) Header() http.Header {
	return w.rw.Header()
}

func (w *SigningResponseWriter) WriteHeader(statusCode int) {
	//if secret key is not empty and http status ok, calc response hash and set it in response header
	if w.secretKey != "" && statusCode == http.StatusOK {
		signature, err := Sign(w.body.Bytes(), w.secretKey)
		defer w.body.Reset()
		if err != nil {
			w.rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.rw.Header().Set(HeaderSignatureKey, signature)
	}
	w.rw.WriteHeader(statusCode)
}

// UnaryValidateSignatureInterceptor is the interceptor for gRPC server for validating remote calls
func UnaryValidateSignatureInterceptor(secretKey string) func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if secretKey != "" {
			pbRequest, ok := req.(*pb.BatchUpdateRequest)
			//do validation only for BatchUpdateRequest
			if !ok {
				return handler(ctx, req)
			}
			var signature string
			md, ok := metadata.FromIncomingContext(ctx)
			if !ok {
				return nil, status.Error(codes.InvalidArgument, "no metadata found")
			}
			values := md.Get(HeaderSignatureKey)
			if len(values) == 0 {
				return nil, status.Error(codes.InvalidArgument, "no signature")
			}
			signature = values[0]
			if signature == "" {
				return nil, status.Error(codes.InvalidArgument, "signature is empty")
			}
			marshaled, err := json.Marshal(pbRequest)
			if err != nil {
				return nil, err
			}
			if !IsValid(signature, marshaled, secretKey) {
				return nil, status.Error(codes.InvalidArgument, "signature is wrong")
			}
		}
		return handler(ctx, req)
	}
}
