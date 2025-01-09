package handlers

import (
	"errors"
	"fmt"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/service"
)

type Handler struct {
	Service service.Service
}

func (h *Handler) CreateUser(user domain.CreateUserRequest) (uint, error) {
	createUserResponse, err := h.Service.CreateUser(user)
	if err != nil {
		errorMessage := fmt.Sprint(err)
		return 0, errors.New(errorMessage)
	}
	return createUserResponse.Id, nil
}
