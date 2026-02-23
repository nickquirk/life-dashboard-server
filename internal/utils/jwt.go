package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Helper to prevent 401 Aunauthorised errors due to secret being empty
func getSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		slog.Error("JWT_SECRET not set")
		os.Exit(1)
	}
	return []byte(secret)
}

func GenerateToken(id uint, email string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":    id,
		"email": email,
		"exp":   time.Now().Add(time.Hour * 6).Unix(),
	})
	return token.SignedString(getSecret())
}

func VerifyToken(tokenString string) (jwt.MapClaims, error) {
	parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, errors.New("unexpected signing method")
		}
		return getSecret(), nil
	})

	if err != nil {
		return nil, errors.New("could not parse token: " + err.Error())
	}

	if !parsedToken.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

func GetUserIdFromToken(tokenString string) (uint, error) {
	claims, err := VerifyToken(tokenString)
	if err != nil {
		return 0, err
	}

	// JWT JSON numbers are often float64 by default
	idFloat, ok := claims["id"].(float64)
	if ok {
		return uint(idFloat), nil
	}

	// Fallback: sometimes it might be stored as a string or int depending on config
	idInt, ok := claims["id"].(int)
	if ok {
		return uint(idInt), nil
	}

	// Fallback if strict JSON parsing wasn't used
	idUint, ok := claims["id"].(uint)
	if ok {
		return idUint, nil
	}

	return 0, fmt.Errorf("unable to parse ID from token claims: %v", claims["id"])
}

// GenerateRefreshToken creates a cryptographically random 32-byte token and returns it base64url-encoded.
func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// HashRefreshToken returns the hex-encoded SHA-256 hash of a refresh token.
func HashRefreshToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}

// GetUserIdFromExpiredToken parses a JWT verifying the signature but skipping expiry validation.
// This is used during token refresh when the JWT has expired but we still need the user ID.
func GetUserIdFromExpiredToken(tokenString string) (uint, error) {
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	parsedToken, err := parser.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return getSecret(), nil
	})
	if err != nil {
		return 0, fmt.Errorf("could not parse expired token: %w", err)
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return 0, errors.New("invalid token claims")
	}

	idFloat, ok := claims["id"].(float64)
	if ok {
		return uint(idFloat), nil
	}

	return 0, fmt.Errorf("unable to parse ID from expired token claims: %v", claims["id"])
}
