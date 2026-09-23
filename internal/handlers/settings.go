package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

// The settings document is small; cap the body so a bad client can't stream
// megabytes into the JSON decoder.
const maxSettingsBodyBytes = 32 * 1024

func (h *Handler) getUserSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "Unauthorized", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	resp, err := h.Service.GetUserSettings(domain.GetUserSettingsRequest{UserID: userID})
	if err != nil {
		h.respondWithError(w, "Failed to fetch settings", err, http.StatusInternalServerError, "userID", userID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID)
}

func (h *Handler) updateUserSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "Unauthorized", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxSettingsBodyBytes))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			h.respondWithError(w, "Request body too large", err, http.StatusRequestEntityTooLarge, "userID", userID)
			return
		}
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}

	resp, err := h.Service.UpdateUserSettings(domain.UpdateUserSettingsRequest{
		UserID: userID,
		Patch:  body,
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			h.respondWithError(w, err.Error(), err, http.StatusBadRequest, "userID", userID)
			return
		}
		h.respondWithError(w, "Failed to save settings", err, http.StatusInternalServerError, "userID", userID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID)
}
