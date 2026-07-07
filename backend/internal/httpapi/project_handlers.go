package httpapi

import (
	"net/http"

	"github.com/balu-dk/hefai/backend/internal/domain"
	"github.com/balu-dk/hefai/backend/internal/service"
)

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.svc.Projects.List(r.Context(), userID(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, projects)
}

func (s *Server) createProject(w http.ResponseWriter, r *http.Request) {
	var in service.ProjectInput
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	project, err := s.svc.Projects.Create(r.Context(), userID(r), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, project)
}

func (s *Server) getProject(w http.ResponseWriter, r *http.Request) {
	projectID, err := pathUUID(r, "projectID")
	if err != nil {
		writeError(w, err)
		return
	}
	project, err := s.svc.Projects.Get(r.Context(), userID(r), projectID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (s *Server) updateProject(w http.ResponseWriter, r *http.Request) {
	projectID, err := pathUUID(r, "projectID")
	if err != nil {
		writeError(w, err)
		return
	}
	var in service.ProjectInput
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	project, err := s.svc.Projects.Update(r.Context(), userID(r), projectID, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (s *Server) deleteProject(w http.ResponseWriter, r *http.Request) {
	projectID, err := pathUUID(r, "projectID")
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.svc.Projects.Delete(r.Context(), userID(r), projectID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (s *Server) listMembers(w http.ResponseWriter, r *http.Request) {
	projectID, err := pathUUID(r, "projectID")
	if err != nil {
		writeError(w, err)
		return
	}
	members, err := s.svc.Projects.ListMembers(r.Context(), userID(r), projectID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, members)
}

func (s *Server) addMember(w http.ResponseWriter, r *http.Request) {
	projectID, err := pathUUID(r, "projectID")
	if err != nil {
		writeError(w, err)
		return
	}
	var in struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	err = s.svc.Projects.AddMember(r.Context(), userID(r), projectID, in.Email, domain.ProjectRole(in.Role))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (s *Server) removeMember(w http.ResponseWriter, r *http.Request) {
	projectID, err := pathUUID(r, "projectID")
	if err != nil {
		writeError(w, err)
		return
	}
	memberID, err := pathUUID(r, "userID")
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.svc.Projects.RemoveMember(r.Context(), userID(r), projectID, memberID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}
