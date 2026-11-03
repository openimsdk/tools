package tokenverify

import (
	"errors"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/openimsdk/protocol/constant"
	"github.com/openimsdk/tools/errs"
)

var secret = "OpenIM_server"

func Test_ParseToken(t *testing.T) {
	claims1 := BuildClaims("123456", constant.AndroidPadPlatformID, 10)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims1)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	claim2, err := GetClaimFromToken(tokenString, secretFun())
	if err != nil {
		t.Fatal(err)
	}
	t.Log(claim2)
}

func Test_GetClaimFromTokenErrors(t *testing.T) {
	tests := []struct {
		name  string
		token string
		key   string
		want  error
	}{
		{name: "expired", token: signedToken(t, BuildClaims("123456", constant.AndroidPadPlatformID, -1), secret), key: secret, want: errs.ErrTokenExpired},
		{name: "malformed", token: "not-a-jwt", key: secret, want: errs.ErrTokenMalformed},
		{name: "wrong secret", token: signedToken(t, BuildClaims("123456", constant.AndroidPadPlatformID, 10), "other-secret"), key: secret, want: errs.ErrTokenUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetClaimFromToken(tt.token, func(token *jwt.Token) (any, error) {
				return []byte(tt.key), nil
			})
			if !errors.Is(err, tt.want) {
				t.Fatalf("GetClaimFromToken() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func signedToken(t *testing.T, claims Claims, key string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(key))
	if err != nil {
		t.Fatal(err)
	}
	return tokenString
}

func secretFun() jwt.Keyfunc {
	return func(token *jwt.Token) (any, error) {
		return []byte(secret), nil
	}
}
