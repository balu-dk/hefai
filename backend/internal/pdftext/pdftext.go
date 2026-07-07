// Package pdftext udtrækker tekst fra tekstbaserede PDF'er (fx lokalplaner
// fra plandata.dk). Skannede dokumenter uden tekstlag giver tom/kort tekst —
// kalderen skal behandle det ærligt og bede om manuel indtastning frem for
// at gætte.
package pdftext

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/ledongthuc/pdf"
)

// Extract returnerer dokumentets tekst med sideskift som tomme linjer.
func Extract(data []byte) (string, error) {
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("kunne ikke læse PDF: %w", err)
	}

	var out strings.Builder
	for pageNo := 1; pageNo <= reader.NumPage(); pageNo++ {
		page := reader.Page(pageNo)
		if page.V.IsNull() {
			continue
		}
		text, err := pageText(page)
		if err != nil {
			continue // enkelte defekte sider vælter ikke hele udtrækket
		}
		if strings.TrimSpace(text) != "" {
			out.WriteString(strings.TrimSpace(text))
			out.WriteString("\n\n")
		}
	}
	return strings.TrimSpace(out.String()), nil
}

// pageText samler sidens tekst i læserækkefølge: rækker oppefra og ned,
// glyffer venstre mod højre. Mange PDF'er (bl.a. plandata-lokalplaner)
// positionerer hver glyf separat, så ord genskabes ud fra X-afstanden:
// mellemrum indsættes kun ved reelle huller i skriftbilledet.
func pageText(page pdf.Page) (_ string, err error) {
	// Biblioteket kan panikke på defekte fonte; det oversættes til en fejl.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("siden kunne ikke afkodes: %v", r)
		}
	}()

	rows, err := page.GetTextByRow()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, row := range rows {
		glyphs := make([]pdf.Text, len(row.Content))
		copy(glyphs, row.Content)
		sort.SliceStable(glyphs, func(i, j int) bool { return glyphs[i].X < glyphs[j].X })

		var line strings.Builder
		prevEnd := 0.0
		for i, g := range glyphs {
			if g.S == "" {
				continue
			}
			if i > 0 && line.Len() > 0 {
				gap := g.X - prevEnd
				// Tærskel skaleret efter skriftstørrelsen; faldback for
				// glyffer uden bredde-/størrelsesinfo.
				threshold := g.FontSize * 0.22
				if threshold <= 0 {
					threshold = 1.5
				}
				if gap > threshold {
					line.WriteString(" ")
				}
			}
			line.WriteString(g.S)
			prevEnd = g.X + g.W
		}
		if text := strings.TrimSpace(line.String()); text != "" {
			b.WriteString(text)
			b.WriteString("\n")
		}
	}
	return b.String(), nil
}
