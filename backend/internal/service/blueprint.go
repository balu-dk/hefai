package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/balu-dk/hefai/backend/internal/ai"
	"github.com/balu-dk/hefai/backend/internal/domain"
)

// Blueprints implementerer AI-projektstarten: interviewsvar → struktureret
// plan (blueprint) → oprettelse af rum, opgaver, budget og materialer.
// Med LLM-provider genereres planen af modellen efter den redigerbare
// instruktionsfil i ai-docs/; uden provider bygges en deterministisk
// skabelon af svarene. Begge veje validerer det samme før noget oprettes.
type Blueprints struct {
	provider  ai.Provider
	docsDir   string
	projects  ProjectRepo
	phases    PhaseRepo
	rooms     RoomRepo
	tasks     TaskRepo
	budget    BudgetRepo
	materials MaterialRepo
	caseFiles CaseFileRepo
	access    ProjectAccess
}

func NewBlueprints(provider ai.Provider, docsDir string, projects ProjectRepo, phases PhaseRepo,
	rooms RoomRepo, tasks TaskRepo, budget BudgetRepo, materials MaterialRepo,
	caseFiles CaseFileRepo, access ProjectAccess) *Blueprints {
	return &Blueprints{
		provider: provider, docsDir: docsDir, projects: projects, phases: phases, rooms: rooms,
		tasks: tasks, budget: budget, materials: materials, caseFiles: caseFiles, access: access,
	}
}

// Interview er svarene fra "grill mig"-wizarden. Generisk for alle
// projekttyper — nybyggeri, renovering, tilbygning m.m.
type Interview struct {
	Goal         string   `json:"goal"`         // hvad skal der ske, fritekst
	PropertyType string   `json:"propertyType"` // sommerhus, villa, rækkehus, lejlighed, andet
	SizeM2       float64  `json:"sizeM2"`
	Rooms        []string `json:"rooms"`
	Features     []string `json:"features"` // terrasse, carport, nyt køkken …
	BudgetOre    int64    `json:"budgetOre"`
	SelfBuild    string   `json:"selfBuild"` // self, mixed, contractors
	Timeline     string   `json:"timeline"`
	Notes        string   `json:"notes"`
}

type BlueprintRoom struct {
	Name   string   `json:"name"`
	Kind   string   `json:"kind"`
	AreaM2 *float64 `json:"areaM2"`
}

type BlueprintTask struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Phase       string `json:"phase"`
	DependsOn   []int  `json:"dependsOn"`
}

type BlueprintBudgetItem struct {
	Description        string `json:"description"`
	Category           string `json:"category"`
	Phase              string `json:"phase"`
	EstimatedAmountOre int64  `json:"estimatedAmountOre"`
}

type BlueprintMaterial struct {
	Name     string  `json:"name"`
	Spec     string  `json:"spec"`
	Quantity float64 `json:"quantity"`
	Unit     string  `json:"unit"`
	Phase    string  `json:"phase"`
}

type Blueprint struct {
	ProjectDescription string                `json:"projectDescription"`
	CaseDescription    string                `json:"caseDescription"`
	NeedsBuildingCase  bool                  `json:"needsBuildingCase"`
	Rooms              []BlueprintRoom       `json:"rooms"`
	Tasks              []BlueprintTask       `json:"tasks"`
	BudgetItems        []BlueprintBudgetItem `json:"budgetItems"`
	Materials          []BlueprintMaterial   `json:"materials"`
	Notes              string                `json:"notes"`
}

type BlueprintResult struct {
	Blueprint Blueprint `json:"blueprint"`
	Source    string    `json:"source"` // "llm" eller "template"
	Provider  string    `json:"provider"`
	Notice    string    `json:"notice,omitempty"`
}

