// SPDX-License-Identifier: Apache-2.0
// Part of the A3 universe. See LICENSE.
// Package temporal implements SPEC_A3-EP §5: four instants, order, history, fold.
package temporal

import (
	"sort"
	"time"

	"github.com/valeriobitzone-png/a3-go/reject"
)

const layout = time.RFC3339

// Stamp is t_event, t_observe, t_admit, t_present. No clock is read.
type Stamp struct {
	TEvent   time.Time
	TObserve time.Time
	TAdmit   time.Time
	TPresent time.Time
}

func ParseInstant(s string) (time.Time, error) {
	return time.Parse(layout, s)
}

func FormatInstant(t time.Time) string {
	return t.UTC().Format(layout)
}

// NewStamp rejects inverted observe/event or admit/observe. t_present is explicit.
func NewStamp(tEvent, tObserve, tAdmit, tPresent time.Time) (Stamp, error) {
	if tObserve.Before(tEvent) {
		return Stamp{}, reject.New("TM-001", "t_observe < t_event")
	}
	if tAdmit.Before(tObserve) {
		return Stamp{}, reject.New("TM-002", "t_admit < t_observe")
	}
	return Stamp{TEvent: tEvent, TObserve: tObserve, TAdmit: tAdmit, TPresent: tPresent}, nil
}

func StampFromStrings(tEvent, tObserve, tAdmit, tPresent string) (Stamp, error) {
	a, err := ParseInstant(tEvent)
	if err != nil {
		return Stamp{}, err
	}
	b, err := ParseInstant(tObserve)
	if err != nil {
		return Stamp{}, err
	}
	c, err := ParseInstant(tAdmit)
	if err != nil {
		return Stamp{}, err
	}
	d, err := ParseInstant(tPresent)
	if err != nil {
		return Stamp{}, err
	}
	return NewStamp(a, b, c, d)
}

func (s Stamp) Map() map[string]any {
	return map[string]any{
		"t_admit":   FormatInstant(s.TAdmit),
		"t_event":   FormatInstant(s.TEvent),
		"t_observe": FormatInstant(s.TObserve),
		"t_present": FormatInstant(s.TPresent),
	}
}

// Observation is one append-only log row.
type Observation struct {
	SourceID string
	Subject  string
	Key      string
	Seq      int64
	Stamp    Stamp
	Value    string
}

func (o Observation) Map() map[string]any {
	return map[string]any{
		"key":       o.Key,
		"seq":       o.Seq,
		"source_id": o.SourceID,
		"stamp":     o.Stamp.Map(),
		"subject":   o.Subject,
		"value":     o.Value,
	}
}

func (o Observation) currentMap() map[string]any {
	return map[string]any{
		"key":       o.Key,
		"seq":       o.Seq,
		"source_id": o.SourceID,
		"subject":   o.Subject,
		"value":     o.Value,
	}
}

func (o Observation) fieldMap() map[string]any {
	return map[string]any{
		"key":       o.Key,
		"seq":       o.Seq,
		"source_id": o.SourceID,
		"t_observe": FormatInstant(o.Stamp.TObserve),
		"value":     o.Value,
	}
}

func dedupKey(o Observation) string {
	return o.SourceID + "\x1f" + o.Subject + "\x1f" + o.Key
}

func less(a, b Observation) bool {
	if !a.Stamp.TObserve.Equal(b.Stamp.TObserve) {
		return a.Stamp.TObserve.Before(b.Stamp.TObserve)
	}
	if a.SourceID != b.SourceID {
		return a.SourceID < b.SourceID
	}
	if a.Seq != b.Seq {
		return a.Seq < b.Seq
	}
	if a.Subject != b.Subject {
		return a.Subject < b.Subject
	}
	return a.Key < b.Key
}

// Sort is the protocol order (t_observe, source_id, seq) with residual (subject, key).
func Sort(rows []Observation) []Observation {
	out := append([]Observation(nil), rows...)
	sort.SliceStable(out, func(i, j int) bool { return less(out[i], out[j]) })
	return out
}

