package scenario_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/uknoAI/kno-examples/internal/scenario"
)

// The projection's asymmetry is the whole design, so it gets the test.
//
// A full-document golden would churn on every run id, timestamp, duration and
// version, so it would be regenerated reflexively and rubber-stamped — a
// golden file that has stopped being a test. A projection over the fields a
// recipe's prose actually claims churns only when a claim becomes false.
func TestSubset(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		want     string
		got      string
		findings []string
	}{
		{
			name: "an additive field passes, because no recipe claimed anything about it",
			want: `{"score": 1, "scored": 8}`,
			got:  `{"score": 1, "scored": 8, "p95_latency_ms": 4}`,
		},
		{
			name:     "a removed field fails, because a recipe's claim just became false",
			want:     `{"score": 1, "scored": 8}`,
			got:      `{"score": 1}`,
			findings: []string{"baseline.scored: field is gone from the --json document (renamed or removed)"},
		},
		{
			name:     "a renamed field fails the same way",
			want:     `{"holdout_cases": 4}`,
			got:      `{"heldout_cases": 4}`,
			findings: []string{"baseline.holdout_cases: field is gone from the --json document (renamed or removed)"},
		},
		{
			name:     "a moved value fails",
			want:     `{"score": 1}`,
			got:      `{"score": 0.75}`,
			findings: []string{"baseline.score: expected 1, got 0.75"},
		},
		{
			name: "numbers compare as the literals the binary printed, not as float64",
			want: `{"low": -0.39597252156206514}`,
			got:  `{"low": -0.39597252156206514}`,
		},
		{
			name:     "a float that lost precision fails",
			want:     `{"low": -0.39597252156206514}`,
			got:      `{"low": -0.3959725215620651}`,
			findings: []string{"baseline.low: expected -0.39597252156206514, got -0.3959725215620651"},
		},
		{
			name: "nested objects and arrays are projected the same way",
			want: `{"valuations": [{"asset_id": "a"}, {"asset_id": "b"}]}`,
			got:  `{"valuations": [{"asset_id": "a", "n_dev": 6}, {"asset_id": "b", "n_dev": 6}]}`,
		},
		{
			name:     "an array that changed length fails: a missing Valuation is a finding",
			want:     `{"valuations": [{"asset_id": "a"}, {"asset_id": "b"}]}`,
			got:      `{"valuations": [{"asset_id": "a"}]}`,
			findings: []string{"baseline.valuations: expected 2 element(s), got 1"},
		},
		{
			name: "an explicit null is asserted rather than ignored: an empty Portfolio is a result",
			want: `{"selected": null}`,
			got:  `{"selected": null}`,
		},
		{
			name:     "a null that became populated fails",
			want:     `{"selected": null}`,
			got:      `{"selected": [{"asset_id": "a"}]}`,
			findings: []string{"baseline.selected: expected null, got [map[asset_id:a]]"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := scenario.Subset(decode(t, tc.want), decode(t, tc.got), "baseline")
			if len(got) != len(tc.findings) {
				t.Fatalf("findings:\n got %q\nwant %q", got, tc.findings)
			}
			for i := range got {
				if !strings.Contains(got[i], tc.findings[i]) {
					t.Errorf("finding %d:\n got %q\nwant it to contain %q", i, got[i], tc.findings[i])
				}
			}
		})
	}
}

func decode(t *testing.T, s string) map[string]any {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader([]byte(s)))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		t.Fatalf("decode %s: %v", s, err)
	}
	return m
}
