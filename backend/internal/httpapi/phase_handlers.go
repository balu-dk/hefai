package httpapi

import (
	"net/http"

	"github.com/balu-dk/hefai/backend/internal/service"
)

func (s *Server) listPhases(w http.ResponseWriter, r *http.Request) {
	projectID, err := pathUUID(r, "projectID")
	if err != nil {
		writeError(w, err)
		return
	}
	phases, err := s.svc.Phases.List(r.Context(), userID(r), projectID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, phases)
}

func (s *Server) createPhase(w http.ResponseWriter, r *http.Request) {
	projectID, err := pathUUID(r, "projectID")
	if err != nil {
		writeError(w, err)
		return
	}
	var in service.PhasePatch
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	phase, err := s.svc.Phases.Create(r.Context(), userID(r), projectID, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, phase)
}

func (s *Server) updatePhase(w http.ResponseWriter, r *http.Request) {
	phaseID, err := pathUUID(r, "phaseID")
	if err != nil {
		writeError(w, err)
		return
	}
	var in service.PhasePatch
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	phase, err := s.svc.Phases.Update(r.Context(), userID(r), phaseID, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, phase)
}

func (s *Server) deletePhase(w http.ResponseWriter, r *http.Request) {
	phaseID, err := pathUUID(r, "phaseID")
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.svc.Phases.Delete(r.Context(), userID(r), phaseID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}
