package service

import (
	"fmt"
	"strings"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

// templateBlueprint bygger en fornuftig standardplan direkte af
// interviewsvarene — deterministisk og uden LLM. Den dækker alle
// projekttyper og bruges som fallback (og som sikkerhedsnet hvis modellens
// svar ikke kan læses).
func templateBlueprint(project *domain.Project, in Interview) Blueprint {
	b := Blueprint{
		ProjectDescription: strings.TrimSpace(in.Goal),
		Notes: "Standardplan bygget af Hefais skabelon ud fra dine svar. Justér frit — " +
			"og husk: alt om tilladelser og bærende konstruktioner KRÆVER BEKRÆFTELSE " +
			"fra kommune eller rådgiver.",
	}

	for _, name := range in.Rooms {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		kind := "room"
		lower := strings.ToLower(name)
		if strings.Contains(lower, "terrasse") || strings.Contains(lower, "have") ||
			strings.Contains(lower, "carport") || strings.Contains(lower, "indkørsel") {
			kind = "outdoor"
		}
		b.Rooms = append(b.Rooms, BlueprintRoom{Name: name, Kind: kind})
	}

	isNewBuild := project.Kind == domain.ProjectKindNewBuild || project.Kind == domain.ProjectKindExtension
	b.NeedsBuildingCase = isNewBuild
	if isNewBuild {
		b.CaseDescription = strings.TrimSpace(in.Goal)
	}

	// task hjælper med at holde afhængigheder læselige: hver opgave afhænger
	// af den forrige medmindre andet angives.
	add := func(title, phase string, deps ...int) int {
		b.Tasks = append(b.Tasks, BlueprintTask{Title: title, Phase: phase, DependsOn: deps})
		return len(b.Tasks) - 1
	}

	if isNewBuild {
		plan := add("Afklar lokalplan og byggeret med kommunen", "")
		drawing := add("Tegn grundplan og placering på grunden i Hefai", "")
		permit := add("Ansøg om byggetilladelse", "", plan, drawing)
		quotes := add("Indhent tilbud fra håndværkere", "", drawing)
		found := add("Udgravning og fundament", "Grund & fundament", permit, quotes)
		shell := add("Råhus: vægge og etagedæk", "Råhus", found)
		roof := add("Tagkonstruktion og tagdækning", "Tag", shell)
		closed := add("Vinduer og døre monteres (lukket hus)", "Lukning", roof)
		el := add("El-installation (autoriseret elektriker)", "Installationer", closed)
		vvs := add("VVS og kloak (autoriseret installatør)", "Installationer", closed)
		inside := add("Indvendige vægge, gulve og lofter", "Indvendig", el, vvs)
		add("Maler og finish", "Finish", inside)
		add("Færdigmelding til kommunen", "Finish", inside)
	} else {
		scope := add("Beslut endeligt omfang og lav rækkefølgeplan", "")
		quotes := add("Indhent tilbud på de fag der kræver autorisation (el/VVS)", "", scope)
		demo := add("Nedrivning/klargøring af de berørte rum", "Indvendig", scope)
		craft := add("Håndværksarbejde udføres", "Indvendig", demo, quotes)
		add("Maler og finish", "Finish", craft)
		add("Ryd op og aflever billeddokumentation i Hefai", "Finish", craft)
		if containsAny(in.Goal, "bærende", "væg fjernes", "facade", "tilbygning", "udvid") {
			b.NeedsBuildingCase = true
			b.CaseDescription = strings.TrimSpace(in.Goal)
			b.Tasks = append(b.Tasks, BlueprintTask{
				Title: "Afklar med kommunen om ændringen kræver tilladelse (KRÆVER BEKRÆFTELSE)",
			})
		}
	}

	for _, feature := range in.Features {
		feature = strings.TrimSpace(feature)
		if feature != "" {
			add(fmt.Sprintf("Planlæg og udfør: %s", feature), "Finish")
		}
	}

	// Budgetfordeling: brugerens eget tal fordeles med runde andele — aldrig
	// opfundne priser. Uden budget oprettes poster med 0 kr. til udfyldning.
	type split struct {
		desc     string
		category string
		phase    string
		share    int // procent
	}
	var splits []split
	if isNewBuild {
		splits = []split{
			{"Fundament og terrændæk", "Håndværker", "Grund & fundament", 15},
			{"Råhus og tag", "Håndværker", "Råhus", 30},
			{"Vinduer og døre", "Materialer", "Lukning", 10},
			{"El og VVS", "Håndværker", "Installationer", 15},
			{"Indvendige arbejder", "Materialer", "Indvendig", 20},
			{"Gebyrer og tilladelser", "Gebyrer", "", 3},
			{"Uforudset (buffer)", "Andet", "", 7},
		}
	} else {
		splits = []split{
			{"Materialer", "Materialer", "Indvendig", 40},
			{"Håndværkere", "Håndværker", "Indvendig", 45},
			{"Uforudset (buffer)", "Andet", "", 15},
		}
	}
	for _, sp := range splits {
		b.BudgetItems = append(b.BudgetItems, BlueprintBudgetItem{
			Description:        sp.desc,
			Category:           sp.category,
			Phase:              sp.phase,
			EstimatedAmountOre: in.BudgetOre * int64(sp.share) / 100,
		})
	}

	return b
}

func containsAny(text string, needles ...string) bool {
	lower := strings.ToLower(text)
	for _, n := range needles {
		if strings.Contains(lower, n) {
			return true
		}
	}
	return false
}
