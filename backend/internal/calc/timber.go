package calc

import (
	"encoding/json"
	"fmt"

	"github.com/balu-dk/hefai/backend/internal/domain"
)

// TimberBeamInput describes a simply supported beam with uniformly
// distributed load — the classic joist/rafter estimate.
type TimberBeamInput struct {
	SpanM             float64 `json:"spanM"`
	SpacingM          float64 `json:"spacingM"`          // c/c distance between beams
	PermanentLoadKNM2 float64 `json:"permanentLoadKNM2"` // egenlast (incl. beam self-weight approximation)
	VariableLoadKNM2  float64 `json:"variableLoadKNM2"`  // nyttelast/snelast, dominerende variabel last
	TimberClass       string  `json:"timberClass"`       // C16, C18, C24, C30
	WidthMM           float64 `json:"widthMm"`
	HeightMM          float64 `json:"heightMm"`
	// SuggestDimension: additionally scan standard sizes for the smallest
	// section where all utilizations ≤ 1.
	SuggestDimension bool `json:"suggestDimension"`
}

// Characteristic strengths for structural timber, EN 338.
type timberClass struct {
	fmk   float64 // bending strength [MPa]
	fvk   float64 // shear strength [MPa]
	eMean float64 // mean modulus of elasticity [MPa]
}

var timberClasses = map[string]timberClass{
	"C16": {fmk: 16, fvk: 3.2, eMean: 8000},
	"C18": {fmk: 18, fvk: 3.4, eMean: 9000},
	"C24": {fmk: 24, fvk: 4.0, eMean: 11000},
	"C30": {fmk: 30, fvk: 4.0, eMean: 12000},
}

// Partial and modification factors (service class 1/2, medium-term load).
const (
	gammaM         = 1.3  // solid timber, EC5 DK NA
	kMod           = 0.8  // medium-term, service class 1/2
	kCr            = 0.67 // crack factor for shear, EC5 6.1.7
	gammaG         = 1.2  // permanent, EN 1990 DK NA (6.10b-agtig kombination)
	gammaQ         = 1.5  // variable
	deflectionLimit = 300.0 // L/300 for u_inst
)

var standardSizes = [][2]float64{
	{45, 95}, {45, 120}, {45, 145}, {45, 170}, {45, 195}, {45, 220}, {45, 245},
	{70, 145}, {70, 170}, {70, 195}, {70, 220}, {70, 245},
}

type beamCheck struct {
	BendingUtilization    float64 `json:"bendingUtilization"`
	ShearUtilization      float64 `json:"shearUtilization"`
	DeflectionUtilization float64 `json:"deflectionUtilization"`
	DeflectionMM          float64 `json:"deflectionMm"`
	OK                    bool    `json:"ok"`
}

func checkBeam(in TimberBeamInput, cls timberClass, widthMM, heightMM float64) beamCheck {
	b := widthMM / 1000  // m
	h := heightMM / 1000 // m

	// Design line load [kN/m].
	wd := (gammaG*in.PermanentLoadKNM2 + gammaQ*in.VariableLoadKNM2) * in.SpacingM
	// Serviceability (characteristic) line load [kN/m].
	wser := (in.PermanentLoadKNM2 + in.VariableLoadKNM2) * in.SpacingM

	// Bending: σ = M/W ≤ fm,d
	md := wd * in.SpanM * in.SpanM / 8          // kNm
	sectionModulus := b * h * h / 6             // m³
	sigma := md / 1000 / sectionModulus         // MN/m² = MPa
	fmd := kMod * cls.fmk / gammaM              // MPa
	// Shear: τ = 1.5·V/(kcr·b·h) ≤ fv,d
	vd := wd * in.SpanM / 2                     // kN
	tau := 1.5 * vd / 1000 / (kCr * b * h)      // MPa
	fvd := kMod * cls.fvk / gammaM              // MPa
	// Deflection: u_inst = 5wL⁴/(384EI) ≤ L/300
	inertia := b * h * h * h / 12                                    // m⁴
	uInst := 5 * wser * 1000 * pow4(in.SpanM) / (384 * cls.eMean * 1e6 * inertia) // m
	uLimit := in.SpanM / deflectionLimit

	check := beamCheck{
		BendingUtilization:    round(sigma/fmd, 3),
		ShearUtilization:      round(tau/fvd, 3),
		DeflectionUtilization: round(uInst/uLimit, 3),
		DeflectionMM:          round(uInst*1000, 1),
	}
	check.OK = check.BendingUtilization <= 1 && check.ShearUtilization <= 1 && check.DeflectionUtilization <= 1
	return check
}

func pow4(v float64) float64 { return v * v * v * v }

