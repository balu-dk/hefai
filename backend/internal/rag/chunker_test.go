package rag

import (
	"strings"
	"testing"
)

func TestSplitTracksSectionRefs(t *testing.T) {
	text := `Kapitel 8 Byggeret

Generelle bestemmelser om byggeret.

§ 180
Afstand til skel skal være mindst 2,5 m for sommerhuse.

Mere tekst om skelafstand og hvad der gælder.

§ 181, stk. 2
Højden må ikke overstige 5,0 m.`

	chunks := Split(text)
	if len(chunks) == 0 {
		t.Fatal("no chunks produced")
	}

	foundSkel, foundHeight := false, false
	for _, c := range chunks {
		if strings.Contains(c.Content, "skel skal være mindst") {
			foundSkel = true
			if !strings.HasPrefix(c.SectionRef, "§ 180") {
				t.Errorf("skel chunk should cite § 180, got %q", c.SectionRef)
			}
		}
		if strings.Contains(c.Content, "5,0 m") {
			foundHeight = true
			if !strings.HasPrefix(c.SectionRef, "§ 181") {
				t.Errorf("height chunk should cite § 181, got %q", c.SectionRef)
			}
		}
	}
	if !foundSkel || !foundHeight {
		t.Error("expected content missing from chunks")
	}
}

func TestSplitBoundsChunkSize(t *testing.T) {
	long := strings.Repeat("Meget lang paragraf uden afsnit. ", 300) // ~10k chars
	chunks := Split(long)
	if len(chunks) < 2 {
		t.Fatalf("oversized text should split into multiple chunks, got %d", len(chunks))
	}
	for i, c := range chunks {
		if n := len([]rune(c.Content)); n > maxChunkSize {
			t.Errorf("chunk %d has %d runes, max %d", i, n, maxChunkSize)
		}
		if c.Index != i {
			t.Errorf("chunk %d has index %d", i, c.Index)
		}
	}
}

func TestSplitEmpty(t *testing.T) {
	if chunks := Split("   \n\n  "); len(chunks) != 0 {
		t.Errorf("blank input should yield no chunks, got %d", len(chunks))
	}
}
