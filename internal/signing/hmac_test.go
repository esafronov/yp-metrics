package signing

import (
	"testing"
)

func TestIsValid(t *testing.T) {
	type args struct {
		body      []byte
		key       string
		signature string
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