// Generate producerer et blueprint-udkast ud fra interviewet. Intet oprettes
// før brugeren godkender via Apply.
func (s *Blueprints) Generate(ctx context.Context, userID, projectID uuid.UUID, interview Interview) (*BlueprintResult, error) {
	if err := requireWrite(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(interview.Goal) == "" {
		return nil, domain.Validation("beskriv hvad der skal ske — det er udgangspunktet for planen")
	}
	project, err := s.projects.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}

	result := &BlueprintResult{Provider: s.provider.Name()}

	instructions, err := os.ReadFile(filepath.Join(s.docsDir, "blueprint.md"))
	if err != nil {
		instructions = nil // uden instruktionsfil bruges skabelonen
	}

	if instructions != nil {
		answersJSON, _ := json.MarshalIndent(interview, "", "  ")
		prompt := fmt.Sprintf("PROJEKT:\nNavn: %s\nType: %s\nAdresse: %s\nKommune: %s\nGrundareal: %v m²\n\nINTERVIEWSVAR:\n%s",
			project.Name, project.Kind, project.Address, project.Municipality,
			plotAreaOrDash(project.PlotAreaM2), answersJSON)

		raw, llmErr := s.provider.Complete(ctx, ai.Request{
			System:   string(instructions),
			Messages: []ai.Message{{Role: "user", Content: prompt}},
		})
		if llmErr == nil {
			blueprint, parseErr := parseBlueprint(raw)
			if parseErr == nil {
				result.Blueprint = *blueprint
				result.Source = "llm"
				return result, nil
			}
			result.Notice = "Modellens svar kunne ikke læses (" + parseErr.Error() + ") — der vises i stedet en standardplan."
		} else if !errors.Is(llmErr, ai.ErrNotConfigured) {
			result.Notice = "LLM-kaldet fejlede (" + llmErr.Error() + ") — der vises i stedet en standardplan."
		} else {
			result.Notice = "Ingen LLM-provider er sat op endnu, så planen er bygget af Hefais standardskabelon. " +
				"Sæt LLM_BASE_URL/LLM_API_KEY for at få planer skræddersyet af en model."
		}
	}

	result.Blueprint = templateBlueprint(project, interview)
	result.Source = "template"
	normalizeBlueprint(&result.Blueprint)
	return result, nil
}

// normalizeBlueprint sikrer at alle lister er tomme frem for nil, så JSON
// aldrig indeholder null-arrays.
func normalizeBlueprint(b *Blueprint) {
	if b.Rooms == nil {
		b.Rooms = []BlueprintRoom{}
	}
	if b.Tasks == nil {
		b.Tasks = []BlueprintTask{}
	}
	if b.BudgetItems == nil {
		b.BudgetItems = []BlueprintBudgetItem{}
	}
	if b.Materials == nil {
		b.Materials = []BlueprintMaterial{}
	}
	for i := range b.Tasks {
		if b.Tasks[i].DependsOn == nil {
			b.Tasks[i].DependsOn = []int{}
		}
	}
}

// parseBlueprint læser modellens svar: JSON, evt. pakket i markdown-hegn.
func parseBlueprint(raw string) (*Blueprint, error) {
	text := strings.TrimSpace(raw)
	if idx := strings.Index(text, "```"); idx >= 0 {
		text = text[idx+3:]
		text = strings.TrimPrefix(text, "json")
		if end := strings.Index(text, "```"); end >= 0 {
			text = text[:end]
		}
	}
	// Find første { så evt. præambel ignoreres.
	if idx := strings.Index(text, "{"); idx > 0 {
		text = text[idx:]
	}
	var b Blueprint
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &b); err != nil {
		return nil, fmt.Errorf("ugyldig JSON: %w", err)
	}
	if err := validateBlueprint(&b); err != nil {
		return nil, err
	}
	normalizeBlueprint(&b)
	return &b, nil
}

var validPhaseNames = func() map[string]bool {
	m := map[string]bool{}
	for _, name := range domain.DefaultPhaseNames {
		m[name] = true
	}
	return m
}()

func validateBlueprint(b *Blueprint) error {
	if len(b.Tasks) == 0 {
		return errors.New("planen indeholder ingen opgaver")
	}
	if len(b.Tasks) > 60 || len(b.Rooms) > 40 || len(b.BudgetItems) > 30 || len(b.Materials) > 80 {
		return errors.New("planen er urealistisk stor")
	}
	for i, t := range b.Tasks {
		if strings.TrimSpace(t.Title) == "" {
			return fmt.Errorf("opgave %d mangler titel", i)
		}
		if t.Phase != "" && !validPhaseNames[t.Phase] {
			b.Tasks[i].Phase = "" // ukendt fasenavn: opret uden fase frem for at fejle
		}
		for _, dep := range t.DependsOn {
			if dep < 0 || dep >= len(b.Tasks) || dep == i {
				return fmt.Errorf("opgave %d har ugyldig afhængighed %d", i, dep)
			}
		}
	}
	if hasCycle(b.Tasks) {
		return errors.New("opgavernes afhængigheder danner en cirkel")
	}
	for i, item := range b.BudgetItems {
		if item.EstimatedAmountOre < 0 {
			return fmt.Errorf("budgetpost %d har negativt beløb", i)
		}
		if item.Phase != "" && !validPhaseNames[item.Phase] {
			b.BudgetItems[i].Phase = ""
		}
	}
	for i, m := range b.Materials {
		if m.Quantity < 0 {
			return fmt.Errorf("materiale %d har negativt antal", i)
		}
		if m.Phase != "" && !validPhaseNames[m.Phase] {
			b.Materials[i].Phase = ""
		}
	}
	for i, r := range b.Rooms {
		if strings.TrimSpace(r.Name) == "" {
			return fmt.Errorf("rum %d mangler navn", i)
		}
		switch r.Kind {
		case "room", "zone", "outdoor":
		default:
			b.Rooms[i].Kind = "room"
		}
	}
	return nil
}

