// Package envelope implements SPEC_A3-EP §6 CloudEvents lock v2 and §8 attestation.
package envelope

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/valeriobitzone-png/a3-go/jcs"
	"github.com/valeriobitzone-png/a3-go/reject"
	"github.com/valeriobitzone-png/a3-go/temporal"
	"github.com/valeriobitzone-png/a3-go/truth"
)

const (
	SpecVersion     = "1.0"
	DataContentType = "application/json"

	TypeBelief = "io.a3ep.belief.admitted"
	TypeAction = "io.a3ep.action.authorized"
	TypeEnv    = "io.a3ep.env.postcondition"

	typeBeliefV1 = "a3.belief.admitted"
	typeObsV1    = "a3.observation.admitted"
)

type Attestation struct {
	AttesterID  string
	RequesterID string
}

func (a Attestation) Map() map[string]any {
	return map[string]any{
		"attester_id":  a.AttesterID,
		"requester_id": a.RequesterID,
	}
}

func ParseAttestation(m map[string]any) *Attestation {
	if m == nil {
		return nil
	}
	att, _ := m["attester_id"].(string)
	req, _ := m["requester_id"].(string)
	if att == "" && req == "" {
		return nil
	}
	return &Attestation{AttesterID: att, RequesterID: req}
}

func ValidateAttestation(typ string, att *Attestation) error {
	if att == nil {
		return nil
	}
	if typ == TypeAction && att.AttesterID == att.RequesterID {
		return reject.New("CF-004", "attester_id must not equal requester_id on irreversible action")
	}
	return nil
}

func NormalizeType(t string) (string, error) {
	switch t {
	case typeBeliefV1, TypeBelief:
		return TypeBelief, nil
	case typeObsV1, TypeEnv:
		return TypeEnv, nil
	case TypeAction:
		return TypeAction, nil
	default:
		return "", reject.New("EN-TYPE", "type not in CORE registry")
	}
}

type Payload struct {
	Temporal    temporal.Stamp
	Truth       truth.Bearer
	Content     map[string]any
	FoldRef     string
	Attestation *Attestation
}

func (p Payload) Map() map[string]any {
	content := p.Content
	if content == nil {
		content = map[string]any{}
	}
	m := map[string]any{
		"content":  content,
		"temporal": p.Temporal.Map(),
		"truth":    p.Truth.Map(),
	}
	if p.FoldRef != "" {
		m["fold_ref"] = p.FoldRef
	}
	if p.Attestation != nil {
		m["attestation"] = p.Attestation.Map()
	}
	return m
}

type Event struct {
	SpecVersion     string
	Type            string
	Source          string
	ID              string
	Time            string
	DataContentType string
	Subject         string
	Data            Payload
}

func (e Event) Map() map[string]any {
	return map[string]any{
		"data":            e.Data.Map(),
		"datacontenttype": e.DataContentType,
		"id":              e.ID,
		"source":          e.Source,
		"specversion":     e.SpecVersion,
		"subject":         e.Subject,
		"time":            e.Time,
		"type":            e.Type,
	}
}

func EventID(payload map[string]any) (string, error) {
	b, err := jcs.CanonicalValue(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func Pack(typ, source, subject string, data Payload) (Event, error) {
	n, err := NormalizeType(typ)
	if err != nil {
		return Event{}, err
	}
	if err := ValidateAttestation(n, data.Attestation); err != nil {
		return Event{}, err
	}
	id, err := EventID(data.Map())
	if err != nil {
		return Event{}, err
	}
	tPresent := temporal.FormatInstant(data.Temporal.TPresent)
	return Event{
		SpecVersion:     SpecVersion,
		Type:            n,
		Source:          source,
		ID:              id,
		Time:            tPresent,
		DataContentType: DataContentType,
		Subject:         subject,
		Data:            data,
	}, nil
}

func PackV2(typ, source, subject string, data Payload) (Event, error) {
	if data.Attestation == nil {
		return Event{}, reject.New("EN-ATT", "lock v2 payload requires attestation")
	}
	return Pack(typ, source, subject, data)
}

func Parse(raw []byte) (Event, error) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return Event{}, err
	}
	m, ok := v.(map[string]any)
	if !ok {
		return Event{}, reject.New("EN-OBJ", "CloudEvents object required")
	}
	return ParseMap(m)
}

