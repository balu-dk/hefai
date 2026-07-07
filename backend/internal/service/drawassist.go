package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/balu-dk/hefai/backend/internal/ai"
	"github.com/balu-dk/hefai/backend/internal/domain"
)

// DrawAssist genererer et tegningsudkast ud fra en beskrivelse. Med
// LLM-provider følger modellen ai-docs/drawing.md; uden bygges en simpel
// deterministisk plan af beskrivelsens mål og rumantal. Udkastet indlæses i
// tegnefladen og gemmes først når brugeren selv gemmer en version.
type DrawAssist struct {
	provider ai.Provider
	docsDir  string
	drawings DrawingRepo
	access   ProjectAccess
}

func NewDrawAssist(provider ai.Provider, docsDir string, drawings DrawingRepo, access ProjectAccess) *DrawAssist {
	return &DrawAssist{provider: provider, docsDir: docsDir, drawings: drawings, access: access}
}

type DrawPromptInput struct {
	Prompt string `json:"prompt"`
}

type DrawPromptResult struct {
	Data     domain.DrawingData `json:"data"`
	Source   string             `json:"source"` // "llm" eller "template"
	Provider string             `json:"provider"`
	Notice   string             `json:"notice,omitempty"`
}

func (s *DrawAssist) Generate(ctx context.Context, userID, drawingID uuid.UUID, in DrawPromptInput) (*DrawPromptResult, error) {
	drawing, err := s.drawings.Get(ctx, drawingID)
	if err != nil {
		return nil, err
	}
	if err := requireWrite(ctx, s.access, drawing.ProjectID, userID); err != nil {
		return nil, err
	}
	prompt := strings.TrimSpace(in.Prompt)
	if prompt == "" {
		return nil, domain.Validation("beskriv hvad der skal tegnes — fx \"hus på 8x10 m med 3 rum og saddeltag\"")
	}

	result := &DrawPromptResult{Provider: s.provider.Name()}

	if instructions, readErr := os.ReadFile(filepath.Join(s.docsDir, "drawing.md")); readErr == nil {
		raw, llmErr := s.provider.Complete(ctx, ai.Request{
			System:   string(instructions),
			Messages: []ai.Message{{Role: "user", Content: prompt}},
		})
		if llmErr == nil {
			if data, parseErr := parseDrawingData(raw); parseErr == nil {
				result.Data = *data
				result.Source = "llm"
				return result, nil
			} else {
				result.Notice = "Modellens tegning kunne ikke læses (" + parseErr.Error() + ") — der vises en simpel standardplan i stedet."
			}
		} else if errors.Is(llmErr, ai.ErrNotConfigured) {
			result.Notice = "Ingen LLM-provider er sat op, så tegningen er bygget af en simpel skabelon. " +
				"Med LLM aktiveret kan assistenten tegne langt mere detaljeret efter din beskrivelse."
		} else {
			result.Notice = "LLM-kaldet fejlede (" + llmErr.Error() + ") — der vises en simpel standardplan i stedet."
		}
	}

	data, err := templateDrawing(prompt)
	if err != nil {
		return nil, err
	}
	result.Data = *data
	result.Source = "template"
	return result, nil
}

func parseDrawingData(raw string) (*domain.DrawingData, error) {
	text := strings.TrimSpace(raw)
	if idx := strings.Index(text, "```"); idx >= 0 {
		text = text[idx+3:]
		text = strings.TrimPrefix(text, "json")
		if end := strings.Index(text, "```"); end >= 0 {
			text = text[:end]
		}
	}
	if idx := strings.Index(text, "{"); idx > 0 {
		text = text[idx:]
	}
	var data domain.DrawingData
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &data); err != nil {
		return nil, fmt.Errorf("ugyldig JSON: %w", err)
	}
	// AI'en tegner kun bygningen; grund/træer/geo er brugerens.
	data.Plot = nil
	data.Trees = nil
	data.Geo = nil
	if err := data.Validate(); err != nil {
		return nil, err
	}
	if len(data.Walls) == 0 {
		return nil, errors.New("tegningen indeholder ingen vægge")
	}
	return &data, nil
}

