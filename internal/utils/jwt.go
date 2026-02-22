package utils

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Helper to prevent 401 Aunauthorised errors due to secret being empty
func getSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatalf("JWT_SECRET not set")
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
