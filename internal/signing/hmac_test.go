package signing

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValid(t *testing.T) {
	type args struct {
		key       string
		signature string
		body      []byte
	}

	body := []byte(`{"id":"ggg","type":"gauge","value":1.00001}`)
	s1, _ := Sign([]byte(body), "123")

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "positive for signature " + s1,
			args: args{
				key:       "123",
				body:      []byte(body),
				signature: s1,
			},
			want: true,
		},
		{
			name: "negative",
			args: args{
				key:       "123",
				body:      []byte("hello2"),
				signature: s1,
			},
			want: false,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValid(tt.args.signature, tt.args.body, tt.args.key); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateSignature(t *testing.T) {
	reqbody := `
		{
			"id":"someparam",
			"type":"counter",
			"delta":1
		}
	`
	secretKey := "123"
	ValidateSignatureHandler := ValidateSignature(secretKey)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		require.Equal(t, reqbody, string(body))
	})
	handlerToTest := ValidateSignatureHandler(nextHandler)
	w := httptest.NewRecorder()
	signature, err := Sign([]byte(reqbody), "123")
	if err != nil {
		require.NoError(t, err)
	}
	reader := strings.NewReader(reqbody)
	req := httptest.NewRequest("POST", "/", reader)
	req.Header.Set(HeaderSignatureKey, signature)
	req.Header.Set("Content-Type", "application/json")
	handlerToTest.ServeHTTP(w, req)
	res := w.Result()
	require.Equal(t, http.StatusOK, res.StatusCode)
}
