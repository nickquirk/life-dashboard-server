package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
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

func (h *Handler) GetUserHTTP(w http.ResponseWriter, r *http.Request) {
	// get ID from URL param
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// 2. Call your existing service logic
	req := domain.GetUserRequest{Id: uint(id)}
	resp, err := h.Service.GetUser(req)
	if err != nil {
		// Ideally check if error is "record not found" to return 404
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// 3. Write JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
