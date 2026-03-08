package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/service"
)

type Handler struct {
	Service service.Service
	Cookies CookieConfig
}

func (h *Handler) CreateUser(user domain.CreateUserRequest) (domain.CreateUserResponse, error) {
	resp, err := h.Service.CreateUser(user)
	if err != nil {
		slog.Error("error creating user", "error", err)
		return domain.CreateUserResponse{}, errors.New("error creating user")
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

	// Call your existing service logic
	req := domain.GetUserRequest{Id: uint(id)}
	resp, err := h.Service.GetUser(req)
	if err != nil {
		// Ideally check if error is "record not found" to return 404
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Write JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(domain.UserProfileResponse{
		Email:   resp.Email,
		Picture: resp.Picture,
	})
}

func (h *Handler) getCurrentUser(w http.ResponseWriter, r *http.Request) {
	// Get UserID from context (set by Authenticate middleware)
	userID, ok := h.GetUserID(r)
	if !ok {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Fetch User Details
	req := domain.GetUserRequest{Id: userID}
	resp, err := h.Service.GetUser(req)
	if err != nil {
		http.Error(w, "Failed to fetch user profile", http.StatusInternalServerError)
		return
	}

	// Return JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(domain.UserProfileResponse{
		Email:   resp.Email,
		Picture: resp.Picture,
	})
}

func (h *Handler) getCurrentUserID(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(domain.GetCurrentUserIDResponse{
		Id: userID,
	})
}

func (h *Handler) deleteAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	if err := h.Service.DeleteAccount(userID); err != nil {
		http.Error(w, "Failed to delete account", http.StatusInternalServerError)
		return
	}

	// Expire cookies exactly like the logout handler does
	http.SetCookie(w, h.Cookies.ExpireSessionCookie())
	http.SetCookie(w, h.Cookies.ExpireRefreshCookie())

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Account and all data deleted"}`))
}
