// SPDX-License-Identifier: Apache-2.0
// Part of the A3 universe. See LICENSE.
// Package truth implements SPEC_A3-EP §3–4 truth class, provenance, and promotion.
package truth

import (
	"strings"

	"github.com/valeriobitzone-png/a3-go/reject"
)

type Class string

const (
	FACT        Class = "fact"
	OBSERVATION Class = "observation"
	HYPOTHESIS  Class = "hypothesis"
	UNKNOWN     Class = "unknown"
)

type Provenance string

const (
	ObservedSigned Provenance = "observed_signed"
	DerivedModel   Provenance = "derived_model"
	Inferred       Provenance = "inferred"
	HumanAdmitted  Provenance = "human_admitted"
)

type Environment string

const (
	REAL    Environment = "real"
	SANDBOX Environment = "sandbox"
)

type Bearer struct {
	Class      Class
	Provenance Provenance
	Ref        string
}

func (b Bearer) Map() map[string]any {
	m := map[string]any{
		"provenance":  string(b.Provenance),
		"truth_class": string(b.Class),
	}
	if b.Ref != "" {
		m["ref"] = b.Ref
	}
	return m
}

type Verification struct {
	VerificationID string
	AdmittedID     string
	Environment    Environment
}

func (v Verification) Map() map[string]any {
	return map[string]any{
		"admitted_id":     v.AdmittedID,
		"environment":     string(v.Environment),
		"verification_id": v.VerificationID,
	}
}

func parseClass(s string) (Class, error) {
	switch Class(strings.ToLower(s)) {
	case FACT, OBSERVATION, HYPOTHESIS, UNKNOWN:
		return Class(strings.ToLower(s)), nil
	default:
		return "", reject.New("TR-003", "unknown truth class")
	}
}

func parseProvenance(s string) (Provenance, error) {
	switch Provenance(strings.ToLower(s)) {
	case ObservedSigned, DerivedModel, Inferred, HumanAdmitted:
		return Provenance(strings.ToLower(s)), nil
	default:
		return "", reject.New("TR-003", "unknown provenance")
	}
}

func ParseBearer(m map[string]any) (Bearer, error) {
	if m == nil {
		return Bearer{}, reject.New("TR-003", "missing truth")
	}
	cs, ok := m["truth_class"].(string)
	if !ok || cs == "" {
		return Bearer{}, reject.New("TR-003", "missing truthClass")
	}
	ps, ok := m["provenance"].(string)
	if !ok || ps == "" {
		return Bearer{}, reject.New("TR-003", "missing provenance")
	}
	c, err := parseClass(cs)
	if err != nil {
		return Bearer{}, err
	}
	p, err := parseProvenance(ps)
	if err != nil {
		return Bearer{}, err
	}
	ref, _ := m["ref"].(string)
	return Bearer{Class: c, Provenance: p, Ref: ref}, nil
}

func RequireBearer(class, provenance, ref string) (Bearer, error) {
	if strings.TrimSpace(class) == "" {
		return Bearer{}, reject.New("TR-003", "missing truthClass")
	}
	if strings.TrimSpace(provenance) == "" {
		return Bearer{}, reject.New("TR-003", "missing provenance")
	}
	c, err := parseClass(class)
	if err != nil {
		return Bearer{}, err
	}
	p, err := parseProvenance(provenance)
	if err != nil {
		return Bearer{}, err
	}
	return Bearer{Class: c, Provenance: p, Ref: ref}, nil
}

// FromInsufficient is UNKNOWN. No silent default to FACT or HYPOTHESIS.
func FromInsufficient(ref string) Bearer {
	return Bearer{Class: UNKNOWN, Provenance: Inferred, Ref: ref}
}

// FromProcessOutcome classifies exit 0 / printed SUCCESS as OBSERVATION, never FACT.
func FromProcessOutcome(exitCode int, printed, ref string) Bearer {
	_ = exitCode
	_ = printed
	return Bearer{Class: OBSERVATION, Provenance: ObservedSigned, Ref: ref}
}

func FromSandbox(ref string) Bearer {
	return Bearer{Class: OBSERVATION, Provenance: ObservedSigned, Ref: ref}
}

func FromRealAdmitted(ref string) Bearer {
	return Bearer{Class: FACT, Provenance: ObservedSigned, Ref: ref}
}

// Promote applies TR-001. Without REAL VerificationAdmitted the hypothesis is unchanged.
func Promote(hypothesis Bearer, v *Verification) (Bearer, error) {
	if hypothesis.Class != HYPOTHESIS {
		return Bearer{}, reject.New("TR-001", "promote requires HYPOTHESIS")
	}
	if v == nil {
		return hypothesis, nil
	}
	if v.Environment == SANDBOX {
		return Bearer{Class: OBSERVATION, Provenance: ObservedSigned, Ref: v.VerificationID}, nil
	}
	if v.Environment != REAL {
		return hypothesis, nil
	}
	return Bearer{Class: FACT, Provenance: ObservedSigned, Ref: v.VerificationID}, nil
}

