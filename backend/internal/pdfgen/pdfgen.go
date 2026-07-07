// Package pdfgen renders application documents (arealopgørelse, beskrivelse,
// plantegning, situationsplan, ansøgningsoversigt) as PDFs. Rendering is
// deterministic: same input, same document. Every generated page carries a
// visible draft marking — Hefai prepares material, it does not approve it.
package pdfgen

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

const draftBanner = "KLADDE — udarbejdet i Hefai, kræver kontrol og godkendelse"

type doc struct {
	pdf *fpdf.Fpdf
	tr  func(string) string
}

// symbolFallbacks maps math/Greek characters outside cp1252 onto readable
// ASCII so formulas survive the core-font encoding.
var symbolFallbacks = strings.NewReplacer(
	"γ", "gamma_", "μ", "my_", "σ", "sigma_", "τ", "tau_",
	"ρ", "rho", "Δ", "delta_", "≤", "<=", "≥", ">=", "⁴", "^4", "→", "->",
)

func newDoc(orientation, title string, generatedAt time.Time) *doc {
	pdf := fpdf.New(orientation, "mm", "A4", "")
	cp1252 := pdf.UnicodeTranslatorFromDescriptor("")
	tr := func(s string) string { return cp1252(symbolFallbacks.Replace(s)) }
	pdf.SetTitle(tr(title), false)
	pdf.SetAutoPageBreak(true, 20)

	pdf.SetHeaderFunc(func() {
		pdf.SetFont("Helvetica", "B", 8)
		pdf.SetTextColor(200, 30, 30)
		w, _ := pdf.GetPageSize()
		pdf.SetXY(10, 6)
		pdf.CellFormat(w-20, 5, tr(draftBanner), "", 0, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.SetY(14)
	})
	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		pdf.SetFont("Helvetica", "I", 7)
		pdf.SetTextColor(120, 120, 120)
		w, _ := pdf.GetPageSize()
		footer := fmt.Sprintf("Genereret af Hefai %s — side %d", generatedAt.Format("02-01-2006 15:04"), pdf.PageNo())
		pdf.CellFormat(w-20, 5, tr(footer), "", 0, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
	})
	return &doc{pdf: pdf, tr: tr}
}

