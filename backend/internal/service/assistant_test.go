package service

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/balu-dk/hefai/backend/internal/ai"
	"github.com/balu-dk/hefai/backend/internal/domain"
	"github.com/balu-dk/hefai/backend/internal/rag"
)

type fakeAccess struct{}

func (fakeAccess) GetMemberRole(context.Context, uuid.UUID, uuid.UUID) (domain.ProjectRole, error) {
	return domain.RoleOwner, nil
}

type fakeSources struct {
	hits []*domain.SourceHit
}

func (f *fakeSources) Search(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*domain.SourceHit, error) {
	return f.hits, nil
}
func (f *fakeSources) CreateWithChunks(context.Context, *domain.SourceDocument, []rag.Chunk) (*domain.SourceDocument, error) {
	return nil, nil
}
func (f *fakeSources) Get(context.Context, uuid.UUID) (*domain.SourceDocument, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeSources) ListForProject(context.Context, uuid.UUID) ([]*domain.SourceDocument, error) {
	return nil, nil
}
func (f *fakeSources) Delete(context.Context, uuid.UUID) error { return nil }
func (f *fakeSources) ChunkProject(context.Context, uuid.UUID) (*uuid.UUID, string, error) {
	return nil, "", nil
}

type promptCapturingProvider struct {
	system string
	prompt string
}

func (p *promptCapturingProvider) Complete(_ context.Context, req ai.Request) (string, error) {
	p.system = req.System
	p.prompt = req.Messages[0].Content
	return "Ifølge [1] skal afstanden være mindst 2,5 m. KRÆVER BEKRÆFTELSE fra kommunen.", nil
}
func (p *promptCapturingProvider) Name() string { return "fake" }

func hit(section, content string) *domain.SourceHit {
	return &domain.SourceHit{
		ChunkID: uuid.New(), SourceID: uuid.New(),
		SourceTitle: "BR18", SourceKind: domain.SourceBR18,
		SectionRef: section, Content: content, Rank: 1,
	}
}

func TestAssistantGroundsPromptInSources(t *testing.T) {
	provider := &promptCapturingProvider{}
	assistant := NewAssistant(
		&fakeSources{hits: []*domain.SourceHit{hit("§ 180", "Afstand til skel mindst 2,5 m for sommerhuse.")}},
		nil, provider, fakeAccess{})

	result, err := assistant.Ask(context.Background(), uuid.New(), uuid.New(),
		AskInput{Question: "Hvor tæt på skel må jeg bygge?"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Answered {
		t.Error("expected an answered result with a working provider")
	}
	if len(result.Citations) != 1 {
		t.Errorf("expected 1 citation, got %d", len(result.Citations))
	}
	if !strings.Contains(provider.prompt, "KILDER:") || !strings.Contains(provider.prompt, "§ 180") {
		t.Error("prompt must embed the numbered sources")
	}
	if !strings.Contains(provider.system, "ALDRIG opfinde") {
		t.Error("system prompt must carry the grounding rules")
	}
}

func TestAssistantWithoutProviderReturnsSources(t *testing.T) {
	assistant := NewAssistant(
		&fakeSources{hits: []*domain.SourceHit{hit("§ 180", "Afstand til skel …")}},
		nil, ai.Unconfigured{}, fakeAccess{})

	result, err := assistant.Ask(context.Background(), uuid.New(), uuid.New(),
		AskInput{Question: "skelafstand?"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Answered {
		t.Error("unconfigured provider cannot produce an answer")
	}
	if len(result.Citations) != 1 {
		t.Error("citations must still be returned without a provider")
	}
	if result.Notice == "" {
		t.Error("expected a notice explaining the missing provider")
	}
}

func TestAssistantNoSourcesRequiresConfirmation(t *testing.T) {
	assistant := NewAssistant(&fakeSources{}, nil, ai.Unconfigured{}, fakeAccess{})
	result, err := assistant.Ask(context.Background(), uuid.New(), uuid.New(),
		AskInput{Question: "må jeg bygge i to etager?"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Answered || len(result.Citations) != 0 {
		t.Error("no sources: nothing may be answered")
	}
	if !strings.Contains(result.Notice, "KRÆVER BEKRÆFTELSE") {
		t.Error("notice must flag that unsourced answers are not given")
	}
}
