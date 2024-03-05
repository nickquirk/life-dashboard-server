package utils

import (
	"errors"
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

func VerifyToken(token string) (string, error) {
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)

		if !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return "", errors.New("could not parse token")
	}

	tokenIsValid := parsedToken.Valid

	if !tokenIsValid {
		return "", errors.New("invalid token")
	}

	// claims will automatically be set to type jwt.MapClaims
	claimsMap, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid token claims ")
	}

	accessToken := claimsMap["access_token"].(string)

	return accessToken, nil
}

func GetUserFromToken(token string) (models.User, error) {
	userData := models.User{}

	// claims, err := VerifyToken(token)
	// if err != nil {
	// 	return userData, errors.New("unable to get claims from token")
	// }

	// userData.Id = claims["id"].(string)
	// userData.Email = claims["email"].(string)
	// userData.Picture = claims["picture"].(string)
	// userData.AccessToken = claims["access_token"].(string)

	return userData, nil
}
