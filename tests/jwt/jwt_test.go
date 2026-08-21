package jwt_test

import (
	"errors"
	"testing"
	"time"

	jwtpkg "my-bbs/pkg/jwt"

	gojwt "github.com/golang-jwt/jwt/v5"
)

func TestGeneratedTokenExposesUniqueRevocableClaims(t *testing.T) {
	jwtpkg.Init("jwt-public-contract-test-secret")

	firstToken, err := jwtpkg.GenerateToken(42)
	if err != nil {
		t.Fatalf("GenerateToken(first) error = %v", err)
	}
	secondToken, err := jwtpkg.GenerateToken(42)
	if err != nil {
		t.Fatalf("GenerateToken(second) error = %v", err)
	}

	firstClaims, err := jwtpkg.ParseClaims(firstToken)
	if err != nil {
		t.Fatalf("ParseClaims(first) error = %v", err)
	}
	secondClaims, err := jwtpkg.ParseClaims(secondToken)
	if err != nil {
		t.Fatalf("ParseClaims(second) error = %v", err)
	}
	if firstClaims.UserID != 42 {
		t.Fatalf("UserID = %d, want 42", firstClaims.UserID)
	}
	if firstClaims.ID == "" {
		t.Fatal("generated token is missing JTI")
	}
	if firstClaims.ID == secondClaims.ID {
		t.Fatalf("two tokens share JTI %q", firstClaims.ID)
	}
	if firstClaims.ExpiresAt == nil || !firstClaims.ExpiresAt.After(time.Now()) {
		t.Fatalf("ExpiresAt = %v, want a future time", firstClaims.ExpiresAt)
	}

	userID, err := jwtpkg.ParseToken(firstToken)
	if err != nil || userID != 42 {
		t.Fatalf("ParseToken() = (%d, %v), want (42, nil)", userID, err)
	}
}

func TestParseClaimsRejectsTokensOutsideSigningContract(t *testing.T) {
	const secret = "jwt-signing-contract-test-secret"
	jwtpkg.Init(secret)

	tests := []struct {
		name   string
		method gojwt.SigningMethod
		issuer string
	}{
		{name: "wrong algorithm", method: gojwt.SigningMethodHS512, issuer: "my-bbs"},
		{name: "wrong issuer", method: gojwt.SigningMethodHS256, issuer: "another-service"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := gojwt.NewWithClaims(tt.method, jwtpkg.Claims{
				UserID: 42,
				RegisteredClaims: gojwt.RegisteredClaims{
					ID:        "external-token-id",
					Issuer:    tt.issuer,
					ExpiresAt: gojwt.NewNumericDate(time.Now().Add(time.Hour)),
				},
			})
			signed, err := token.SignedString([]byte(secret))
			if err != nil {
				t.Fatalf("SignedString() error = %v", err)
			}

			if _, err := jwtpkg.ParseClaims(signed); !errors.Is(err, jwtpkg.ErrTokenInvalid) {
				t.Fatalf("ParseClaims() error = %v, want ErrTokenInvalid", err)
			}
		})
	}
}
