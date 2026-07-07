// Package httpapi wires services to HTTP/JSON endpoints. Handlers only
// decode requests, call a service, and encode the result — no business
// logic lives here.
package httpapi

import (
	"net/http"

	"github.com/balu-dk/hefai/backend/internal/auth"
	"github.com/balu-dk/hefai/backend/internal/service"
)

type Services struct {
	Auth     *service.Auth
	Projects *service.Projects
	Phases   *service.Phases
	Tasks    *service.Tasks
}

type Server struct {
	svc    Services
	tokens *auth.TokenIssuer
}

// New builds the full HTTP handler with routing and middleware.
func New(svc Services, tokens *auth.TokenIssuer, corsOrigin string) http.Handler {
	s := &Server{svc: svc, tokens: tokens}

	public := http.NewServeMux()
	public.HandleFunc("GET /api/v1/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	public.HandleFunc("POST /api/v1/auth/register", s.register)
	public.HandleFunc("POST /api/v1/auth/login", s.login)

	protected := http.NewServeMux()
	s.routes(protected)
	public.Handle("/api/v1/", requireAuth(tokens, protected))

	return recoverPanics(logRequests(cors(corsOrigin, public)))
}

func (s *Server) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/me", s.me)

	mux.HandleFunc("GET /api/v1/projects", s.listProjects)
	mux.HandleFunc("POST /api/v1/projects", s.createProject)
	mux.HandleFunc("GET /api/v1/projects/{projectID}", s.getProject)
	mux.HandleFunc("PATCH /api/v1/projects/{projectID}", s.updateProject)
	mux.HandleFunc("DELETE /api/v1/projects/{projectID}", s.deleteProject)

	mux.HandleFunc("GET /api/v1/projects/{projectID}/members", s.listMembers)
	mux.HandleFunc("POST /api/v1/projects/{projectID}/members", s.addMember)
	mux.HandleFunc("DELETE /api/v1/projects/{projectID}/members/{userID}", s.removeMember)

	mux.HandleFunc("GET /api/v1/projects/{projectID}/phases", s.listPhases)
	mux.HandleFunc("POST /api/v1/projects/{projectID}/phases", s.createPhase)
	mux.HandleFunc("PATCH /api/v1/phases/{phaseID}", s.updatePhase)
	mux.HandleFunc("DELETE /api/v1/phases/{phaseID}", s.deletePhase)

	mux.HandleFunc("GET /api/v1/projects/{projectID}/tasks", s.listTasks)
	mux.HandleFunc("GET /api/v1/projects/{projectID}/tasks/board", s.taskBoard)
	mux.HandleFunc("POST /api/v1/projects/{projectID}/tasks", s.createTask)
	mux.HandleFunc("GET /api/v1/tasks/{taskID}", s.getTask)
	mux.HandleFunc("PATCH /api/v1/tasks/{taskID}", s.updateTask)
	mux.HandleFunc("DELETE /api/v1/tasks/{taskID}", s.deleteTask)
	mux.HandleFunc("POST /api/v1/tasks/{taskID}/dependencies", s.addTaskDependency)
	mux.HandleFunc("DELETE /api/v1/tasks/{taskID}/dependencies/{dependsOnID}", s.removeTaskDependency)
}
