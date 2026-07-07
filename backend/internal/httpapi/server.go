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
	Documents *service.Documents
	CaseFiles *service.CaseFiles
	Drawings  *service.Drawings
	Checklist *service.Compliance
	Sources   *service.Sources
	Assistant  *service.Assistant
	Generator  *service.Generator
	Structural *service.Structural
	Packages   *service.Packages
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

	mux.HandleFunc("GET /api/v1/projects/{projectID}/documents", s.listDocuments)
	mux.HandleFunc("POST /api/v1/projects/{projectID}/documents", s.uploadDocument)
	mux.HandleFunc("GET /api/v1/documents/{documentID}", getUnder(s.svc.Documents.Get, "documentID"))
	mux.HandleFunc("GET /api/v1/documents/{documentID}/content", s.documentContent)
	mux.HandleFunc("PATCH /api/v1/documents/{documentID}", patchByID(s.svc.Documents.Update, "documentID"))
	mux.HandleFunc("DELETE /api/v1/documents/{documentID}", deleteByID(s.svc.Documents.Delete, "documentID"))
	mux.HandleFunc("PUT /api/v1/documents/{documentID}/tags", s.setDocumentTags)
	mux.HandleFunc("GET /api/v1/documents/{documentID}/links", getUnder(s.svc.Documents.ListLinks, "documentID"))
	mux.HandleFunc("POST /api/v1/documents/{documentID}/links", createUnder(s.svc.Documents.AddLink, "documentID"))
	mux.HandleFunc("DELETE /api/v1/documents/{documentID}/links/{linkID}", s.removeDocumentLink)

	mux.HandleFunc("GET /api/v1/projects/{projectID}/case-files", getUnder(s.svc.CaseFiles.List, "projectID"))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/case-files", createUnder(s.svc.CaseFiles.Create, "projectID"))
	mux.HandleFunc("GET /api/v1/case-files/{caseFileID}", getUnder(s.svc.CaseFiles.Get, "caseFileID"))
	mux.HandleFunc("PATCH /api/v1/case-files/{caseFileID}", patchByID(s.svc.CaseFiles.Update, "caseFileID"))
	mux.HandleFunc("DELETE /api/v1/case-files/{caseFileID}", deleteByID(s.svc.CaseFiles.Delete, "caseFileID"))
	mux.HandleFunc("GET /api/v1/case-files/{caseFileID}/events", getUnder(s.svc.CaseFiles.ListEvents, "caseFileID"))
	mux.HandleFunc("POST /api/v1/case-files/{caseFileID}/events", createUnder(s.svc.CaseFiles.AddEvent, "caseFileID"))

	mux.HandleFunc("GET /api/v1/projects/{projectID}/drawings", getUnder(s.svc.Drawings.List, "projectID"))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/drawings", createUnder(s.svc.Drawings.Create, "projectID"))
	mux.HandleFunc("GET /api/v1/drawings/{drawingID}", getUnder(s.svc.Drawings.Get, "drawingID"))
	mux.HandleFunc("PATCH /api/v1/drawings/{drawingID}", patchByID(s.svc.Drawings.Update, "drawingID"))
	mux.HandleFunc("DELETE /api/v1/drawings/{drawingID}", deleteByID(s.svc.Drawings.Delete, "drawingID"))
	mux.HandleFunc("GET /api/v1/drawings/{drawingID}/versions", getUnder(s.svc.Drawings.ListVersions, "drawingID"))
	mux.HandleFunc("POST /api/v1/drawings/{drawingID}/versions", createUnder(s.svc.Drawings.AddVersion, "drawingID"))

	mux.HandleFunc("GET /api/v1/case-files/{caseFileID}/checklist", getUnder(s.svc.Checklist.List, "caseFileID"))
	mux.HandleFunc("POST /api/v1/case-files/{caseFileID}/checklist", createUnder(s.svc.Checklist.Create, "caseFileID"))
	mux.HandleFunc("PATCH /api/v1/checklist-items/{itemID}", patchByID(s.svc.Checklist.Update, "itemID"))
	mux.HandleFunc("DELETE /api/v1/checklist-items/{itemID}", deleteByID(s.svc.Checklist.Delete, "itemID"))

	mux.HandleFunc("GET /api/v1/projects/{projectID}/sources", getUnder(s.svc.Sources.List, "projectID"))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/sources", createUnder(s.svc.Sources.Ingest, "projectID"))
	mux.HandleFunc("DELETE /api/v1/sources/{sourceID}", deleteByID(s.svc.Sources.Delete, "sourceID"))
	mux.HandleFunc("GET /api/v1/projects/{projectID}/sources/search", s.searchSources)

	mux.HandleFunc("POST /api/v1/projects/{projectID}/assistant/ask", createUnder(s.svc.Assistant.Ask, "projectID"))

	mux.HandleFunc("GET /api/v1/case-files/{caseFileID}/generated", getUnder(s.svc.Generator.List, "caseFileID"))
	mux.HandleFunc("POST /api/v1/case-files/{caseFileID}/generate", createUnder(s.svc.Generator.Generate, "caseFileID"))

	mux.HandleFunc("GET /api/v1/projects/{projectID}/structural-elements", getUnder(s.svc.Structural.ListElements, "projectID"))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/structural-elements", createUnder(s.svc.Structural.CreateElement, "projectID"))
	mux.HandleFunc("PATCH /api/v1/structural-elements/{elementID}", patchByID(s.svc.Structural.UpdateElement, "elementID"))
	mux.HandleFunc("DELETE /api/v1/structural-elements/{elementID}", deleteByID(s.svc.Structural.DeleteElement, "elementID"))

	mux.HandleFunc("GET /api/v1/projects/{projectID}/loads", getUnder(s.svc.Structural.ListLoads, "projectID"))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/loads", createUnder(s.svc.Structural.CreateLoad, "projectID"))
	mux.HandleFunc("PATCH /api/v1/loads/{loadID}", patchByID(s.svc.Structural.UpdateLoad, "loadID"))
	mux.HandleFunc("DELETE /api/v1/loads/{loadID}", deleteByID(s.svc.Structural.DeleteLoad, "loadID"))

	mux.HandleFunc("GET /api/v1/calc/methods", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, s.svc.Structural.Methods())
	})
	mux.HandleFunc("GET /api/v1/projects/{projectID}/estimates", getUnder(s.svc.Structural.ListEstimates, "projectID"))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/estimates", createUnder(s.svc.Structural.RunEstimate, "projectID"))

	mux.HandleFunc("GET /api/v1/projects/{projectID}/structural-packages", getUnder(s.svc.Packages.List, "projectID"))
	mux.HandleFunc("POST /api/v1/projects/{projectID}/structural-packages", createUnder(s.svc.Packages.Create, "projectID"))
	mux.HandleFunc("PATCH /api/v1/structural-packages/{packageID}/status", patchByID(s.svc.Packages.SetStatus, "packageID"))
	mux.HandleFunc("GET /api/v1/structural-packages/{packageID}/reviews", getUnder(s.svc.Packages.ListReviews, "packageID"))
	mux.HandleFunc("POST /api/v1/structural-packages/{packageID}/reviews", createUnder(s.svc.Packages.AddReview, "packageID"))
}