func hasCycle(tasks []BlueprintTask) bool {
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make([]int, len(tasks))
	var visit func(int) bool
	visit = func(node int) bool {
		color[node] = gray
		for _, dep := range tasks[node].DependsOn {
			if color[dep] == gray {
				return true
			}
			if color[dep] == white && visit(dep) {
				return true
			}
		}
		color[node] = black
		return false
	}
	for i := range tasks {
		if color[i] == white && visit(i) {
			return true
		}
	}
	return false
}

// ApplyResult opsummerer hvad der blev oprettet.
type ApplyResult struct {
	RoomsCreated       int        `json:"roomsCreated"`
	TasksCreated       int        `json:"tasksCreated"`
	DependenciesLinked int        `json:"dependenciesLinked"`
	BudgetItemsCreated int        `json:"budgetItemsCreated"`
	MaterialsCreated   int        `json:"materialsCreated"`
	CaseFileID         *uuid.UUID `json:"caseFileId"`
}

// Apply opretter blueprintets indhold i projektet. Brugeren har set og evt.
// redigeret planen først.
func (s *Blueprints) Apply(ctx context.Context, userID, projectID uuid.UUID, b Blueprint) (*ApplyResult, error) {
	if err := requireWrite(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	if err := validateBlueprint(&b); err != nil {
		return nil, domain.Validation(err.Error())
	}
	project, err := s.projects.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Fasenavn → id for det aktuelle projekt.
	phaseByName := map[string]uuid.UUID{}
	phases, err := s.phases.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for _, p := range phases {
		phaseByName[p.Name] = p.ID
	}
	phaseID := func(name string) *uuid.UUID {
		if id, ok := phaseByName[name]; ok {
			return &id
		}
		return nil
	}

	result := &ApplyResult{}

	for _, r := range b.Rooms {
		room := &domain.Room{ProjectID: projectID, Name: r.Name, Kind: domain.RoomKind(r.Kind), AreaM2: r.AreaM2}
		if _, err := s.rooms.Create(ctx, room); err != nil {
			return nil, fmt.Errorf("kunne ikke oprette rum %q: %w", r.Name, err)
		}
		result.RoomsCreated++
	}

	taskIDs := make([]uuid.UUID, len(b.Tasks))
	for i, t := range b.Tasks {
		created, err := s.tasks.Create(ctx, &domain.Task{
			ProjectID:   projectID,
			PhaseID:     phaseID(t.Phase),
			Title:       t.Title,
			Description: t.Description,
			Status:      domain.TaskTodo,
		})
		if err != nil {
			return nil, fmt.Errorf("kunne ikke oprette opgave %q: %w", t.Title, err)
		}
		taskIDs[i] = created.ID
		result.TasksCreated++
	}
	for i, t := range b.Tasks {
		for _, dep := range t.DependsOn {
			if err := s.tasks.AddDependency(ctx, taskIDs[i], taskIDs[dep]); err != nil {
				return nil, err
			}
			result.DependenciesLinked++
		}
	}

	for _, item := range b.BudgetItems {
		if _, err := s.budget.CreateItem(ctx, &domain.BudgetItem{
			ProjectID:          projectID,
			PhaseID:            phaseID(item.Phase),
			Category:           item.Category,
			Description:        item.Description,
			EstimatedAmountOre: item.EstimatedAmountOre,
			Currency:           "DKK",
		}); err != nil {
			return nil, err
		}
		result.BudgetItemsCreated++
	}

	for _, m := range b.Materials {
		if _, err := s.materials.Create(ctx, &domain.Material{
			ProjectID: projectID,
			PhaseID:   phaseID(m.Phase),
			Name:      m.Name,
			Spec:      m.Spec,
			Quantity:  m.Quantity,
			Unit:      orDefault(m.Unit, "stk"),
			Status:    domain.MaterialNeeded,
			Currency:  "DKK",
		}); err != nil {
			return nil, err
		}
		result.MaterialsCreated++
	}

	if b.NeedsBuildingCase {
		caseFile, err := s.caseFiles.Create(ctx, &domain.CaseFile{
			ProjectID:   projectID,
			Title:       "Byggesag — " + project.Name,
			Description: b.CaseDescription,
			CaseType:    domain.CaseTypeUnknown,
			Status:      domain.CaseDraft,
		})
		if err != nil {
			return nil, err
		}
		result.CaseFileID = &caseFile.ID
	}

	if strings.TrimSpace(project.Description) == "" && b.ProjectDescription != "" {
		project.Description = b.ProjectDescription
		_, _ = s.projects.Update(ctx, project)
	}
	return result, nil
}

func orDefault(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func plotAreaOrDash(v *float64) any {
	if v == nil {
		return "ukendt"
	}
	return *v
}
