// Package confidence implements SPEC_A3-EP §7: five dimensions and weighted min.
package confidence

import (
	"math"
	"strings"
	"time"

	"github.com/valeriobitzone-png/a3-go/reject"
	"github.com/valeriobitzone-png/a3-go/truth"
)

const HalfLifeSeconds = 21600.0

type Vector struct {
	SourceReliability float64
	EvidenceStrength  float64
	Recency           float64
	Corroboration     float64
	Verification      float64
}

type Weights struct {
	SourceReliability float64
	EvidenceStrength  float64
	Recency           float64
	Corroboration     float64
	Verification      float64
}

func DefaultWeights() Weights {
	return Weights{
		SourceReliability: 1.0,
		EvidenceStrength:  1.0,
		Recency:           0.8,
		Corroboration:     0.6,
		Verification:      0.9,
	}
}

func UnitWeights() Weights {
	return Weights{1, 1, 1, 1, 1}
}

func (v Vector) Map() map[string]any {
	return map[string]any{
		"corroboration":      v.Corroboration,
		"evidence_strength":  v.EvidenceStrength,
		"recency":            v.Recency,
		"source_reliability": v.SourceReliability,
		"verification":       v.Verification,
	}
}

func (w Weights) Map() map[string]any {
	return map[string]any{
		"corroboration":      w.Corroboration,
		"evidence_strength":  w.EvidenceStrength,
		"recency":            w.Recency,
		"source_reliability": w.SourceReliability,
		"verification":       w.Verification,
	}
}

func dim(name string, x float64) error {
	if math.IsNaN(x) || math.IsInf(x, 0) || x < 0 || x > 1 {
		return reject.New("CF-DIM", "dimension "+name+" not in [0,1]")
	}
	return nil
}

func Validate(v Vector) error {
	if err := dim("source_reliability", v.SourceReliability); err != nil {
		return err
	}
	if err := dim("evidence_strength", v.EvidenceStrength); err != nil {
		return err
	}
	if err := dim("recency", v.Recency); err != nil {
		return err
	}
	if err := dim("corroboration", v.Corroboration); err != nil {
		return err
	}
	if err := dim("verification", v.Verification); err != nil {
		return err
	}
	return nil
}

func ValidateWeights(w Weights) error {
	for _, pair := range []struct {
		n string
		x float64
	}{
		{"source_reliability", w.SourceReliability},
		{"evidence_strength", w.EvidenceStrength},
		{"recency", w.Recency},
		{"corroboration", w.Corroboration},
		{"verification", w.Verification},
	} {
		if pair.x <= 0 || math.IsNaN(pair.x) || math.IsInf(pair.x, 0) {
			return reject.New("CF-W", "weight "+pair.n+" must be positive")
		}
	}
	return nil
}

func ParseVector(m map[string]any) (Vector, error) {
	get := func(k string) (float64, error) {
		if m == nil {
			return 0, reject.New("CF-DIM", "missing "+k)
		}
		raw, ok := m[k]
		if !ok {
			return 0, reject.New("CF-DIM", "missing "+k)
		}
		f, ok := raw.(float64)
		if !ok {
			return 0, reject.New("CF-DIM", "missing "+k)
		}
		return f, nil
	}
	var v Vector
	var err error
	if v.SourceReliability, err = get("source_reliability"); err != nil {
		return Vector{}, err
	}
	if v.EvidenceStrength, err = get("evidence_strength"); err != nil {
		return Vector{}, err
	}
	if v.Recency, err = get("recency"); err != nil {
		return Vector{}, err
	}
	if v.Corroboration, err = get("corroboration"); err != nil {
		return Vector{}, err
	}
	if v.Verification, err = get("verification"); err != nil {
		return Vector{}, err
	}
	return v, Validate(v)
}

func ParseWeights(m map[string]any) (Weights, error) {
	get := func(k string) (float64, error) {
		raw, ok := m[k]
		if !ok {
			return 0, reject.New("CF-W", "missing "+k)
		}
		f, ok := raw.(float64)
		if !ok {
			return 0, reject.New("CF-W", "missing "+k)
		}
		return f, nil
	}
	var w Weights
	var err error
	if w.SourceReliability, err = get("source_reliability"); err != nil {
		return Weights{}, err
	}
	if w.EvidenceStrength, err = get("evidence_strength"); err != nil {
		return Weights{}, err
	}
	if w.Recency, err = get("recency"); err != nil {
		return Weights{}, err
	}
	if w.Corroboration, err = get("corroboration"); err != nil {
		return Weights{}, err
	}
	if w.Verification, err = get("verification"); err != nil {
		return Weights{}, err
	}
	return w, ValidateWeights(w)
}

// Score is min_i(dimension_i * weight_i). A zero dimension yields 0.
func Score(v Vector, w Weights) (float64, error) {
	if err := Validate(v); err != nil {
		return 0, err
	}
	if err := ValidateWeights(w); err != nil {
		return 0, err
	}
	if v.SourceReliability == 0 || v.EvidenceStrength == 0 || v.Recency == 0 || v.Corroboration == 0 || v.Verification == 0 {
		return 0, nil
	}
	products := []float64{
		v.SourceReliability * w.SourceReliability,
		v.EvidenceStrength * w.EvidenceStrength,
		v.Recency * w.Recency,
		v.Corroboration * w.Corroboration,
		v.Verification * w.Verification,
	}
	min := products[0]
	for _, p := range products[1:] {
		if p < min {
			min = p
		}
	}
	return min, nil
}

func Combine(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func Recency(deltaSeconds, halfLife float64) float64 {
	if halfLife <= 0 {
		return 0
	}
	if deltaSeconds < 0 {
		deltaSeconds = 0
	}
	return math.Pow(2, -deltaSeconds/halfLife)
}

func RecencyBetween(observe, present time.Time, halfLife float64) float64 {
	return Recency(present.Sub(observe).Seconds(), halfLife)
}

func Corroboration(sourceIDs []string) float64 {
	seen := map[string]struct{}{}
	for _, id := range sourceIDs {
		seen[strings.ToLower(id)] = struct{}{}
	}
	n := len(seen)
	switch {
	case n <= 1:
		return 0
	case n == 2:
		return 0.5
	default:
		return 1
	}
}

func SourceReliability(p truth.Provenance) float64 {
	switch p {
	case truth.ObservedSigned:
		return 1
	case truth.HumanAdmitted:
		return 0.8
	case truth.Inferred:
		return 0.5
	case truth.DerivedModel:
		return 0.3
	default:
		return 0
	}
}

func VerificationDegree(kind string) float64 {
	switch strings.ToLower(kind) {
	case "deterministic":
		return 1
	case "real":
		return 0.8
	case "sandbox":
		return 0.3
	default:
		return 0
	}
}

func Category(score float64, class truth.Class) string {
	if score == 0 {
		return "unknown"
	}
	if score >= 0.8 && class == truth.FACT {
		return "high"
	}
	if score >= 0.5 {
		return "medium"
	}
	return "low"
}

func Assessment(v Vector, w Weights, class truth.Class) (map[string]any, error) {
	s, err := Score(v, w)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"category":    Category(s, class),
		"score":       s,
		"truth_class": string(class),
		"vector":      v.Map(),
		"weights":     w.Map(),
	}, nil
}

func RejectCompensatingMean(claimed, lawful float64, aggregation string) error {
	if strings.ToLower(aggregation) == "mean" || claimed != lawful {
		return reject.New("CF-007", "confidence aggregated with compensating average")
	}
	return nil
}
