package pdfgen

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

const engineerBanner = "KLADDE — vejledende grundlag, kræver statiker-godkendelse"

// StructuralPackage renders the full engineer hand-over: construction
// description, loads with derivations, advisory calculations with their
// assumptions, and a response section for the engineer.
func StructuralPackage(m Meta, title string, elements []*domain.StructuralElement,
	loads []*domain.Load, estimates []*domain.CalculationEstimate) ([]byte, error) {

	d := newDoc("P", title, m.GeneratedAt)
	d.pdf.AddPage()
	d.h1(title)
	d.pdf.SetFont("Helvetica", "B", 10)
	d.pdf.SetTextColor(200, 30, 30)
	d.pdf.MultiCell(0, 5, d.tr(engineerBanner), "", "L", false)
	d.pdf.SetTextColor(0, 0, 0)
	d.pdf.Ln(3)
	d.metaBlock(m)

	d.para("Dette materiale er struktureret af Hefai som grundlag for statikerens gennemgang. " +
		"Alle beregninger er vejledende overslag med eksplicitte antagelser og standardreferencer. " +
		"Ingen del af materialet må anvendes til udførelse før statikeren har kontrolleret, " +
		"rettet og godkendt det.")

	// --- Konstruktionsbeskrivelse ------------------------------------------
	d.h2("1. Konstruktionselementer")
	if len(elements) == 0 {
		d.para("(Ingen elementer registreret.)")
	}
	for i, e := range elements {
		d.pdf.SetFont("Helvetica", "B", 10)
		d.pdf.MultiCell(0, 5, d.tr(fmt.Sprintf("1.%d %s", i+1, e.Name)), "", "L", false)
		d.kv("Type", elementTypeLabel(e.ElementType)+boolSuffix(e.IsLoadBearing, " (bærende)", " (ikke-bærende)"))
		d.kv("Materiale", fmt.Sprintf("%s %s", materialLabel(e.Material), e.MaterialSpec))
		if geo := prettyJSON(e.Geometry); geo != "" && geo != "{}" {
			d.kv("Geometri", geo)
		}
		if e.Notes != "" {
			d.kv("Noter", e.Notes)
		}
		d.pdf.Ln(2)
	}

	// --- Laster --------------------------------------------------------------
	d.h2("2. Laster og lastantagelser")
	if len(loads) == 0 {
		d.para("(Ingen laster registreret.)")
	}
	for _, l := range loads {
		scope := "Projekt-niveau"
		if l.StructuralElementID != nil {
			scope = "Element: " + elementName(elements, *l.StructuralElementID)
		}
		d.pdf.SetFont("Helvetica", "B", 10)
		d.pdf.MultiCell(0, 5, d.tr(fmt.Sprintf("%s — %.3g %s (%s)", loadTypeLabel(l.LoadType), l.Value, l.Unit, loadStatusLabel(l.Status))), "", "L", false)
		d.kv("Omfang", scope)
		if l.StandardReference != "" {
			d.kv("Reference", l.StandardReference)
		}
		if deriv := prettyJSON(l.Derivation); deriv != "" && deriv != "{}" {
			d.kv("Udledning", deriv)
		}
		if l.Notes != "" {
			d.kv("Noter", l.Notes)
		}
		d.pdf.Ln(2)
	}

	// --- Vejledende beregninger ----------------------------------------------
	d.h2("3. Vejledende beregninger (til verifikation)")
	if len(estimates) == 0 {
		d.para("(Ingen beregninger udført.)")
	}
	for i, e := range estimates {
		d.pdf.SetFont("Helvetica", "B", 10)
		target := ""
		if e.StructuralElementID != nil {
			target = " — " + elementName(elements, *e.StructuralElementID)
		}
		d.pdf.MultiCell(0, 5, d.tr(fmt.Sprintf("3.%d %s%s [%s]", i+1, e.Method, target, e.Status)), "", "L", false)
		d.kv("Standard", e.StandardReference)
		d.kv("Inputs", prettyJSON(e.Inputs))

		var assumptions []struct {
			Text      string `json:"text"`
			Reference string `json:"reference"`
		}
		if err := json.Unmarshal(e.Assumptions, &assumptions); err == nil && len(assumptions) > 0 {
			d.pdf.SetFont("Helvetica", "B", 9)
			d.pdf.MultiCell(0, 5, d.tr("Antagelser (skal be- eller afkræftes):"), "", "L", false)
			d.pdf.SetFont("Helvetica", "", 9)
			for _, a := range assumptions {
				d.pdf.MultiCell(0, 4.5, d.tr("• "+a.Text+" ["+a.Reference+"]"), "", "L", false)
			}
		}
		d.kv("Resultater", prettyJSON(e.Results))
		d.pdf.Ln(2)
	}

	// --- Statikerens svar ------------------------------------------------------
	d.pdf.AddPage()
	d.h2("4. Statikerens gennemgang")
	d.para("Udfyldes af statikeren. Hvert punkt ovenfor bedes markeret som godkendt, ændret " +
		"(med rettede værdier) eller afvist. Svaret registreres i Hefai, så bekræftede og " +
		"ændrede antagelser kan spores.")
	for _, label := range []string{"Navn", "Firma", "Autorisation/anerkendelse", "Dato"} {
		d.pdf.SetFont("Helvetica", "", 10)
		d.pdf.CellFormat(55, 10, d.tr(label+":"), "", 0, "L", false, 0, "")
		d.pdf.CellFormat(0, 10, "", "B", 1, "L", false, 0, "")
	}
	d.pdf.Ln(4)
	d.pdf.SetFont("Helvetica", "B", 10)
	d.pdf.MultiCell(0, 6, d.tr("Kommentarer og rettelser:"), "", "L", false)
	for range 10 {
		d.pdf.CellFormat(0, 8, "", "B", 1, "L", false, 0, "")
	}
	return d.output()
}