func ParseMap(m map[string]any) (Event, error) {
	typ, _ := m["type"].(string)
	n, err := NormalizeType(typ)
	if err != nil {
		return Event{}, err
	}
	dataMap, _ := m["data"].(map[string]any)
	if dataMap == nil {
		return Event{}, reject.New("EN-DATA", "data required")
	}
	var att *Attestation
	if rawAtt, ok := dataMap["attestation"].(map[string]any); ok {
		att = ParseAttestation(rawAtt)
	}
	if err := ValidateAttestation(n, att); err != nil {
		return Event{}, err
	}
	stampMap, _ := dataMap["temporal"].(map[string]any)
	st, err := temporal.StampFromStrings(
		str(stampMap["t_event"]),
		str(stampMap["t_observe"]),
		str(stampMap["t_admit"]),
		str(stampMap["t_present"]),
	)
	if err != nil {
		return Event{}, err
	}
	truthMap, _ := dataMap["truth"].(map[string]any)
	bearer, err := truth.ParseBearer(truthMap)
	if err != nil {
		return Event{}, err
	}
	content, _ := dataMap["content"].(map[string]any)
	p := Payload{
		Temporal:    st,
		Truth:       bearer,
		Content:     content,
		FoldRef:     str(dataMap["fold_ref"]),
		Attestation: att,
	}
	want, err := EventID(p.Map())
	if err != nil {
		return Event{}, err
	}
	got, _ := m["id"].(string)
	if got != want {
		return Event{}, reject.New("CF-008", "envelope id ≠ SHA-256(JCS(payload)): "+got)
	}
	tPresent := temporal.FormatInstant(st.TPresent)
	if str(m["time"]) != tPresent {
		return Event{}, reject.New("EN-TIME", "CloudEvents time MUST equal t_present")
	}
	if str(m["specversion"]) != SpecVersion {
		return Event{}, reject.New("EN-SPEC", "specversion MUST be 1.0")
	}
	dct := str(m["datacontenttype"])
	if dct != "" && dct != DataContentType {
		return Event{}, reject.New("EN-DCT", "datacontenttype MUST be application/json")
	}
	if dct == "" {
		dct = DataContentType
	}
	return Event{
		SpecVersion:     SpecVersion,
		Type:            n,
		Source:          str(m["source"]),
		ID:              got,
		Time:            tPresent,
		DataContentType: dct,
		Subject:         str(m["subject"]),
		Data:            p,
	}, nil
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func LockV2Payload() (Payload, error) {
	st, err := temporal.StampFromStrings(
		"2026-08-27T08:00:00Z",
		"2026-08-27T08:00:01Z",
		"2026-08-27T08:00:02Z",
		"2026-08-27T11:00:00Z",
	)
	if err != nil {
		return Payload{}, err
	}
	return Payload{
		Temporal: st,
		Truth:    truth.Bearer{Class: truth.FACT, Provenance: truth.ObservedSigned, Ref: "ver-1"},
		Content:  map[string]any{"depart": "09:30"},
		FoldRef:  "f80b9b513fc928d11e8aceb66a29d7cfb7540a0bb8451c9017630b105602e9f5",
		Attestation: &Attestation{
			AttesterID:  "urn:a3:party:attester",
			RequesterID: "urn:a3:party:requester",
		},
	}, nil
}

func LockV2Event() (Event, error) {
	p, err := LockV2Payload()
	if err != nil {
		return Event{}, err
	}
	return PackV2(TypeBelief, "urn:a3:source:train", "trip.milano", p)
}
