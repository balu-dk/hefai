package pdftext

import (
	"bytes"
	"strings"
	"testing"

	"github.com/go-pdf/fpdf"
)

// Testen genererer en rigtig PDF med fpdf og udtrækker teksten igen.
func TestExtractRoundtrip(t *testing.T) {
	doc := fpdf.New("P", "mm", "A4", "")
	tr := doc.UnicodeTranslatorFromDescriptor("")
	doc.AddPage()
	doc.SetFont("Helvetica", "", 12)
	doc.MultiCell(0, 6, tr("§ 6.2 Bebyggelsesprocenten må ikke overstige 15 for sommerhusgrunde."), "", "L", false)
	doc.AddPage()
	doc.MultiCell(0, 6, tr("§ 7.1 Bygninger skal holdes mindst 5,0 m fra skel."), "", "L", false)

	var buf bytes.Buffer
	if err := doc.Output(&buf); err != nil {
		t.Fatal(err)
	}

	text, err := Extract(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Bebyggelsesprocenten", "15", "5,0 m fra skel"} {
		if !strings.Contains(text, want) {
			t.Errorf("extracted text missing %q:\n%s", want, text)
		}
	}
	// Sider skal være adskilt så §-chunkeren kan finde afsnittene.
	if !strings.Contains(text, "\n\n") {
		t.Error("pages must be separated by blank lines")
	}
}

func TestExtractRejectsGarbage(t *testing.T) {
	if _, err := Extract([]byte("ikke en pdf")); err == nil {
		t.Error("garbage accepted as PDF")
	}
}
