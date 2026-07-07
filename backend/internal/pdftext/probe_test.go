package pdftext

import (
	"os"
	"strings"
	"testing"
)

// Manuel probe mod en rigtig PDF: go test -run TestProbeRealPDF med
// PDFTEXT_PROBE=/sti/til.pdf. Springes over i normale kørsler.
func TestProbeRealPDF(t *testing.T) {
	path := os.Getenv("PDFTEXT_PROBE")
	if path == "" {
		t.Skip("PDFTEXT_PROBE ikke sat")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text, err := Extract(data)
	if err != nil {
		t.Fatal(err)
	}
	lower := strings.ToLower(text)
	for _, needle := range []string{"bebyggelsesprocent", "skel", "lokalplan"} {
		t.Logf("%-20s: %v", needle, strings.Contains(lower, needle))
	}
	if idx := strings.Index(lower, "bebyggelsesprocent"); idx >= 0 {
		start := max(0, idx-60)
		t.Logf("KONTEKST: %s", strings.ReplaceAll(text[start:min(len(text), idx+160)], "\n", " "))
	}
}