// Dedup keeps the append-only log and last-wins current rows per (source_id, subject, key).
type DedupResult struct {
	Log     []Observation
	Current []Observation
}

func Dedup(rows []Observation) DedupResult {
	log := Sort(rows)
	last := map[string]Observation{}
	order := []string{}
	seen := map[string]bool{}
	for _, o := range log {
		k := dedupKey(o)
		last[k] = o
		if !seen[k] {
			seen[k] = true
			order = append(order, k)
		}
	}
	current := make([]Observation, 0, len(last))
	for _, o := range last {
		current = append(current, o)
	}
	sort.Slice(current, func(i, j int) bool {
		if current[i].SourceID != current[j].SourceID {
			return current[i].SourceID < current[j].SourceID
		}
		if current[i].Subject != current[j].Subject {
			return current[i].Subject < current[j].Subject
		}
		return current[i].Key < current[j].Key
	})
	_ = order
	return DedupResult{Log: log, Current: current}
}

func (d DedupResult) Map() map[string]any {
	cur := make([]any, len(d.Current))
	for i, o := range d.Current {
		cur[i] = o.currentMap()
	}
	log := make([]any, len(d.Log))
	for i, o := range d.Log {
		log[i] = o.Map()
	}
	return map[string]any{"current": cur, "log": log}
}

// Fold is a canonical projection. History is the ordered log; it is not replaced.
type Fold struct {
	Subject string
	Fields  map[string]Observation
	History []Observation
}

func Project(rows []Observation) Fold {
	d := Dedup(rows)
	fields := map[string]Observation{}
	history := []Observation{}
	subject := ""
	if len(d.Log) > 0 {
		subject = d.Log[0].Subject
	}
	for _, o := range d.Log {
		if subject == "" {
			subject = o.Subject
		}
		if o.Subject != subject {
			continue
		}
		history = append(history, o)
	}
	for _, o := range d.Current {
		if o.Subject != subject {
			continue
		}
		fields[o.Key] = o
	}
	return Fold{Subject: subject, Fields: fields, History: history}
}

func (f Fold) Map() map[string]any {
	fields := map[string]any{}
	for k, o := range f.Fields {
		fields[k] = o.fieldMap()
	}
	hist := make([]any, len(f.History))
	for i, o := range f.History {
		hist[i] = o.Map()
	}
	return map[string]any{
		"fields":  fields,
		"history": hist,
		"subject": f.Subject,
	}
}

func MapsOf(rows []Observation) []any {
	out := make([]any, len(rows))
	for i, o := range rows {
		out[i] = o.Map()
	}
	return out
}

func RowsFromAny(v any) ([]Observation, error) {
	list, ok := v.([]any)
	if !ok {
		return nil, reject.New("TM-003", "observation list required")
	}
	out := make([]Observation, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, reject.New("TM-003", "observation object required")
		}
		o, err := rowFromMap(m)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, nil
}

func rowFromMap(m map[string]any) (Observation, error) {
	stampMap, _ := m["stamp"].(map[string]any)
	st, err := StampFromStrings(
		str(stampMap["t_event"]),
		str(stampMap["t_observe"]),
		str(stampMap["t_admit"]),
		str(stampMap["t_present"]),
	)
	if err != nil {
		return Observation{}, err
	}
	seq, err := asInt64(m["seq"])
	if err != nil {
		return Observation{}, err
	}
	return Observation{
		SourceID: str(m["source_id"]),
		Subject:  str(m["subject"]),
		Key:      str(m["key"]),
		Seq:      seq,
		Stamp:    st,
		Value:    str(m["value"]),
	}, nil
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func asInt64(v any) (int64, error) {
	switch x := v.(type) {
	case float64:
		return int64(x), nil
	case int64:
		return x, nil
	case int:
		return int64(x), nil
	default:
		return 0, reject.New("TM-003", "seq must be an integer")
	}
}