var (
	dimsRe  = regexp.MustCompile(`(\d+(?:[.,]\d+)?)\s*[x×]\s*(\d+(?:[.,]\d+)?)`)
	roomsRe = regexp.MustCompile(`(\d+)\s*(?:rum|værelse)`)
)

// templateDrawing bygger en simpel retvinklet plan: ydervægge, jævnt
// fordelte skillevægge, dør i syd, vindue pr. rum i nord.
func templateDrawing(prompt string) (*domain.DrawingData, error) {
	lower := strings.ToLower(prompt)

	widthM, depthM := 8.0, 6.0
	if m := dimsRe.FindStringSubmatch(lower); m != nil {
		a, _ := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", "."), 64)
		b, _ := strconv.ParseFloat(strings.ReplaceAll(m[2], ",", "."), 64)
		// Mål over 30 tolkes som mm ellers meter.
		if a > 30 {
			a /= 1000
		}
		if b > 30 {
			b /= 1000
		}
		if a >= 2 && b >= 2 && a <= 40 && b <= 40 {
			widthM, depthM = max(a, b), min(a, b)
		}
	}
	roomCount := 2
	if m := roomsRe.FindStringSubmatch(lower); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil && n >= 1 && n <= 8 {
			roomCount = n
		}
	}

	const outer = 350.0 // ydervægstykkelse mm
	const inner = 100.0
	w := widthM * 1000
	d := depthM * 1000

	data := &domain.DrawingData{
		WallHeightMM: 2500,
		Walls: []domain.Wall{
			{ID: "yder-nord", From: domain.Point{X: 0, Y: 0}, To: domain.Point{X: w, Y: 0}, ThicknessMM: outer, IsLoadBearing: true},
			{ID: "yder-oest", From: domain.Point{X: w, Y: 0}, To: domain.Point{X: w, Y: d}, ThicknessMM: outer, IsLoadBearing: true},
			{ID: "yder-syd", From: domain.Point{X: w, Y: d}, To: domain.Point{X: 0, Y: d}, ThicknessMM: outer, IsLoadBearing: true},
			{ID: "yder-vest", From: domain.Point{X: 0, Y: d}, To: domain.Point{X: 0, Y: 0}, ThicknessMM: outer, IsLoadBearing: true},
		},
	}
	if strings.Contains(lower, "saddeltag") || strings.Contains(lower, "sadeltag") {
		data.RoofAngleDeg = 30
	}

	// Skillevægge og rum: jævn opdeling langs den lange (x-) akse.
	sectionW := w / float64(roomCount)
	for i := 1; i < roomCount; i++ {
		x := float64(i) * sectionW
		data.Walls = append(data.Walls, domain.Wall{
			ID:          fmt.Sprintf("skille-%d", i),
			From:        domain.Point{X: x, Y: 0},
			To:          domain.Point{X: x, Y: d},
			ThicknessMM: inner,
		})
	}
	for i := 0; i < roomCount; i++ {
		x0 := float64(i)*sectionW + outer/2
		x1 := float64(i+1)*sectionW - outer/2
		name := fmt.Sprintf("Rum %d", i+1)
		if i == 0 {
			name = "Stue/køkken"
		}
		data.Rooms = append(data.Rooms, domain.RoomShape{
			Name: name,
			Polygon: []domain.Point{
				{X: x0, Y: outer / 2}, {X: x1, Y: outer / 2},
				{X: x1, Y: d - outer/2}, {X: x0, Y: d - outer/2},
			},
		})
		// Vindue pr. rum i nordvæggen.
		data.Openings = append(data.Openings, domain.Opening{
			WallID: "yder-nord", Type: domain.OpeningWindow,
			OffsetMM: x0 + (x1-x0)/2 - 600, WidthMM: 1200, HeightMM: 1200,
		})
	}
	// Hoveddør i sydvæggen. Sydvæggen løber fra (w,d) mod (0,d), så offset
	// måles fra øst.
	data.Openings = append(data.Openings, domain.Opening{
		WallID: "yder-syd", Type: domain.OpeningDoor,
		OffsetMM: w/2 - 500, WidthMM: 1000, HeightMM: 2100,
	})

	if err := data.Validate(); err != nil {
		return nil, err
	}
	return data, nil
}