func runTimberBeam(raw json.RawMessage) (*Outcome, error) {
	var in TimberBeamInput
	if err := decodeInputs(raw, &in); err != nil {
		return nil, err
	}
	switch {
	case in.SpanM <= 0 || in.SpanM > 8:
		return nil, domain.Validation("spændvidden skal være mellem 0 og 8 m — større spænd er uden for overslagets gyldighed, kontakt statiker")
	case in.SpacingM <= 0 || in.SpacingM > 2:
		return nil, domain.Validation("c/c-afstanden skal være mellem 0 og 2 m")
	case in.PermanentLoadKNM2 < 0 || in.VariableLoadKNM2 < 0:
		return nil, domain.Validation("laster kan ikke være negative")
	case in.PermanentLoadKNM2+in.VariableLoadKNM2 == 0:
		return nil, domain.Validation("mindst én last skal være angivet")
	case in.WidthMM <= 0 || in.HeightMM <= 0:
		return nil, domain.Validation("tværsnittet skal have positiv bredde og højde")
	}
	cls, ok := timberClasses[in.TimberClass]
	if !ok {
		return nil, domain.Validation("træklasse skal være C16, C18, C24 eller C30")
	}

	check := checkBeam(in, cls, in.WidthMM, in.HeightMM)

	results := map[string]any{
		"designLineLoadKNM":  round((gammaG*in.PermanentLoadKNM2+gammaQ*in.VariableLoadKNM2)*in.SpacingM, 3),
		"designMomentKNM":    round((gammaG*in.PermanentLoadKNM2+gammaQ*in.VariableLoadKNM2)*in.SpacingM*in.SpanM*in.SpanM/8, 3),
		"check":              check,
		"formulaBending":     "σ = (w·L²/8)/(b·h²/6) ≤ kmod·fm,k/γM",
		"formulaShear":       "τ = 1,5·(w·L/2)/(kcr·b·h) ≤ kmod·fv,k/γM",
		"formulaDeflection":  "u = 5·w·L⁴/(384·E·I) ≤ L/300",
	}

	if in.SuggestDimension {
		for _, size := range standardSizes {
			if c := checkBeam(in, cls, size[0], size[1]); c.OK {
				results["suggestedSection"] = map[string]any{
					"widthMm": size[0], "heightMm": size[1], "check": c,
				}
				break
			}
		}
		if _, found := results["suggestedSection"]; !found {
			results["suggestedSection"] = nil
			results["suggestionNote"] = "Ingen standarddimension (op til 70×245) klarer kravene i ét lag. " +
				"Overvej mindre c/c-afstand, limtræ/LVL eller stål — det kræver statikerens vurdering."
		}
	}

	inputsJSON, _ := json.Marshal(in)
	return &Outcome{
		Method:            "timber_beam_udl_v1",
		MethodVersion:     "1",
		StandardReference: "DS/EN 1995-1-1 (Eurocode 5) + DK NA",
		Inputs:            inputsJSON,
		Assumptions: []Assumption{
			{Text: "Simpelt understøttet bjælke med jævnt fordelt last, enkelt spænd, ingen indspænding og ingen huller/udveksling.",
				Reference: "Statisk model"},
			{Text: fmt.Sprintf("Lastkombination γG·G + γQ·Q med γG = %.1f og γQ = %.1f (KFI = 1,0, konsekvensklasse CC2).", gammaG, gammaQ),
				Reference: "DS/EN 1990 DK NA"},
			{Text: fmt.Sprintf("kmod = %.1f (mellemlang lastvarighed, anvendelsesklasse 1/2) og γM = %.1f for konstruktionstræ.", kMod, gammaM),
				Reference: "DS/EN 1995-1-1, tabel 3.1 og DK NA"},
			{Text: "Kipning (vipning) er IKKE eftervist — forudsætter fastholdt trykside (fx fastgjort gulv/tagplade). Skal bekræftes.",
				Reference: "DS/EN 1995-1-1, 6.3.3"},
			{Text: fmt.Sprintf("Nedbøjningskrav u_inst ≤ L/%d med karakteristisk lastkombination; krybning (u_fin) er ikke medregnet.", int(deflectionLimit)),
				Reference: "DS/EN 1995-1-1, 7.2 + DK NA"},
			{Text: fmt.Sprintf("Styrketal for %s efter EN 338 (fm,k = %.0f MPa, fv,k = %.1f MPa, E = %.0f MPa).", in.TimberClass, cls.fmk, cls.fvk, cls.eMean),
				Reference: "DS/EN 338"},
		},
		Results:  results,
		Advisory: true,
		Notice:   advisoryNotice,
	}, nil
}
