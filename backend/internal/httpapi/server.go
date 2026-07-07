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
	Auth      *service.Auth
	Projects  *service.Projects
	Phases    *service.Phases
	Tasks     *service.Tasks
	Rooms     *service.Rooms
	Suppliers *service.Suppliers
	Budget    *service.Budget
	Materials *service.Materials
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

	mux.HandleFunc("GET /api/v1/projects/{projectID}/rooms", getUnder(s.svc.Rooms.List, "projectID"))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/rooms", createUnder(s.svc.Rooms.Create, "projectID"))
	mux.HandleFunc("PATCH /api/v1/rooms/{roomID}", patchByID(s.svc.Rooms.Update, "roomID"))
	mux.HandleFunc("DELETE /api/v1/rooms/{roomID}", deleteByID(s.svc.Rooms.Delete, "roomID"))

	mux.HandleFunc("GET /api/v1/projects/{projectID}/suppliers", getUnder(s.svc.Suppliers.List, "projectID"))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/suppliers", createUnder(s.svc.Suppliers.Create, "projectID"))
	mux.HandleFunc("PATCH /api/v1/suppliers/{supplierID}", patchByID(s.svc.Suppliers.Update, "supplierID"))
	mux.HandleFunc("DELETE /api/v1/suppliers/{supplierID}", deleteByID(s.svc.Suppliers.Delete, "supplierID"))

	mux.HandleFunc("GET /api/v1/projects/{projectID}/budget-items", getUnder(s.svc.Budget.ListItems, "projectID"))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/budget-items", createUnder(s.svc.Budget.CreateItem, "projectID"))
	mux.HandleFunc("PATCH /api/v1/budget-items/{itemID}", patchByID(s.svc.Budget.UpdateItem, "itemID"))
	mux.HandleFunc("DELETE /api/v1/budget-items/{itemID}", deleteByID(s.svc.Budget.DeleteItem, "itemID"))

	mux.HandleFunc("GET /api/v1/projects/{projectID}/expenses", getUnder(s.svc.Budget.ListExpenses, "projectID"))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/expenses", createUnder(s.svc.Budget.CreateExpense, "projectID"))
	mux.HandleFunc("PATCH /api/v1/expenses/{expenseID}", patchByID(s.svc.Budget.UpdateExpense, "expenseID"))
	mux.HandleFunc("DELETE /api/v1/expenses/{expenseID}", deleteByID(s.svc.Budget.DeleteExpense, "expenseID"))

	mux.HandleFunc("GET /api/v1/projects/{projectID}/budget/summary", getUnder(s.svc.Budget.Summary, "projectID"))

	mux.HandleFunc("GET /api/v1/projects/{projectID}/materials", getUnder(s.svc.Materials.List, "projectID"))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/materials", createUnder(s.svc.Materials.Create, "projectID"))
	mux.HandleFunc("PATCH /api/v1/materials/{materialID}", patchByID(s.svc.Materials.Update, "materialID"))
	mux.HandleFunc("DELETE /api/v1/materials/{materialID}", deleteByID(s.svc.Materials.Delete, "materialID"))
	mux.HandleFunc("GET /api/v1/projects/{projectID}/materials/shopping-list", getUnder(s.svc.Materials.ShoppingList, "projectID"))
}
