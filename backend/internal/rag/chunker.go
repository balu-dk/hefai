// Package rag prepares source material (BR18, lokalplaner, kommunale krav)
// for retrieval: chunking with section detection now, embeddings later once
// a provider is chosen. Retrieval itself is PostgreSQL full-text search.
package rag

import (
	"regexp"
	"strings"
)

// Chunk is one retrievable passage of a source document.
type Chunk struct {
	Index      int
	Content    string
	SectionRef string // e.g. "§ 180" or "Kapitel 8" when detected
}

const (
	targetChunkSize = 1200 // runes; roughly a few paragraphs
	maxChunkSize    = 2000
)

// Matches Danish regulation section headings: "§ 180", "§180, stk. 2",
// "Kapitel 8", "8.4.2 Overskrift".
var sectionRe = regexp.MustCompile(`(?m)^\s*(§+\s*\d+[a-z]?(?:,?\s*stk\.\s*\d+)?|[Kk]apitel\s+\d+|\d+(?:\.\d+)+\s+\S.*)`)

// Split breaks text into chunks along paragraph boundaries, keeping track of
// the most recent section heading so every chunk can cite its origin.
func Split(text string) []Chunk {
	paragraphs := splitParagraphs(text)
	var chunks []Chunk
	var buf strings.Builder
	currentRef := ""
	bufRef := ""

	flush := func() {
		content := strings.TrimSpace(buf.String())
		if content != "" {
			chunks = append(chunks, Chunk{Index: len(chunks), Content: content, SectionRef: bufRef})
		}
		buf.Reset()
	}

	for _, p := range paragraphs {
		if m := sectionRe.FindString(p); m != "" {
			// A new section starts: close the running chunk so every chunk
			// cites exactly one section.
			if buf.Len() > 0 {
				flush()
			}
			currentRef = normalizeRef(m)
		}
		// Oversized single paragraphs are split hard to keep chunks bounded.
		for len([]rune(p)) > maxChunkSize {
			runes := []rune(p)
			cut := maxChunkSize
			if idx := strings.LastIndex(string(runes[:cut]), " "); idx > maxChunkSize/2 {
				cut = len([]rune(string(runes[:cut])[:idx]))
			}
			if buf.Len() > 0 {
				flush()
			}
			bufRef = currentRef
			buf.WriteString(string(runes[:cut]))
			flush()
			p = strings.TrimSpace(string(runes[cut:]))
		}
		if buf.Len() > 0 && len([]rune(buf.String()))+len([]rune(p)) > targetChunkSize {
			flush()
		}
		if buf.Len() == 0 {
			bufRef = currentRef
		}
		if buf.Len() > 0 {
			buf.WriteString("\n\n")
		}
		buf.WriteString(p)
	}
	flush()
	return chunks
}

func splitParagraphs(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	raw := regexp.MustCompile(`\n{2,}`).Split(text, -1)
	var out []string
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func normalizeRef(heading string) string {
	heading = strings.TrimSpace(heading)
	if len([]rune(heading)) > 80 {
		heading = string([]rune(heading)[:80])
	}
	return heading
}
