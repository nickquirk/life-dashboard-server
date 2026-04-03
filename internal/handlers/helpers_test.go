package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nickquirk/life-dashboard-server/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
)

func TestRespondWithError_ReturnsSanitizedMessage(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	rr := httptest.NewRecorder()

	h.respondWithError(rr, "Something went wrong", errors.New("secret db error"), http.StatusInternalServerError)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Equal(t, "Something went wrong\n", rr.Body.String())
}

func TestRespondWithError_SetsStatusCode(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{"bad request", http.StatusBadRequest},
		{"unauthorized", http.StatusUnauthorized},
		{"not found", http.StatusNotFound},
		{"internal server error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := testHandler(&mocks.MockService{})
			rr := httptest.NewRecorder()

			h.respondWithError(rr, "error", errors.New("internal"), tt.status)

			assert.Equal(t, tt.status, rr.Code)
		})
	}
}

func TestRespondWithError_DoesNotLeakInternalError(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	rr := httptest.NewRecorder()

	h.respondWithError(rr, "User not found", errors.New("pq: relation \"users\" does not exist"), http.StatusNotFound)

	assert.NotContains(t, rr.Body.String(), "pq:")
	assert.Contains(t, rr.Body.String(), "User not found")
}

func TestRespondWithError_AcceptsExtraLogArgs(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	rr := httptest.NewRecorder()

	// Should not panic when extra args are passed
	h.respondWithError(rr, "Failed", errors.New("err"), http.StatusInternalServerError, "userID", uint(1), "taskID", "abc")

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestRespondWithJSON_ReturnsJSON(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	rr := httptest.NewRecorder()

	payload := map[string]string{"message": "ok"}
	h.respondWithJSON(rr, http.StatusOK, payload)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var body map[string]string
	err := json.NewDecoder(rr.Body).Decode(&body)
	assert.NoError(t, err)
	assert.Equal(t, "ok", body["message"])
}

func TestRespondWithJSON_SetsStatusCode(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{"200 OK", http.StatusOK},
		{"201 Created", http.StatusCreated},
		{"204 No Content", http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := testHandler(&mocks.MockService{})
			rr := httptest.NewRecorder()

			h.respondWithJSON(rr, tt.status, map[string]string{})

			assert.Equal(t, tt.status, rr.Code)
		})
	}
}

func TestRespondWithJSON_EncodesStruct(t *testing.T) {
	type response struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	h := testHandler(&mocks.MockService{})
	rr := httptest.NewRecorder()

	h.respondWithJSON(rr, http.StatusOK, response{ID: 42, Name: "test"})

	var body response
	err := json.NewDecoder(rr.Body).Decode(&body)
	assert.NoError(t, err)
	assert.Equal(t, 42, body.ID)
	assert.Equal(t, "test", body.Name)
}

func TestRespondWithJSON_AcceptsExtraLogArgs(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	rr := httptest.NewRecorder()

	// Should not panic when extra args are passed
	h.respondWithJSON(rr, http.StatusOK, map[string]string{"ok": "true"}, "userID", uint(1))

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRespondWithJSON_EncodesNilAsNull(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	rr := httptest.NewRecorder()

	h.respondWithJSON(rr, http.StatusOK, nil)

	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	assert.Equal(t, "null\n", rr.Body.String())
}
