package utils

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/nickquirk/life-dashboard-server/internal/models"
)

var secretKey = os.Getenv("SECRET")

func GenerateToken(user models.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":           user.Id,
		"email":        user.Email,
		"picture":      user.Picture,
		"access_token": user.AccessToken,
		"exp":          time.Now().Add(time.Hour * 6).Unix(),
	})
	return token.SignedString([]byte(secretKey))
}

func verifyToken(token string) (models.User, error) {
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)

		if !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secretKey), nil
	})

	userData := models.User{}

	if err != nil {
		return userData, errors.New("could not parse token")
	}

	tokenIsValid := parsedToken.Valid

	if tokenIsValid {
		return userData, errors.New("invalid token")
	}

	// claims will automatically be set to type jwt.MapClaims
	claims, ok := parsedToken.Claims.(jwt.MapClaims)

	fmt.Printf("claims: %v", claims)

	if !ok {
		return userData, errors.New("invalid token claims ")
	}

	return userData, nil
}
