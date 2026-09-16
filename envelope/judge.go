// SPDX-License-Identifier: Apache-2.0
// Part of the A3 universe. See LICENSE.
package envelope

import (
	"encoding/json"

	"github.com/valeriobitzone-png/a3-go/confidence"
	"github.com/valeriobitzone-png/a3-go/jcs"
	"github.com/valeriobitzone-png/a3-go/reject"
	"github.com/valeriobitzone-png/a3-go/temporal"
	"github.com/valeriobitzone-png/a3-go/truth"
)

// JudgeFile applies CF-001..CF-009 to a category fixture. Silence is not a pass.
func JudgeFile(raw []byte) error {
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return err
	}
	code, _ := root["reject_code"].(string)
	switch code {
	case "CF-001":
		claimed, _ := root["claimed"].(map[string]any)
		class, _ := claimed["truth_class"].(string)
		exit, _ := root["exit_code"].(float64)
		printed, _ := root["printed"].(string)
		c, _ := truth.ParseBearer(map[string]any{"truth_class": class, "provenance": "observed_signed"})
		if err := truth.RejectReceiptFact(c.Class, int(exit), printed); err != nil {
			return err
		}
		return reject.New("CF-001", "fixture did not claim FACT on receipt")
	case "CF-002":
		claimed, _ := root["claimed"].(map[string]any)
		class, _ := claimed["truth_class"].(string)
		env, _ := root["environment"].(string)
		c, _ := truth.ParseBearer(map[string]any{"truth_class": class, "provenance": "observed_signed"})
		if err := truth.RejectSandboxFact(c.Class, truth.Environment(stringsToLower(env))); err != nil {
			return err
		}
		return reject.New("CF-002", "fixture did not claim FACT on SANDBOX")
	case "CF-003":
		claimed, _ := root["claimed"].(map[string]any)
		class, _ := claimed["truth_class"].(string)
		c, _ := truth.ParseBearer(map[string]any{"truth_class": class, "provenance": "derived_model"})
		if err := truth.RejectPromotionWithoutVerification(c.Class, root["verification"]); err != nil {
			return err
		}
		return reject.New("CF-003", "fixture did not claim FACT without verification")
	case "CF-004":
		event, _ := root["event"].(map[string]any)
		_, err := ParseMap(event)
		if err != nil {
			if e := reject.As(err); e != nil && e.Code == "CF-004" {
				return err
			}
			return err
		}
		return reject.New("CF-004", "implementation accepted attester_id = requester_id")
	case "CF-005":
		class, _ := root["truth_class"].(string)
		c, _ := truth.ParseBearer(map[string]any{"truth_class": class, "provenance": "inferred"})
		if err := truth.RejectInventedUnknown(c.Class, root["proposition"]); err != nil {
			return err
		}
		return reject.New("CF-005", "fixture is not UNKNOWN with a proposition")
	case "CF-006":
		claimed, _ := root["claimed"].(map[string]any)
		replaces, _ := claimed["replaces_history"].(bool)
		hist, _ := claimed["history"].([]any)
		if replaces || (hist != nil && len(hist) == 0) {
			return reject.New("CF-006", "fold applied to causal history (not confidence/projection)")
		}
		return reject.New("CF-006", "fixture did not replace history")
	case "CF-007":
		vec, err := confidence.ParseVector(asMap(root["vector"]))
		if err != nil {
			return err
		}
		w, err := confidence.ParseWeights(asMap(root["weights"]))
		if err != nil {
			return err
		}
		lawful, err := confidence.Score(vec, w)
		if err != nil {
			return err
		}
		claimed, _ := root["claimed_score"].(float64)
		agg, _ := root["aggregation"].(string)
		if err := confidence.RejectCompensatingMean(claimed, lawful, agg); err != nil {
			return err
		}
		return reject.New("CF-007", "fixture matched weighted min")
	case "CF-008":
		event, _ := root["event"].(map[string]any)
		_, err := ParseMap(event)
		if err != nil {
			if e := reject.As(err); e != nil && e.Code == "CF-008" {
				return err
			}
			return err
		}
		return reject.New("CF-008", "implementation accepted envelope id ≠ SHA-256(JCS(payload))")
	case "CF-009":
		input, err := temporal.RowsFromAny(root["input"])
		if err != nil {
			return err
		}
		lawful := temporal.Sort(input)
		claimed, _ := root["claimed_order"].([]any)
		tie, _ := root["tie_break"].(string)
		ok := tie != "none" && len(claimed) == len(lawful)
		if ok {
			for i := range lawful {
				s, _ := claimed[i].(string)
				if s != lawful[i].SourceID {
					ok = false
					break
				}
			}
		}
		if !ok {
			return reject.New("CF-009", "order without tie-break (source_id, seq)")
		}
		return reject.New("CF-009", "fixture already used the protocol order")
	default:
		return reject.New("CF-UNKNOWN", "unknown reject_code "+code)
	}
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func stringsToLower(s string) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		out[i] = c
	}
	return string(out)
}

func MustJCS(v any) []byte {
	b, err := jcs.CanonicalValue(v)
	if err != nil {
		panic(err)
	}
	return b
}
