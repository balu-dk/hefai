package service

import (
	"strings"
	"testing"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

func TestParseBlueprintHandlesFencesAndPreamble(t *testing.T) {
	raw := "Her er planen:\n```json\n{\"tasks\":[{\"title\":\"Ring til kommunen\",\"dependsOn\":[]}]}\n```"
	b, err := parseBlueprint(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Tasks) != 1 || b.Tasks[0].Title != "Ring til kommunen" {
		t.Errorf("unexpected parse result: %+v", b)
	}
}

func TestParseBlueprintRejectsGarbageAndCycles(t *testing.T) {
	if _, err := parseBlueprint("bare tekst uden json"); err == nil {
		t.Error("garbage accepted")
	}
	if _, err := parseBlueprint(`{"tasks":[]}`); err == nil {
		t.Error("empty task list accepted")
	}
	cycle := `{"tasks":[{"title":"A","dependsOn":[1]},{"title":"B","dependsOn":[0]}]}`
	if _, err := parseBlueprint(cycle); err == nil || !strings.Contains(err.Error(), "cirkel") {
		t.Errorf("cycle not rejected: %v", err)
	}
	outOfRange := `{"tasks":[{"title":"A","dependsOn":[5]}]}`
	if _, err := parseBlueprint(outOfRange); err == nil {
		t.Error("out-of-range dependency accepted")
	}
}

func TestParseBlueprintNormalisesUnknownValues(t *testing.T) {
	raw := `{"tasks":[{"title":"A","phase":"Ukendt fase"}],
		"rooms":[{"name":"Stue","kind":"palads"}]}`
	b, err := parseBlueprint(raw)
	if err != nil {
		t.Fatal(err)
	}
	if b.Tasks[0].Phase != "" {
		t.Error("unknown phase should be cleared, not fail")
	}
	if b.Rooms[0].Kind != "room" {
		t.Error("unknown room kind should fall back to room")
	}
}

func TestTemplateBlueprintNewBuild(t *testing.T) {
	project := &domain.Project{Kind: domain.ProjectKindNewBuild, Name: "Test"}
	b := templateBlueprint(project, Interview{
		Goal:      "Nyt sommerhus på 70 m2",
		Rooms:     []string{"Stue/køkken", "Terrasse"},
		BudgetOre: 100_000_00,
	})
	if err := validateBlueprint(&b); err != nil {
		t.Fatalf("template must always validate: %v", err)
	}
	if !b.NeedsBuildingCase {
		t.Error("new build must open a building case")
	}
	if b.Rooms[1].Kind != "outdoor" {
		t.Error("terrasse should be outdoor")
	}
	var total int64
	for _, item := range b.BudgetItems {
		total += item.EstimatedAmountOre
	}
	if total > 100_000_00 {
		t.Errorf("budget split %d exceeds the user's budget", total)
	}
	if total < 95_000_00 {
		t.Errorf("budget split %d leaves too much unallocated", total)
	}
}

func TestTemplateBlueprintRenovationDetectsLoadBearing(t *testing.T) {
	project := &domain.Project{Kind: domain.ProjectKindRenovation, Name: "Test"}
	plain := templateBlueprint(project, Interview{Goal: "Nyt køkken og malede vægge"})
	if plain.NeedsBuildingCase {
		t.Error("plain interior renovation should not open a building case")
	}
	structural := templateBlueprint(project, Interview{Goal: "Vi vil fjerne en bærende væg mellem køkken og stue"})
	if !structural.NeedsBuildingCase {
		t.Error("removing a load-bearing wall must flag a building case")
	}
	if err := validateBlueprint(&structural); err != nil {
		t.Fatal(err)
	}
}
