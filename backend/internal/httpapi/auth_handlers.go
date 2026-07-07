package httpapi

import (
	"net/http"

	"github.com/balu-dk/hefai/backend/internal/service"
)

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var in service.Credentials
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	result, err := s.svc.Auth.Register(r.Context(), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in service.Credentials
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	result, err := s.svc.Auth.Login(r.Context(), in.Email, in.Password)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	user, err := s.svc.Auth.GetUser(r.Context(), userID(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}
