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

func (h *Handler) CreateUser(user domain.CreateUserRequest) (domain.CreateUserResponse, error) {
	resp, err := h.Service.CreateUser(user)
	if err != nil {
		errorMessage := fmt.Sprint(err)
		return domain.CreateUserResponse{}, errors.New(errorMessage)
	}
	return resp, nil
}

func (h *Handler) GetUser(user domain.GetUserRequest) (domain.GetUserResponse, error) {
	resp, err := h.Service.GetUser(user)
	if err != nil {
		return domain.GetUserResponse{}, err
	}
	return resp, nil
}