func elementName(elements []*domain.StructuralElement, id interface{ String() string }) string {
	for _, e := range elements {
		if e.ID.String() == id.String() {
			return e.Name
		}
	}
	return "ukendt element"
}

func prettyJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	return flatten(v)
}

// flatten renders JSON as "key: value; key: value" with deterministic key
// order for compact, reproducible PDF display.
func flatten(v any) string {
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := ""
		for _, k := range keys {
			if out != "" {
				out += "; "
			}
			out += k + ": " + flatten(t[k])
		}
		return out
	case []any:
		out := ""
		for _, item := range t {
			if out != "" {
				out += ", "
			}
			out += flatten(item)
		}
		return "[" + out + "]"
	case float64:
		return fmt.Sprintf("%.4g", t)
	case bool:
		return fmt.Sprintf("%t", t)
	case nil:
		return "-"
	default:
		return fmt.Sprintf("%v", t)
	}
}

func boolSuffix(v bool, yes, no string) string {
	if v {
		return yes
	}
	return no
}

func elementTypeLabel(t domain.StructuralElementType) string {
	switch t {
	case domain.ElementBeam:
		return "Bjælke"
	case domain.ElementColumn:
		return "Søjle"
	case domain.ElementWall:
		return "Væg"
	case domain.ElementFoundation:
		return "Fundament"
	case domain.ElementRoof:
		return "Tagkonstruktion"
	case domain.ElementSlab:
		return "Dæk"
	default:
		return "Andet"
	}
}

func materialLabel(m domain.StructuralMaterial) string {
	switch m {
	case domain.MaterialTimber:
		return "Træ"
	case domain.MaterialSteel:
		return "Stål"
	case domain.MaterialConcrete:
		return "Beton"
	case domain.MaterialMasonry:
		return "Murværk"
	default:
		return "Andet"
	}
}

func loadTypeLabel(t domain.LoadType) string {
	switch t {
	case domain.LoadDead:
		return "Egenlast"
	case domain.LoadLive:
		return "Nyttelast"
	case domain.LoadSnow:
		return "Snelast"
	case domain.LoadWind:
		return "Vindlast"
	case domain.LoadPoint:
		return "Punktlast"
	case domain.LoadLine:
		return "Linjelast"
	default:
		return "Anden last"
	}
}

func loadStatusLabel(s domain.LoadStatus) string {
	switch s {
	case domain.LoadEngineerConfirmed:
		return "bekræftet af statiker"
	case domain.LoadEngineerChanged:
		return "ændret af statiker"
	default:
		return "antaget — kræver bekræftelse"
	}
}