func (d *doc) output() ([]byte, error) {
	var buf bytes.Buffer
	if err := d.pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (d *doc) h1(text string) {
	d.pdf.SetFont("Helvetica", "B", 16)
	d.pdf.MultiCell(0, 8, d.tr(text), "", "L", false)
	d.pdf.Ln(2)
}

func (d *doc) h2(text string) {
	d.pdf.SetFont("Helvetica", "B", 12)
	d.pdf.MultiCell(0, 6, d.tr(text), "", "L", false)
	d.pdf.Ln(1)
}

func (d *doc) para(text string) {
	d.pdf.SetFont("Helvetica", "", 10)
	d.pdf.MultiCell(0, 5, d.tr(text), "", "L", false)
	d.pdf.Ln(2)
}

func (d *doc) kv(key, value string) {
	d.pdf.SetFont("Helvetica", "B", 10)
	d.pdf.CellFormat(55, 6, d.tr(key), "", 0, "L", false, 0, "")
	d.pdf.SetFont("Helvetica", "", 10)
	d.pdf.MultiCell(0, 6, d.tr(value), "", "L", false)
}

// Meta carries the shared header info for all generated documents.
type Meta struct {
	Project     *domain.Project
	CaseFile    *domain.CaseFile
	GeneratedAt time.Time
}

func (d *doc) metaBlock(m Meta) {
	d.kv("Projekt", m.Project.Name)
	if m.Project.Address != "" {
		d.kv("Adresse", m.Project.Address)
	}
	if m.Project.Municipality != "" {
		d.kv("Kommune", m.Project.Municipality)
	}
	if m.Project.CadastralID != "" {
		d.kv("Matrikel", m.Project.CadastralID)
	}
	if m.CaseFile != nil {
		d.kv("Byggesag", m.CaseFile.Title)
	}
	d.pdf.Ln(4)
}

// AreaStatement renders the arealopgørelse from measured drawing data.
func AreaStatement(m Meta, data *domain.DrawingData, drawingTitle string) ([]byte, error) {
	d := newDoc("P", "Arealopgørelse", m.GeneratedAt)
	d.pdf.AddPage()
	d.h1("Arealopgørelse")
	d.metaBlock(m)
	d.kv("Grundlag", fmt.Sprintf("Tegning: %s", drawingTitle))
	d.pdf.Ln(4)

	d.h2("Rum og arealer")
	areas := data.RoomAreas()
	names := make([]string, 0, len(areas))
	for name := range areas {
		names = append(names, name)
	}
	sort.Strings(names)

	d.pdf.SetFont("Helvetica", "B", 10)
	d.pdf.CellFormat(120, 7, d.tr("Rum"), "B", 0, "L", false, 0, "")
	d.pdf.CellFormat(40, 7, d.tr("Areal (m²)"), "B", 1, "R", false, 0, "")
	d.pdf.SetFont("Helvetica", "", 10)
	for _, name := range names {
		d.pdf.CellFormat(120, 7, d.tr(name), "", 0, "L", false, 0, "")
		d.pdf.CellFormat(40, 7, fmt.Sprintf("%.1f", areas[name]), "", 1, "R", false, 0, "")
	}
	total := data.TotalRoomAreaM2()
	d.pdf.SetFont("Helvetica", "B", 10)
	d.pdf.CellFormat(120, 8, d.tr("Samlet indvendigt areal"), "T", 0, "L", false, 0, "")
	d.pdf.CellFormat(40, 8, fmt.Sprintf("%.1f", total), "T", 1, "R", false, 0, "")
	d.pdf.Ln(6)

	if m.Project.PlotAreaM2 != nil && *m.Project.PlotAreaM2 > 0 {
		d.h2("Bebyggelsesprocent (vejledende)")
		pct := total / *m.Project.PlotAreaM2 * 100
		d.kv("Grundareal", fmt.Sprintf("%.0f m²", *m.Project.PlotAreaM2))
		d.kv("Beregnet areal", fmt.Sprintf("%.1f m²", total))
		d.kv("Bebyggelsesprocent", fmt.Sprintf("%.1f %%", pct))
		d.para("Bemærk: beregningen bruger det indvendige areal fra tegningen. Den officielle " +
			"bebyggelsesprocent opgøres efter BR18's regler om etageareal og skal bekræftes af " +
			"kommunen eller din rådgiver.")
	}
	return d.output()
}

// ProjectDescription renders the fritekst-beskrivelse as an application
// attachment.
func ProjectDescription(m Meta) ([]byte, error) {
	d := newDoc("P", "Projektbeskrivelse", m.GeneratedAt)
	d.pdf.AddPage()
	d.h1("Projektbeskrivelse")
	d.metaBlock(m)
	if m.CaseFile != nil {
		d.kv("Sagstype", caseTypeLabel(m.CaseFile.CaseType))
		d.pdf.Ln(4)
		d.h2("Beskrivelse af det ønskede byggeri")
		if m.CaseFile.Description != "" {
			d.para(m.CaseFile.Description)
		} else {
			d.para("(Ingen beskrivelse angivet endnu.)")
		}
	}
	if m.Project.Description != "" {
		d.h2("Om projektet")
		d.para(m.Project.Description)
	}
	return d.output()
}

// ApplicationSummary renders the case overview with checklist state and the
// list of generated/attached material.
func ApplicationSummary(m Meta, checklist []*domain.ComplianceCheckItem, attachments []string) ([]byte, error) {
	d := newDoc("P", "Ansøgningsoversigt", m.GeneratedAt)
	d.pdf.AddPage()
	d.h1("Ansøgningsoversigt")
	d.metaBlock(m)
	if m.CaseFile != nil {
		d.kv("Sagstype", caseTypeLabel(m.CaseFile.CaseType))
		d.kv("Status", string(m.CaseFile.Status))
		if m.CaseFile.MunicipalCaseNumber != "" {
			d.kv("Kommunens sagsnr.", m.CaseFile.MunicipalCaseNumber)
		}
	}
	d.pdf.Ln(4)

	if len(checklist) > 0 {
		d.h2("Egenkontrol (ikke-bindende)")
		d.para("Tjeklisten er en hjælp til egenkontrol og udgør ikke en myndighedsvurdering.")
		for _, item := range checklist {
			status := complianceLabel(item.Status)
			line := fmt.Sprintf("[%s] %s", status, item.Requirement)
			if item.ExpectedValue != "" {
				line += fmt.Sprintf(" — krav: %s", item.ExpectedValue)
			}
			if item.ActualValue != "" {
				line += fmt.Sprintf(", projekt: %s", item.ActualValue)
			}
			if item.SourceRef != "" {
				line += fmt.Sprintf(" (%s)", item.SourceRef)
			}
			d.pdf.SetFont("Helvetica", "", 9)
			d.pdf.MultiCell(0, 5, d.tr(line), "", "L", false)
			d.pdf.Ln(1)
		}
		d.pdf.Ln(3)
	}

	if len(attachments) > 0 {
		d.h2("Bilag")
		for i, a := range attachments {
			d.para(fmt.Sprintf("%d. %s", i+1, a))
		}
	}
	return d.output()
}

func caseTypeLabel(t domain.CaseType) string {
	switch t {
	case domain.CaseTypeNotification:
		return "Anmeldelse"
	case domain.CaseTypeBuildingPermit:
		return "Byggetilladelse"
	default:
		return "Ikke afklaret"
	}
}

func complianceLabel(s domain.ComplianceStatus) string {
	switch s {
	case domain.ComplianceOK:
		return "OK"
	case domain.ComplianceAttention:
		return "OBS"
	case domain.ComplianceNeedsConfirmation:
		return "KRÆVER BEKRÆFTELSE"
	case domain.ComplianceConfirmed:
		return "BEKRÆFTET"
	default:
		return "IKKE TJEKKET"
	}
}