func RejectInventedUnknown(class Class, proposition any) error {
	if class == UNKNOWN && proposition != nil && proposition != "" {
		return reject.New("CF-005", "UNKNOWN with invented conclusion (proposition not null)")
	}
	return nil
}

// Receipt is an execution receipt: what a process, tool, HTTP API or MCP
// server answered. Its content is kept for the record; it is never read to
// decide a truth class.
type Receipt struct {
	Channel  string // "process", "tool", "http", "mcp", ...
	ExitCode *int   // process receipts only
	Printed  string // process output, if any
	Payload  any    // structured result, if any
}

// FromReceipt applies SPEC_A3-EP §3: an execution receipt is OBSERVATION,
// whatever it contains.
func FromReceipt(r Receipt, ref string) Bearer {
	return Bearer{Class: OBSERVATION, Provenance: ObservedSigned, Ref: ref}
}

// AdmitOnReceipt admits a claimed class whose basis is a receipt. FACT is
// rejected with CF-001 whatever the receipt contains; any other claim gets
// the lawful classification, OBSERVATION.
func AdmitOnReceipt(claimed Class, r Receipt, ref string) (Bearer, error) {
	if claimed == FACT {
		return Bearer{}, rejectReceiptFact()
	}
	return FromReceipt(r, ref), nil
}

func rejectReceiptFact() error {
	return reject.New("CF-001", "FACT from an execution receipt (a receipt is OBSERVATION whatever it contains)")
}

// RejectReceiptFact rejects FACT on a receipt (CF-001). The exit code and the
// printed text are accepted for compatibility and deliberately ignored: until
// 2026-09-23 this function rejected only exit 0 + "SUCCESS", so a receipt
// printing "OK" passed as FACT. SPEC_A3-EP §2, §3, §10: receipt ≠ fact.
func RejectReceiptFact(class Class, exitCode int, printed string) error {
	_, _ = exitCode, printed
	if class == FACT {
		return rejectReceiptFact()
	}
	return nil
}

func RejectSandboxFact(class Class, env Environment) error {
	if class == FACT && env == SANDBOX {
		return reject.New("CF-002", "FACT on sandbox (not REAL)")
	}
	return nil
}

func RejectPromotionWithoutVerification(claimed Class, verification any) error {
	if claimed == FACT && verification == nil {
		return reject.New("CF-003", "HYPOTHESIS promoted to FACT without VerificationAdmitted")
	}
	return nil
}

type Axis struct {
	Support   string
	Freshness string
	Status    string
	Action    string
}

func (a Axis) Map() map[string]any {
	return map[string]any{
		"action":    a.Action,
		"freshness": a.Freshness,
		"status":    a.Status,
		"support":   a.Support,
	}
}

// ProjectAxis is declarative and does not import a3ui.
func ProjectAxis(class Class, freshness string) Axis {
	switch class {
	case UNKNOWN:
		return Axis{Support: "unknown", Freshness: "fresh", Status: "unknown", Action: "unknown"}
	case HYPOTHESIS:
		return Axis{Support: "medium", Freshness: "fresh", Status: "held", Action: "na"}
	case FACT:
		if freshness == "stale" {
			return Axis{Support: "high", Freshness: "stale", Status: "held", Action: "na"}
		}
		return Axis{Support: "high", Freshness: "fresh", Status: "believed", Action: "na"}
	default:
		return Axis{Support: "high", Freshness: freshness, Status: "held", Action: "na"}
	}
}

func RequireAxisCoherent(class Class, axis Axis) error {
	if class == HYPOTHESIS && axis.Status == "believed" {
		return reject.New("TR-006", "HYPOTHESIS must not be BELIEVED")
	}
	if class == UNKNOWN && axis.Status == "believed" {
		return reject.New("TR-006", "UNKNOWN must not be BELIEVED")
	}
	if class == FACT && axis.Freshness == "stale" && axis.Status == "believed" {
		return reject.New("TR-006", "stale FACT must not be BELIEVED")
	}
	return nil
}

func LockVectors() map[string]any {
	hyp := Bearer{Class: HYPOTHESIS, Provenance: DerivedModel, Ref: "model:slot"}
	fact, _ := Promote(hyp, &Verification{VerificationID: "ver-1", AdmittedID: "obs-verify", Environment: REAL})
	return map[string]any{
		"axis_fact_fresh": ProjectAxis(FACT, "fresh").Map(),
		"axis_fact_stale": ProjectAxis(FACT, "stale").Map(),
		"axis_hypothesis": ProjectAxis(HYPOTHESIS, "fresh").Map(),
		"axis_unknown":    ProjectAxis(UNKNOWN, "fresh").Map(),
		"fact_promoted":   fact.Map(),
		"hypothesis":      hyp.Map(),
		"receipt":         FromProcessOutcome(0, "SUCCESS", "exit:0:SUCCESS").Map(),
		"sandbox":         FromSandbox("ver-sand").Map(),
		"unknown":         FromInsufficient("gap:price").Map(),
		"verification": Verification{
			VerificationID: "ver-1",
			AdmittedID:     "obs-verify",
			Environment:    REAL,
		}.Map(),
	}
}
