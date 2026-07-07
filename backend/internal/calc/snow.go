package calc

import (
	"encoding/json"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

// SnowLoadInput describes the roof for the characteristic snow load.
type SnowLoadInput struct {
	RoofAngleDeg float64 `json:"roofAngleDeg"`
	// Exposure/thermal coefficients default to 1.0 (normal topography,
	// normal thermal transmittance) and can be overridden explicitly.
	ExposureCe *float64 `json:"exposureCe,omitempty"`
	ThermalCt  *float64 `json:"thermalCt,omitempty"`
}

// Characteristic ground snow load for Denmark per the Danish National Annex.
const groundSnowLoadDK = 0.9 // kN/m²

// runSnowLoad computes s = μ1 · Ce · Ct · sk per EN 1991-1-3.
func runSnowLoad(raw json.RawMessage) (*Outcome, error) {
	var in SnowLoadInput
	if err := decodeInputs(raw, &in); err != nil {
		return nil, err
	}
	if in.RoofAngleDeg < 0 || in.RoofAngleDeg > 90 {
		return nil, domain.Validation("taghældning skal være mellem 0 og 90 grader")
	}

	ce, ct := 1.0, 1.0
	if in.ExposureCe != nil {
		ce = *in.ExposureCe
	}
	if in.ThermalCt != nil {
		ct = *in.ThermalCt
	}
	if ce <= 0 || ct <= 0 {
		return nil, domain.Validation("Ce og Ct skal være positive")
	}

	// Shape coefficient μ1 for monopitch/duopitch roofs (EN 1991-1-3 tab. 5.2):
	// 0.8 up to 30°, linear to 0 at 60°, 0 above.
	var mu1 float64
	switch {
	case in.RoofAngleDeg <= 30:
		mu1 = 0.8
	case in.RoofAngleDeg < 60:
		mu1 = 0.8 * (60 - in.RoofAngleDeg) / 30
	default:
		mu1 = 0
	}

	s := mu1 * ce * ct * groundSnowLoadDK

	inputsJSON, _ := json.Marshal(in)
	return &Outcome{
		Method:            "snow_load_dk_v1",
		MethodVersion:     "1",
		StandardReference: "DS/EN 1991-1-3 + DK NA",
		Inputs:            inputsJSON,
		Assumptions: []Assumption{
			{Text: "Karakteristisk terrænværdi sk = 0,9 kN/m² (Danmark). Lokale forhold (fx Bornholm) kan afvige — SKAL bekræftes af statiker.",
				Reference: "DS/EN 1991-1-3 DK NA"},
			{Text: "Formfaktor μ1 for sadeltag/ensidigt tag efter tabel 5.2 uden snesækkeeffekter (ingen højdespring, ingen ophobning mod højere bygning).",
				Reference: "DS/EN 1991-1-3, tabel 5.2"},
			{Text: "Eksponeringsfaktor Ce og termisk faktor Ct som angivet (standard 1,0 — normal topografi og normalt isoleret tag).",
				Reference: "DS/EN 1991-1-3, 5.2(7)"},
		},
		Results: map[string]any{
			"mu1":                round(mu1, 3),
			"groundSnowLoadKNM2": groundSnowLoadDK,
			"roofSnowLoadKNM2":   round(s, 3),
			"formula":            "s = μ1 · Ce · Ct · sk",
		},
		Advisory: true,
		Notice:   advisoryNotice,
	}, nil
}
