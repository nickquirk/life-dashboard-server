package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/service"
)

func createUser(w http.ResponseWriter, r *http.Request, s service.Service) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "failed to read request body\n")
		return
	}

	user := domain.CreateUserRequest{}

	err = json.Unmarshal(body, &user)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "failed to parse JSON\n")
		return
	}

	createUserDto, err := s.CreateUser(user)
	response, _ := json.Marshal(domain.CreateUserResponse{Id: createUserDto.Id})

	if err != nil {
		io.WriteString(w, err.Error())
		return
	}
	w.Write(response)
}
