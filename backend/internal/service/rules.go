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
	"github.com/balu-dk/hefai/backend/internal/calc"
	"github.com/balu-dk/hefai/backend/internal/domain"
)

// Rules er lovligheds-motoren: strukturerede grænseværdier med kildekrav,
// som evalueres deterministisk mod tegningens målte fakta. AI'en foreslår
// regler ud fra kildematerialet (med ordret citat); brugeren bekræfter;
// evalueringen er ren Go-kode og kører live.

type RuleRepo interface {
	Upsert(ctx context.Context, rule *domain.ComplianceRule) (*domain.ComplianceRule, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.ComplianceRule, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.ComplianceRule, error)
	Update(ctx context.Context, rule *domain.ComplianceRule) (*domain.ComplianceRule, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type Rules struct {
	repo     RuleRepo
	provider ai.Provider
	docsDir  string
	projects ProjectRepo
	drawings DrawingRepo
	sources  SourceRepo
	access   ProjectAccess
}

func NewRules(repo RuleRepo, provider ai.Provider, docsDir string, projects ProjectRepo,
	drawings DrawingRepo, sources SourceRepo, access ProjectAccess) *Rules {
	return &Rules{repo: repo, provider: provider, docsDir: docsDir, projects: projects,
		drawings: drawings, sources: sources, access: access}
}

// RuleParameter beskriver en målbar parameter motoren kan tjekke.
type RuleParameter struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Unit      string `json:"unit"`
	Direction string `json:"direction"` // "max" eller "min"
}

var ruleCatalog = []RuleParameter{
	{ID: "max_bebyggelsesprocent", Label: "Maks. bebyggelsesprocent", Unit: "%", Direction: "max"},
	{ID: "min_skelafstand", Label: "Min. afstand til skel", Unit: "m", Direction: "min"},
	{ID: "max_bygningshoejde", Label: "Maks. bygningshøjde", Unit: "m", Direction: "max"},
	{ID: "max_taghaeldning", Label: "Maks. taghældning", Unit: "°", Direction: "max"},
	{ID: "max_bebygget_areal", Label: "Maks. bebygget areal", Unit: "m²", Direction: "max"},
}

func catalogEntry(parameter string) *RuleParameter {
	for i := range ruleCatalog {
		if ruleCatalog[i].ID == parameter {
			return &ruleCatalog[i]
		}
	}
	return nil
}

func (s *Rules) Catalog() []RuleParameter { return ruleCatalog }

// --- CRUD ---------------------------------------------------------------------

type RuleInput struct {
	Parameter     string     `json:"parameter"`
	Value         float64    `json:"value"`
	SourceChunkID *uuid.UUID `json:"sourceChunkId"`
	Quote         string     `json:"quote"`
	Status        string     `json:"status"`
	Note          string     `json:"note"`
}

func (s *Rules) Create(ctx context.Context, userID, projectID uuid.UUID, in RuleInput) (*domain.ComplianceRule, error) {
	if err := requireWrite(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	if catalogEntry(in.Parameter) == nil {
		return nil, domain.Validation("ukendt parameter — se kataloget")
	}
	if in.Value <= 0 {
		return nil, domain.Validation("grænseværdien skal være positiv")
	}
	status := domain.RuleStatus(in.Status)
	if in.Status == "" {
		status = domain.RuleConfirmed // manuelt indtastet = brugeren står inde for den
	}
	if !status.Valid() {
		return nil, domain.Validation("ugyldig status")
	}
	return s.repo.Upsert(ctx, &domain.ComplianceRule{
		ProjectID:     projectID,
		Parameter:     in.Parameter,
		Value:         in.Value,
		SourceChunkID: in.SourceChunkID,
		Quote:         in.Quote,
		Status:        status,
		Note:          in.Note,
	})
}

func (s *Rules) List(ctx context.Context, userID, projectID uuid.UUID) ([]*domain.ComplianceRule, error) {
	if _, err := requireRead(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListByProject(ctx, projectID)
}

type RulePatch struct {
	Value  *float64 `json:"value"`
	Status *string  `json:"status"`
	Note   *string  `json:"note"`
}

func (s *Rules) Update(ctx context.Context, userID, ruleID uuid.UUID, patch RulePatch) (*domain.ComplianceRule, error) {
	rule, err := s.repo.Get(ctx, ruleID)
	if err != nil {
		return nil, err
	}
	if err := requireWrite(ctx, s.access, rule.ProjectID, userID); err != nil {
		return nil, err
	}
	if patch.Value != nil {
		if *patch.Value <= 0 {
			return nil, domain.Validation("grænseværdien skal være positiv")
		}
		rule.Value = *patch.Value
	}
	if patch.Status != nil {
		status := domain.RuleStatus(*patch.Status)
		if !status.Valid() {
			return nil, domain.Validation("ugyldig status")
		}
		rule.Status = status
	}
	if patch.Note != nil {
		rule.Note = *patch.Note
	}
	return s.repo.Update(ctx, rule)
}

func (s *Rules) Delete(ctx context.Context, userID, ruleID uuid.UUID) error {
	rule, err := s.repo.Get(ctx, ruleID)
	if err != nil {
		return err
	}
	if err := requireWrite(ctx, s.access, rule.ProjectID, userID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, ruleID)
}

// --- evaluering ------------------------------------------------------------------

type RuleEvaluation struct {
	Rule      *domain.ComplianceRule `json:"rule"`
	Parameter RuleParameter          `json:"parameter"`
	FactValue *float64               `json:"factValue"`
	Status    string                 `json:"status"` // "ok", "violation", "unknown"
	Margin    *float64               `json:"margin"` // positiv = luft til grænsen
}

type EvaluationResult struct {
	Facts        *calc.ProjectFacts `json:"facts"`
	DrawingTitle string             `json:"drawingTitle,omitempty"`
	Evaluations  []RuleEvaluation   `json:"evaluations"`
	Violations   int                `json:"violations"`
	Notice       string             `json:"notice"`
}

// Evaluate måler den nyeste tegning op og sammenligner med alle ikke-
// afviste regler. Ren deterministisk kode — LLM'en er ikke involveret.
func (s *Rules) Evaluate(ctx context.Context, userID, projectID uuid.UUID) (*EvaluationResult, error) {
	if _, err := requireRead(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	project, err := s.projects.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	rules, err := s.repo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	result := &EvaluationResult{
		Evaluations: []RuleEvaluation{},
		Notice: "Automatisk egenkontrol mod dine bekræftede kilder — vejledende hjælp, " +
			"ikke en myndighedsafgørelse. Kommunen kan stille yderligere krav.",
	}

	// Nyeste tegning med gemt version lægges til grund.
	var data *domain.DrawingData
	drawings, err := s.drawings.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	var newest *domain.DrawingVersion
	for _, d := range drawings {
		v, verErr := s.drawings.LatestVersion(ctx, d.ID)
		if verErr != nil {
			continue
		}
		if newest == nil || v.CreatedAt.After(newest.CreatedAt) {
			newest = v
			result.DrawingTitle = d.Title
		}
	}
	if newest != nil {
		data = &newest.Data
	} else {
		data = &domain.DrawingData{}
	}

	plotArea := 0.0
	if project.PlotAreaM2 != nil {
		plotArea = *project.PlotAreaM2
	}
	result.Facts = calc.ComputeFacts(data, plotArea)

	for _, rule := range rules {
		if rule.Status == domain.RuleRejected {
			continue
		}
		param := catalogEntry(rule.Parameter)
		if param == nil {
			continue
		}
		eval := RuleEvaluation{Rule: rule, Parameter: *param, Status: "unknown"}
		if fact := factFor(rule.Parameter, result.Facts); fact != nil {
			eval.FactValue = fact
			var margin float64
			if param.Direction == "max" {
				margin = rule.Value - *fact
			} else {
				margin = *fact - rule.Value
			}
			eval.Margin = &margin
			if margin >= 0 {
				eval.Status = "ok"
			} else {
				eval.Status = "violation"
				result.Violations++
			}
		}
		result.Evaluations = append(result.Evaluations, eval)
	}
	return result, nil
}

func factFor(parameter string, facts *calc.ProjectFacts) *float64 {
	switch parameter {
	case "max_bebyggelsesprocent":
		return facts.BebyggelsesprocentPct
	case "min_skelafstand":
		return facts.MinSkelafstandM
	case "max_bygningshoejde":
		return facts.BygningshoejdeM
	case "max_taghaeldning":
		return facts.TaghaeldningDeg
	case "max_bebygget_areal":
		return facts.FootprintM2
	}
	return nil
}

// --- AI-udtræk af regler fra kildematerialet ------------------------------------

type ExtractResult struct {
	Suggested []*domain.ComplianceRule `json:"suggested"`
	Notice    string                   `json:"notice,omitempty"`
	Provider  string                   `json:"provider"`
}

type extractedRule struct {
	Parameter  string  `json:"parameter"`
	Value      float64 `json:"value"`
	ChunkIndex int     `json:"chunkIndex"`
	Quote      string  `json:"quote"`
	Note       string  `json:"note"`
}

// Extract lader modellen foreslå regler ud fra kildematerialet. Hvert
// forslag skal citere sit uddrag; alt gemmes som "suggested" til manuel
// bekræftelse.
func (s *Rules) Extract(ctx context.Context, userID, projectID uuid.UUID, _ struct{}) (*ExtractResult, error) {
	if err := requireWrite(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}

	// Relevante uddrag findes med søgninger pr. emne.
	queries := []string{
		"bebyggelsesprocent", "afstand skel", "højde bygning etage", "taghældning tag vinkel", "bebygget areal",
	}
	seen := map[uuid.UUID]bool{}
	var hits []*domain.SourceHit
	for _, q := range queries {
		results, err := s.sources.Search(ctx, projectID, q, 4)
		if err != nil {
			return nil, err
		}
		for _, h := range results {
			if !seen[h.ChunkID] {
				seen[h.ChunkID] = true
				hits = append(hits, h)
			}
		}
	}
	if len(hits) == 0 {
		return nil, domain.Validation("intet kildemateriale at udtrække regler fra — indlæs BR18/lokalplan under Kildemateriale først")
	}

	instructions, err := os.ReadFile(filepath.Join(s.docsDir, "rules.md"))
	if err != nil {
		return nil, fmt.Errorf("instruktionsfil mangler: %w", err)
	}

	var b strings.Builder
	b.WriteString("KILDER:\n")
	for i, h := range hits {
		fmt.Fprintf(&b, "[%d] %s (%s):\n%s\n\n", i, h.SourceTitle, h.SectionRef, h.Content)
	}

	raw, err := s.provider.Complete(ctx, ai.Request{
		System:   string(instructions),
		Messages: []ai.Message{{Role: "user", Content: b.String()}},
	})
	if err != nil {
		if errors.Is(err, ai.ErrNotConfigured) {
			return &ExtractResult{
				Suggested: []*domain.ComplianceRule{},
				Provider:  s.provider.Name(),
				Notice: "Ingen LLM-provider er sat op. Tilføj reglerne manuelt — søg i kildematerialet, " +
					"find grænseværdien og opret reglen med henvisning. Med LLM aktiveret udtrækkes forslagene automatisk.",
			}, nil
		}
		return nil, err
	}

	extracted, err := parseExtractedRules(raw)
	if err != nil {
		return nil, domain.Validation("modellens svar kunne ikke læses: " + err.Error())
	}

	result := &ExtractResult{Suggested: []*domain.ComplianceRule{}, Provider: s.provider.Name()}
	for _, e := range extracted {
		if catalogEntry(e.Parameter) == nil || e.Value <= 0 {
			continue
		}
		if e.ChunkIndex < 0 || e.ChunkIndex >= len(hits) {
			continue // forslag uden gyldig kilde afvises
		}
		chunkID := hits[e.ChunkIndex].ChunkID
		rule, upsertErr := s.repo.Upsert(ctx, &domain.ComplianceRule{
			ProjectID:     projectID,
			Parameter:     e.Parameter,
			Value:         e.Value,
			SourceChunkID: &chunkID,
			Quote:         e.Quote,
			Status:        domain.RuleSuggested,
			Note:          e.Note,
		})
		if upsertErr != nil {
			return nil, upsertErr
		}
		result.Suggested = append(result.Suggested, rule)
	}
	if len(result.Suggested) == 0 {
		result.Notice = "Modellen fandt ingen målbare grænseværdier i kildematerialet."
	}
	return result, nil
}

func parseExtractedRules(raw string) ([]extractedRule, error) {
	text := strings.TrimSpace(raw)
	if idx := strings.Index(text, "```"); idx >= 0 {
		text = text[idx+3:]
		text = strings.TrimPrefix(text, "json")
		if end := strings.Index(text, "```"); end >= 0 {
			text = text[:end]
		}
	}
	if idx := strings.Index(text, "["); idx > 0 {
		text = text[idx:]
	}
	var rules []extractedRule
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &rules); err != nil {
		return nil, err
	}
	return rules, nil
}
