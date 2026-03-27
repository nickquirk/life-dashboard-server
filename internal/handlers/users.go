package handlers

import (
	"encoding/json"
	"fmt"
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

func (h *Handler) GetUserHTTP(w http.ResponseWriter, r *http.Request) {
	// get ID from URL param
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.respondWithError(w, "Invalid user ID", err, http.StatusBadRequest)
		return
	}

	// Call service logic directly
	req := domain.GetUserRequest{ID: uint(id)}
	resp, err := h.Service.GetUser(req)
	if err != nil {
		h.respondWithError(w, "User not found", err, http.StatusNotFound)
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
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	// Fetch User Details directly from Service
	req := domain.GetUserRequest{ID: userID}
	resp, err := h.Service.GetUser(req)
	if err != nil {
		h.respondWithError(w, "Failed to fetch user profile", err, http.StatusInternalServerError, "userID", userID)
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
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(domain.GetCurrentUserIDResponse{
		ID: userID,
	})
}

func (h *Handler) deleteAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found in context", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	if err := h.Service.DeleteAccount(userID); err != nil {
		h.respondWithError(w, "Failed to delete account", err, http.StatusInternalServerError, "userID", userID)
		return
	}

	// Expire cookies exactly like the logout handler does
	http.SetCookie(w, h.Cookies.ExpireSessionCookie())
	http.SetCookie(w, h.Cookies.ExpireRefreshCookie())

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Account and all data deleted"}`))
}
