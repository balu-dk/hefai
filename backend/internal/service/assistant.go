package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/balu-dk/hefai/backend/internal/ai"
	"github.com/balu-dk/hefai/backend/internal/domain"
)

// assistantSystemPrompt encodes the non-negotiable grounding rules. The
// assistant structures and explains — it never invents regulation text and
// never promises approval.
const assistantSystemPrompt = `Du er byggesagsassistenten i Hefai, et værktøj til private byggeprojekter i Danmark.

Ufravigelige regler:
1. Du må KUN henvise til paragraffer, krav og tal der står ordret i KILDER-blokken nedenfor. Du må ALDRIG opfinde eller gætte paragrafnumre, grænseværdier eller krav.
2. Findes svaret ikke i kilderne, siger du det direkte og anbefaler at spørge kommunen eller en rådgiver.
3. Du garanterer aldrig godkendelse og træffer aldrig myndighedsafgørelser.
4. Alt der kræver bekræftelse fra kommune, statiker eller rådgiver markeres tydeligt med "KRÆVER BEKRÆFTELSE".
5. Du regner ikke selv på konstruktioner. Henvis til Hefais beregningsmodul og til statiker.
6. Citér kilder med [nummer] der matcher KILDER-blokken.

Svar på dansk, kort og konkret.`

type Assistant struct {
	sources   SourceRepo
	caseFiles CaseFileRepo
	provider  ai.Provider
	access    ProjectAccess
}

func NewAssistant(sources SourceRepo, caseFiles CaseFileRepo, provider ai.Provider, access ProjectAccess) *Assistant {
	return &Assistant{sources: sources, caseFiles: caseFiles, provider: provider, access: access}
}

type AskInput struct {
	Question   string     `json:"question"`
	CaseFileID *uuid.UUID `json:"caseFileId"`
}

type AskResult struct {
	Answer    string              `json:"answer"`
	Answered  bool                `json:"answered"` // false when no LLM provider is configured
	Provider  string              `json:"provider"`
	Citations []*domain.SourceHit `json:"citations"`
	Notice    string              `json:"notice,omitempty"`
}

// Ask retrieves relevant source chunks and asks the LLM a grounded
// question. Without a configured provider the retrieval results are still
// returned so the user can read the sources directly.
func (s *Assistant) Ask(ctx context.Context, userID, projectID uuid.UUID, in AskInput) (*AskResult, error) {
	if _, err := requireRead(ctx, s.access, projectID, userID); err != nil {
		return nil, err
	}
	question := strings.TrimSpace(in.Question)
	if question == "" {
		return nil, domain.Validation("spørgsmål kræves")
	}

	hits, err := s.sources.Search(ctx, projectID, question, 8)
	if err != nil {
		return nil, err
	}

	result := &AskResult{Citations: hits, Provider: s.provider.Name()}

	if len(hits) == 0 {
		result.Notice = "Ingen af de indlæste kilder matcher spørgsmålet. Tilføj relevant materiale " +
			"(BR18-kapitler, lokalplan, kommunens vejledning) under Kilder, eller kontakt kommunen. " +
			"KRÆVER BEKRÆFTELSE: svar uden kildegrundlag gives ikke."
		return result, nil
	}

	prompt := buildAssistantPrompt(ctx, s.caseFiles, question, in.CaseFileID, hits)
	answer, err := s.provider.Complete(ctx, ai.Request{
		System:   assistantSystemPrompt,
		Messages: []ai.Message{{Role: "user", Content: prompt}},
	})
	if err != nil {
		if errors.Is(err, ai.ErrNotConfigured) {
			result.Notice = "Ingen LLM-provider er konfigureret endnu. Nedenfor er de mest relevante " +
				"passager fra dit kildemateriale — læs dem direkte. Assistentens formulerede svar " +
				"aktiveres når en provider er valgt."
			return result, nil
		}
		return nil, err
	}
	result.Answer = answer
	result.Answered = true
	return result, nil
}

// buildAssistantPrompt assembles the grounded user message: case context,
// numbered sources, then the question.
func buildAssistantPrompt(ctx context.Context, caseFiles CaseFileRepo, question string, caseFileID *uuid.UUID, hits []*domain.SourceHit) string {
	var b strings.Builder

	if caseFileID != nil {
		if c, err := caseFiles.Get(ctx, *caseFileID); err == nil {
			b.WriteString("BYGGESAG:\n")
			fmt.Fprintf(&b, "Titel: %s\nSagstype: %s\nStatus: %s\n", c.Title, c.CaseType, c.Status)
			if c.Description != "" {
				fmt.Fprintf(&b, "Beskrivelse: %s\n", c.Description)
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("KILDER:\n")
	for i, h := range hits {
		ref := h.SectionRef
		if ref == "" {
			ref = "uden afsnitsreference"
		}
		fmt.Fprintf(&b, "[%d] %s (%s) — %s:\n%s\n\n", i+1, h.SourceTitle, h.SourceKind, ref, h.Content)
	}

	fmt.Fprintf(&b, "SPØRGSMÅL:\n%s\n", question)
	return b.String()
}
