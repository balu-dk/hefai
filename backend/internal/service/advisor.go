package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/balu-dk/hefai/backend/internal/ai"
	"github.com/balu-dk/hefai/backend/internal/domain"
)

// Advisor er projektrådgiveren på overblikket: deterministiske indsigter
// bygget af projektets faktiske tilstand, plus (når LLM er sat op) en kort
// statusvurdering formuleret af modellen ud fra samme øjebliksbillede —
// aldrig ud fra andet.
type Advisor struct {
	provider  ai.Provider
	docsDir   string
	phases    PhaseRepo
	tasks     TaskRepo
	budget    BudgetRepo
	materials MaterialRepo
	caseFiles CaseFileRepo
	drawings  DrawingRepo
	sources   SourceRepo
	access    ProjectAccess
}

func NewAdvisor(provider ai.Provider, docsDir string, phases PhaseRepo, tasks TaskRepo,
	budget BudgetRepo, materials MaterialRepo, caseFiles CaseFileRepo, drawings DrawingRepo,
	sources SourceRepo, access ProjectAccess) *Advisor {
	return &Advisor{provider: provider, docsDir: docsDir, phases: phases, tasks: tasks,
		budget: budget, materials: materials, caseFiles: caseFiles, drawings: drawings,
		sources: sources, access: access}
}

type Insight struct {
	Level  string `json:"level"` // "warning", "tip", "info"
	Text   string `json:"text"`
	LinkTo string `json:"linkTo"` // relativ sti i projektet, fx "budget"
}

type AdvisorResult struct {
	Insights []Insight `json:"insights"`
	Summary  string    `json:"summary,omitempty"` // LLM-formuleret statusvurdering
	Provider string    `json:"provider,omitempty"`
}

// snapshot er det (og kun det) modellen får at se.
type advisorSnapshot struct {
	TasksTotal      int    `json:"tasksTotal"`
	TasksActionable int    `json:"tasksActionable"`
	TasksWaiting    int    `json:"tasksWaiting"`
	TasksInProgress int    `json:"tasksInProgress"`
	TasksDone       int    `json:"tasksDone"`
	PhasesCompleted int    `json:"phasesCompleted"`
	PhasesTotal     int    `json:"phasesTotal"`
	PhasesWithDates int    `json:"phasesWithDates"`
	BudgetOre       int64  `json:"budgetOre"`
	SpentOre        int64  `json:"spentOre"`
	MaterialsNeeded int    `json:"materialsNeeded"`
	CaseStatus      string `json:"caseStatus,omitempty"`
	HasDrawing      bool   `json:"hasDrawing"`
	HasSources      bool   `json:"hasSources"`
	NextActionable  string `json:"nextActionable,omitempty"`
}

func (s *Advisor) Get(ctx context.Context, userID, projectID uuid.UUID) (*AdvisorResult, error) {
	if _, err := requireRead(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}

	snap, insights, err := s.analyse(ctx, projectID)
	if err != nil {
		return nil, err
	}
	result := &AdvisorResult{Insights: insights}

	// LLM-vurderingen er ren bonus — fejl her må aldrig vælte overblikket.
	if instructions, readErr := os.ReadFile(filepath.Join(s.docsDir, "advisor.md")); readErr == nil {
		snapJSON, _ := json.MarshalIndent(snap, "", "  ")
		summary, llmErr := s.provider.Complete(ctx, ai.Request{
			System:    string(instructions),
			Messages:  []ai.Message{{Role: "user", Content: "ØJEBLIKSBILLEDE:\n" + string(snapJSON)}},
			MaxTokens: 400,
		})
		if llmErr == nil {
			result.Summary = summary
			result.Provider = s.provider.Name()
		} else if !errors.Is(llmErr, ai.ErrNotConfigured) {
			result.Insights = append(result.Insights, Insight{
				Level: "info", Text: "Rådgiverens AI-vurdering er utilgængelig lige nu (" + llmErr.Error() + ").",
			})
		}
	}
	return result, nil
}

