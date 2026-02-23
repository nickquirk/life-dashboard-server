package utils

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupJWT(t *testing.T) {
	t.Helper()
	t.Setenv("JWT_SECRET", "test-secret-key-for-unit-tests-32chars!")
}

func TestGenerateToken_And_VerifyToken_Roundtrip(t *testing.T) {
	setupJWT(t)

	token, err := GenerateToken(42, "alice@example.com")
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := VerifyToken(token)
	require.NoError(t, err)
	assert.Equal(t, float64(42), claims["id"])
	assert.Equal(t, "alice@example.com", claims["email"])
}

func TestVerifyToken_Expired(t *testing.T) {
	setupJWT(t)

	// Manually create an expired token
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":    float64(1),
		"email": "bob@example.com",
		"exp":   time.Now().Add(-time.Hour).Unix(),
	})
	signed, err := tok.SignedString([]byte("test-secret-key-for-unit-tests-32chars!"))
	require.NoError(t, err)

	_, err = VerifyToken(signed)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse token")
}

func TestVerifyToken_BadSignature(t *testing.T) {
	setupJWT(t)

	// Sign with a different secret
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":    float64(1),
		"email": "bad@example.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	signed, err := tok.SignedString([]byte("wrong-secret-key-for-unit-tests-32chars"))
	require.NoError(t, err)

	_, err = VerifyToken(signed)
	assert.Error(t, err)
}

func TestVerifyToken_GarbageInput(t *testing.T) {
	setupJWT(t)
	_, err := VerifyToken("not.a.jwt")
	assert.Error(t, err)
}

func TestGetUserIdFromToken_Success(t *testing.T) {
	setupJWT(t)

	token, err := GenerateToken(99, "user@example.com")
	require.NoError(t, err)

	id, err := GetUserIdFromToken(token)
	require.NoError(t, err)
	assert.Equal(t, uint(99), id)
}

func TestGetUserIdFromToken_InvalidToken(t *testing.T) {
	setupJWT(t)
	_, err := GetUserIdFromToken("garbage")
	assert.Error(t, err)
}

func TestGetUserIdFromExpiredToken_Success(t *testing.T) {
	setupJWT(t)

	// Create an expired token
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":    float64(7),
		"email": "expired@example.com",
		"exp":   time.Now().Add(-time.Hour).Unix(),
	})
	signed, err := tok.SignedString([]byte("test-secret-key-for-unit-tests-32chars!"))
	require.NoError(t, err)

	id, err := GetUserIdFromExpiredToken(signed)
	require.NoError(t, err)
	assert.Equal(t, uint(7), id)
}

func TestGetUserIdFromExpiredToken_BadSignature(t *testing.T) {
	setupJWT(t)

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":  float64(1),
		"exp": time.Now().Add(-time.Hour).Unix(),
	})
	signed, err := tok.SignedString([]byte("wrong-secret"))
	require.NoError(t, err)

	_, err = GetUserIdFromExpiredToken(signed)
	assert.Error(t, err)
}

func TestGenerateRefreshToken_Uniqueness(t *testing.T) {
	t1, err := GenerateRefreshToken()
	require.NoError(t, err)
	t2, err := GenerateRefreshToken()
	require.NoError(t, err)

	assert.NotEqual(t, t1, t2)
	assert.True(t, len(t1) > 20, "token should be reasonably long")
}

func TestGenerateRefreshToken_Length(t *testing.T) {
	tok, err := GenerateRefreshToken()
	require.NoError(t, err)
	// 32 bytes base64url encoded = 44 chars
	assert.Equal(t, 44, len(tok))
}

func TestHashRefreshToken_Deterministic(t *testing.T) {
	h1 := HashRefreshToken("my-token")
	h2 := HashRefreshToken("my-token")
	assert.Equal(t, h1, h2)
}

func TestHashRefreshToken_DifferentInputs(t *testing.T) {
	h1 := HashRefreshToken("token-a")
	h2 := HashRefreshToken("token-b")
	assert.NotEqual(t, h1, h2)
}
