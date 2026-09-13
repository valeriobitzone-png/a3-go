package conformance_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/valeriobitzone-png/a3-go/confidence"
	"github.com/valeriobitzone-png/a3-go/envelope"
	"github.com/valeriobitzone-png/a3-go/jcs"
	"github.com/valeriobitzone-png/a3-go/reject"
	"github.com/valeriobitzone-png/a3-go/temporal"
	"github.com/valeriobitzone-png/a3-go/truth"
)

const eventIDV2 = "1fec213fbaf6d420cf9ff95c51c022c4cdfb1f43fabcf82647e03c03f92f2b7b"
const eventIDV1 = "477e868489f5c48d138e4c084e9bf13a40ed66390b964365869f7578dfa2e75a"

func root(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
}

func read(t *testing.T, parts ...string) []byte {
	t.Helper()
	p := filepath.Join(append([]string{root(t)}, parts...)...)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func sha(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func mustJCS(t *testing.T, v any) []byte {
	t.Helper()
	b, err := jcs.CanonicalValue(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func loadJSON(t *testing.T, parts ...string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal(read(t, parts...), &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestGO001_JCSAppendixA(t *testing.T) {
	got, err := jcs.Canonicalize(read(t, "conformance", "vectors", "v2", "rfc8785-input.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := read(t, "conformance", "vectors", "v2", "envelope-rfc8785.json")
	if !bytes.Equal(got, want) {
		t.Fatalf("GO-001 JCS mismatch\n got %s\nwant %s", got, want)
	}
	t.Logf("PASS GO-001 RFC8785 bytes=%d", len(got))
}

func TestGO002_EnvelopeLockV2(t *testing.T) {
	wantPayload := read(t, "conformance", "vectors", "v2", "envelope-payload.json")
	wantEvent := read(t, "conformance", "vectors", "v2", "envelope-event.json")
	wantID := bytes.TrimSpace(read(t, "conformance", "vectors", "v2", "event_id_expected.txt"))
	if string(wantID) != eventIDV2 {
		t.Fatalf("event_id_expected %s", wantID)
	}
	p, err := envelope.LockV2Payload()
	if err != nil {
		t.Fatal(err)
	}
	gotPayload := mustJCS(t, p.Map())
	if !bytes.Equal(gotPayload, wantPayload) {
		t.Fatalf("GO-002 payload\n got %s\nwant %s", gotPayload, wantPayload)
	}
	id, err := envelope.EventID(p.Map())
	if err != nil {
		t.Fatal(err)
	}
	if id != eventIDV2 {
		t.Fatalf("GO-002 event.id %s", id)
	}
	if sha(wantPayload) != eventIDV2 {
		t.Fatal("payload file is not SHA-256 of itself as lock")
	}
	ev, err := envelope.LockV2Event()
	if err != nil {
		t.Fatal(err)
	}
	if ev.Type != envelope.TypeBelief {
		t.Fatalf("type %s", ev.Type)
	}
	if ev.ID != eventIDV2 {
		t.Fatalf("id %s", ev.ID)
	}
	gotEvent := mustJCS(t, ev.Map())
	if !bytes.Equal(gotEvent, wantEvent) {
		t.Fatalf("GO-002 event\n got %s\nwant %s", gotEvent, wantEvent)
	}
	parsed, err := envelope.Parse(wantEvent)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Type != envelope.TypeBelief || parsed.ID != eventIDV2 {
		t.Fatalf("parse %s %s", parsed.Type, parsed.ID)
	}
	err = envelope.JudgeFile(read(t, "conformance", "fixtures", "envelope-id-violation.json"))
	if e := reject.As(err); e == nil || e.Code != "CF-008" {
		t.Fatalf("GO-002 CF-008 %v", err)
	}
	t.Logf("PASS GO-002 event.id=%s", eventIDV2)
}

func TestGO003_TemporalLock(t *testing.T) {
	raw := loadJSON(t, "conformance", "vectors", "v2", "tm-order.json")
	rows, err := temporal.RowsFromAny(raw)
	if err != nil {
		t.Fatal(err)
	}
	ordered := temporal.Sort(rows)
	gotOrder := mustJCS(t, temporal.MapsOf(ordered))
	wantOrder := read(t, "conformance", "vectors", "v2", "tm-order.json")
	if !bytes.Equal(gotOrder, wantOrder) {
		t.Fatalf("GO-003 order\n got %s\nwant %s", gotOrder, wantOrder)
	}
	for _, perm := range permute(rows) {
		if !bytes.Equal(mustJCS(t, temporal.MapsOf(temporal.Sort(perm))), wantOrder) {
			t.Fatal("GO-003 shuffle")
		}
	}
	d := temporal.Dedup(rows)
	gotDedup := mustJCS(t, d.Map())
	wantDedup := read(t, "conformance", "vectors", "v2", "tm-dedup.json")
	if !bytes.Equal(gotDedup, wantDedup) {
		t.Fatalf("GO-003 dedup\n got %s\nwant %s", gotDedup, wantDedup)
	}
	fold := temporal.Project(rows)
	gotFold := mustJCS(t, fold.Map())
	wantFold := read(t, "conformance", "vectors", "v2", "tm-fold.json")
	if !bytes.Equal(gotFold, wantFold) {
		t.Fatalf("GO-003 fold\n got %s\nwant %s", gotFold, wantFold)
	}
	if len(fold.History) != len(ordered) {
		t.Fatal("fold dropped history")
	}
	err = envelope.JudgeFile(read(t, "conformance", "fixtures", "history-fold-violation.json"))
	if e := reject.As(err); e == nil || e.Code != "CF-006" {
		t.Fatalf("GO-003 CF-006 %v", err)
	}
	err = envelope.JudgeFile(read(t, "conformance", "fixtures", "order-tiebreak-violation.json"))
	if e := reject.As(err); e == nil || e.Code != "CF-009" {
		t.Fatalf("GO-003 CF-009 %v", err)
	}
	t.Logf("PASS GO-003 permutations=%d", len(permute(rows)))
}

func TestGO004_TruthLock(t *testing.T) {
	got := mustJCS(t, truth.LockVectors())
	want := read(t, "conformance", "vectors", "v2", "truth-vectors.json")
	if !bytes.Equal(got, want) {
		t.Fatalf("GO-004 truth\n got %s\nwant %s", got, want)
	}
	fixtures := []struct {
		file, code string
	}{
		{"receipt-fact-violation.json", "CF-001"},
		{"sandbox-fact-violation.json", "CF-002"},
		{"hypothesis-promotion-violation.json", "CF-003"},
		{"unknown-invention-violation.json", "CF-005"},
	}
	for _, f := range fixtures {
		err := envelope.JudgeFile(read(t, "conformance", "fixtures", f.file))
		e := reject.As(err)
		if e == nil || e.Code != f.code {
			t.Fatalf("GO-004 %s got %v", f.file, err)
		}
	}
	t.Log("PASS GO-004 truth-vectors + CF-001/002/003/005")
}

func TestGO005_ConfidenceLock(t *testing.T) {
	raw := loadJSON(t, "conformance", "vectors", "v2", "confidence-vectors.json").(map[string]any)
	fixture := raw["fixture"].(map[string]any)
	vec, err := confidence.ParseVector(fixture["vector"].(map[string]any))
	if err != nil {
		t.Fatal(err)
	}
	w, err := confidence.ParseWeights(fixture["weights"].(map[string]any))
	if err != nil {
		t.Fatal(err)
	}
	score, err := confidence.Score(vec, w)
	if err != nil {
		t.Fatal(err)
	}
	wantScore := fixture["score"].(float64)
	if score != wantScore {
		t.Fatalf("GO-005 fixture score got=%v want=%v", score, wantScore)
	}
	observe, _ := temporal.ParseInstant("2026-08-27T08:00:01Z")
	present, _ := temporal.ParseInstant("2026-08-27T11:00:00Z")
	rec := confidence.RecencyBetween(observe, present, confidence.HalfLifeSeconds)
	wantRec := raw["recency"].(map[string]any)["fixture_10799s"].(float64)
	if rec != wantRec {
		if math.Nextafter(rec, wantRec) != wantRec && math.Nextafter(wantRec, rec) != rec {
			t.Fatalf("GO-005 recency off by >1 ULP got=%v want=%v", rec, wantRec)
		}
		t.Logf("DIVERGENCE recency: spec 2^(-Δt/21600) via Go math.Pow=%v; lock/Kotlin stored=%v (1 ULP). Not rounded to hide it.", rec, wantRec)
	}
	if rec != vec.Recency {
		t.Logf("DIVERGENCE lock vector.recency=%v Go Pow=%v", vec.Recency, rec)
	}
	zero, err := confidence.ParseVector(raw["zero_not_compensated"].(map[string]any)["vector"].(map[string]any))
	if err != nil {
		t.Fatal(err)
	}
	zw, err := confidence.ParseWeights(raw["zero_not_compensated"].(map[string]any)["weights"].(map[string]any))
	if err != nil {
		t.Fatal(err)
	}
	zs, err := confidence.Score(zero, zw)
	if err != nil || zs != 0 {
		t.Fatalf("GO-005 zero compensated %v %v", zs, err)
	}
	err = envelope.JudgeFile(read(t, "conformance", "fixtures", "compensating-mean-violation.json"))
	if e := reject.As(err); e == nil || e.Code != "CF-007" {
		t.Fatalf("GO-005 CF-007 %v", err)
	}
	t.Logf("PASS GO-005 score=%v recency=%v", score, rec)
}

func TestGO006_TripleCollision(t *testing.T) {
	ev, err := envelope.LockV2Event()
	if err != nil {
		t.Fatal(err)
	}
	got := mustJCS(t, ev.Map())
	kotlin := read(t, "conformance", "vectors", "v2", "envelope-event.json")
	if !bytes.Equal(got, kotlin) {
		t.Fatal("GO-006 Go event bytes ≠ lock v2 (Kotlin/TS canonical)")
	}
	p, err := envelope.LockV2Payload()
	if err != nil {
		t.Fatal(err)
	}
	payload := mustJCS(t, p.Map())
	if !bytes.Equal(payload, read(t, "conformance", "vectors", "v2", "envelope-payload.json")) {
		t.Fatal("GO-006 payload bytes ≠ lock v2")
	}
	t.Log("PASS GO-006 triple collision on lock v2 bytes")
}

func TestGO007_ParseV1NormalizesType(t *testing.T) {
	raw := read(t, "conformance", "vectors", "v1", "envelope-event.json")
	var peek map[string]any
	if err := json.Unmarshal(raw, &peek); err != nil {
		t.Fatal(err)
	}
	if peek["type"] != "a3.belief.admitted" {
		t.Fatalf("v1 disk type %v", peek["type"])
	}
	ev, err := envelope.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Type != envelope.TypeBelief {
		t.Fatalf("GO-007 normalized type %s", ev.Type)
	}
	if ev.Data.Attestation != nil {
		t.Fatal("GO-007 v1 must omit attestation")
	}
	if ev.ID != eventIDV1 {
		t.Fatalf("GO-007 v1 id %s", ev.ID)
	}
	t.Log("PASS GO-007 a3.belief.admitted → io.a3ep.belief.admitted")
}

func TestGO008_AttesterNotRequester(t *testing.T) {
	err := envelope.JudgeFile(read(t, "conformance", "fixtures", "attester-requester-violation.json"))
	e := reject.As(err)
	if e == nil || e.Code != "CF-004" {
		t.Fatalf("GO-008 %v", err)
	}
	p, err := envelope.LockV2Payload()
	if err != nil {
		t.Fatal(err)
	}
	p.Attestation = &envelope.Attestation{AttesterID: "urn:a3:party:alice", RequesterID: "urn:a3:party:alice"}
	_, err = envelope.PackV2(envelope.TypeAction, "urn:a3:source:train", "trip.milano", p)
	if e := reject.As(err); e == nil || e.Code != "CF-004" {
		t.Fatalf("GO-008 pack %v", err)
	}
	t.Log("PASS GO-008 CF-004")
}

func TestGO009_Independence(t *testing.T) {
	here := root(t)
	for _, name := range []string{"a3", "a3-ts"} {
		dir := filepath.Clean(filepath.Join(here, "..", name))
		if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
			t.Fatalf("GO-009 missing sibling %s", dir)
		}
		out, err := exec.Command("git", "-C", dir, "diff", "--stat").CombinedOutput()
		if err != nil {
			t.Fatal(err, string(out))
		}
		if len(bytes.TrimSpace(out)) != 0 {
			t.Fatalf("GO-009 %s dirty:\n%s", name, out)
		}
	}
	t.Log("PASS GO-009 a3 and a3-ts diffs empty")
}

func TestVectorSHA256Manifest(t *testing.T) {
	var manifest map[string]string
	if err := json.Unmarshal(read(t, "conformance", "vectors", "v2", "vector-sha256.json"), &manifest); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root(t), "conformance", "vectors", "v2")
	for name, want := range manifest {
		got := sha(read(t, "conformance", "vectors", "v2", name))
		if got != want {
			t.Fatalf("%s sha %s != %s", name, got, want)
		}
		_ = dir
	}
}

func permute(rows []temporal.Observation) [][]temporal.Observation {
	if len(rows) <= 1 {
		return [][]temporal.Observation{append([]temporal.Observation(nil), rows...)}
	}
	var out [][]temporal.Observation
	for i := range rows {
		rest := append(append([]temporal.Observation{}, rows[:i]...), rows[i+1:]...)
		for _, p := range permute(rest) {
			out = append(out, append([]temporal.Observation{rows[i]}, p...))
		}
	}
	return out
}