func (s *Advisor) analyse(ctx context.Context, projectID uuid.UUID) (*advisorSnapshot, []Insight, error) {
	snap := &advisorSnapshot{}
	insights := []Insight{}

	tasks, err := s.tasks.ListByProject(ctx, projectID)
	if err != nil {
		return nil, nil, err
	}
	edges, err := s.tasks.ListDependenciesByProject(ctx, projectID)
	if err != nil {
		return nil, nil, err
	}
	board := buildBoard(tasks, edges)
	snap.TasksTotal = len(board)
	for _, t := range board {
		switch {
		case t.Actionable:
			snap.TasksActionable++
			if snap.NextActionable == "" {
				snap.NextActionable = t.Title
			}
		case t.Status == domain.TaskInProgress:
			snap.TasksInProgress++
		case t.Status.Terminal():
			snap.TasksDone++
		default:
			snap.TasksWaiting++
		}
	}

	phases, err := s.phases.ListByProject(ctx, projectID)
	if err != nil {
		return nil, nil, err
	}
	snap.PhasesTotal = len(phases)
	for _, p := range phases {
		if p.Status == domain.PhaseCompleted {
			snap.PhasesCompleted++
		}
		if p.PlannedStart != nil || p.PlannedEnd != nil {
			snap.PhasesWithDates++
		}
	}

	summary, err := s.budget.Summary(ctx, projectID)
	if err != nil {
		return nil, nil, err
	}
	snap.BudgetOre = summary.EstimatedOre
	snap.SpentOre = summary.SpentOre

	materials, err := s.materials.ListByProject(ctx, projectID)
	if err != nil {
		return nil, nil, err
	}
	missingSupplier := 0
	for _, m := range materials {
		if m.Status == domain.MaterialNeeded {
			snap.MaterialsNeeded++
			if m.SupplierID == nil {
				missingSupplier++
			}
		}
	}

	cases, err := s.caseFiles.ListByProject(ctx, projectID)
	if err != nil {
		return nil, nil, err
	}
	if len(cases) > 0 {
		snap.CaseStatus = string(cases[0].Status)
	}

	drawings, err := s.drawings.ListByProject(ctx, projectID)
	if err != nil {
		return nil, nil, err
	}
	snap.HasDrawing = len(drawings) > 0

	sources, err := s.sources.ListForProject(ctx, projectID)
	if err != nil {
		return nil, nil, err
	}
	snap.HasSources = len(sources) > 0

	// --- regelbaserede indsigter (virker altid, også uden LLM) --------------
	if snap.BudgetOre > 0 && snap.SpentOre > snap.BudgetOre {
		insights = append(insights, Insight{Level: "warning", LinkTo: "budget",
			Text: fmt.Sprintf("Budgettet er overskredet med %s.", kroner(snap.SpentOre-snap.BudgetOre))})
	} else if snap.BudgetOre > 0 && snap.SpentOre*10 >= snap.BudgetOre*9 {
		insights = append(insights, Insight{Level: "warning", LinkTo: "budget",
			Text: "Over 90 % af budgettet er brugt — gennemgå de resterende poster."})
	}
	if snap.TasksActionable > 0 {
		insights = append(insights, Insight{Level: "tip", LinkTo: "tasks",
			Text: fmt.Sprintf("%d opgave(r) er klar til start — først: \"%s\".", snap.TasksActionable, snap.NextActionable)})
	}
	if snap.TasksTotal > 0 && snap.TasksActionable == 0 && snap.TasksInProgress == 0 && snap.TasksDone < snap.TasksTotal {
		insights = append(insights, Insight{Level: "warning", LinkTo: "tasks",
			Text: "Alle åbne opgaver venter på andre — tjek om en afhængighed kan løsnes eller en opgave afsluttes."})
	}
	if snap.PhasesTotal > 0 && snap.PhasesWithDates == 0 {
		insights = append(insights, Insight{Level: "tip", LinkTo: "phases",
			Text: "Ingen faser har datoer endnu — sæt planlagt start/slut, så tidslinjen bliver levende."})
	}
	if missingSupplier > 0 {
		insights = append(insights, Insight{Level: "tip", LinkTo: "materials",
			Text: fmt.Sprintf("%d materiale(r) på indkøbslisten mangler leverandør.", missingSupplier)})
	}
	if snap.CaseStatus == string(domain.CaseDraft) && !snap.HasDrawing {
		insights = append(insights, Insight{Level: "tip", LinkTo: "drawings",
			Text: "Byggesagen er en kladde uden tegning — tegn grundplanen, så bilagene kan genereres."})
	}
	if !snap.HasSources {
		insights = append(insights, Insight{Level: "info", LinkTo: "sources",
			Text: "Indlæs BR18/lokalplan under Kildemateriale, så assistenten kan svare med kilder."})
	}
	if len(insights) == 0 {
		insights = append(insights, Insight{Level: "info",
			Text: "Ingen påmindelser lige nu — projektet ser velholdt ud."})
	}
	return snap, insights, nil
}

func kroner(ore int64) string {
	return fmt.Sprintf("%d.%02d kr.", ore/100, ore%100)
}
