package httpapi

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/balu-dk/hefai/backend/internal/service"
)

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	projectID, err := pathUUID(r, "projectID")
	if err != nil {
		writeError(w, err)
		return
	}
	tasks, err := s.svc.Tasks.List(r.Context(), userID(r), projectID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (s *Server) taskBoard(w http.ResponseWriter, r *http.Request) {
	projectID, err := pathUUID(r, "projectID")
	if err != nil {
		writeError(w, err)
		return
	}
	board, err := s.svc.Tasks.Board(r.Context(), userID(r), projectID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, board)
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	projectID, err := pathUUID(r, "projectID")
	if err != nil {
		writeError(w, err)
		return
	}
	var in service.TaskPatch
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	task, err := s.svc.Tasks.Create(r.Context(), userID(r), projectID, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

func (s *Server) getTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := pathUUID(r, "taskID")
	if err != nil {
		writeError(w, err)
		return
	}
	task, err := s.svc.Tasks.Get(r.Context(), userID(r), taskID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) updateTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := pathUUID(r, "taskID")
	if err != nil {
		writeError(w, err)
		return
	}
	var in service.TaskPatch
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	task, err := s.svc.Tasks.Update(r.Context(), userID(r), taskID, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) deleteTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := pathUUID(r, "taskID")
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.svc.Tasks.Delete(r.Context(), userID(r), taskID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (s *Server) addTaskDependency(w http.ResponseWriter, r *http.Request) {
	taskID, err := pathUUID(r, "taskID")
	if err != nil {
		writeError(w, err)
		return
	}
	var in struct {
		DependsOnTaskID uuid.UUID `json:"dependsOnTaskId"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	if err := s.svc.Tasks.AddDependency(r.Context(), userID(r), taskID, in.DependsOnTaskID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (s *Server) removeTaskDependency(w http.ResponseWriter, r *http.Request) {
	taskID, err := pathUUID(r, "taskID")
	if err != nil {
		writeError(w, err)
		return
	}
	dependsOnID, err := pathUUID(r, "dependsOnID")
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.svc.Tasks.RemoveDependency(r.Context(), userID(r), taskID, dependsOnID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}
