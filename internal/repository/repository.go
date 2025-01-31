package repository

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/gorm"
)

type GormUserRepository struct {
	Db *gorm.DB
}

type UserRepository interface {
	Create(domain.CreateUserRequest) (domain.CreateUserResponse, error)
	Get(domain.GetUserRequest) (domain.GetUserResponse, error)
}

func (g GormUserRepository) Create(c domain.CreateUserRequest) (domain.CreateUserResponse, error) {
	user := domain.User{
		Email:        c.Email,
		Picture:      c.Picture,
		AccessToken:  c.AccessToken,
		RefreshToken: c.RefreshToken,
		TokenExpiry:  c.TokenExpiry,
	}
	result := g.Db.Create(&user)
	if result.Error != nil {
		return domain.CreateUserResponse{}, result.Error
	}

	return domain.CreateUserResponse{
		Id: user.ID,
	}, nil
}

func (g GormUserRepository) Get(r domain.GetUserRequest) (domain.GetUserResponse, error) {
	idAsString := strconv.FormatUint(uint64(r.Id), 10)
	result, err := g.Db.Get(idAsString)
	if !err {
		return domain.GetUserResponse{}, errors.New("error fetching user from database")
	}

	fmt.Printf("Result: %v\n", result)

	resp := domain.GetUserResponse{}

	// errUnmarshal := json.Unmarshal(result, &resp)
	// if errUnmarshal != nil {
	// 	return errors.New("Error unmarshalling user data")
	// }

	return resp, nil
}
