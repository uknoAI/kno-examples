package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// `underpowered-eval` is `support-refunds` with three Cases removed and nothing
// else changed. That is what makes the pair a controlled comparison: one ends
// in `no-effect`, the other in `underpowered`, and the only variable is the
// Case count.
//
// The Cases are therefore duplicated across the two scenario directories, which
// they must be — `internal/scenario` requires every scenario to carry its own
// evals/cases.jsonl, and run.sh copies it into a working directory. This test
// is what pays for that duplication, the same posture this repository takes
// toward the demo fixtures `kno demo` embeds.

func readLines(t *testing.T, parts ...string) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(parts...)) //nolint:gosec // a repo path
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return strings.Split(strings.TrimRight(string(b), "\n"), "\n")
}

func TestUnderpoweredCasesAreAPrefixOfSupportRefunds(t *testing.T) {
	root := repoRoot(t)
	full := readLines(t, root, "scenarios", "support-refunds", "evals", "cases.jsonl")
	short := readLines(t, root, "scenarios", "underpowered-eval", "evals", "cases.jsonl")

	if len(short) >= len(full) {
		t.Fatalf("underpowered-eval has %d Cases and support-refunds has %d: the shorter scenario must be shorter",
			len(short), len(full))
	}
	for i, line := range short {
		if line != full[i] {
			t.Fatalf("Case %d differs, so the two scenarios are no longer a controlled comparison:\n  support-refunds:   %s\n  underpowered-eval: %s",
				i+1, full[i], line)
		}
	}
}

func TestUnderpoweredPoolIsIdenticalToSupportRefunds(t *testing.T) {
	root := repoRoot(t)
	a := readLines(t, root, "scenarios", "support-refunds", "pool", "pool.jsonl")
	b := readLines(t, root, "scenarios", "underpowered-eval", "pool", "pool.jsonl")

	if len(a) != len(b) {
		t.Fatalf("pools differ in size (%d vs %d): the Assets must be the same for the comparison to hold", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("Pool asset %d differs, so the two scenarios no longer offer the same Assets:\n  %s\n  %s", i+1, a[i], b[i])
		}
	}
}

// TestTheTwoScenariosReachDifferentVerdicts is the point of the pair. If both
// ever report the same reason, one of them has stopped earning its place.
func TestTheTwoScenariosReachDifferentVerdicts(t *testing.T) {
	root := repoRoot(t)
	read := func(scenario string) string {
		b, err := os.ReadFile(filepath.Join(root, "scenarios", scenario, "expected", "select.json")) //nolint:gosec // a repo path
		if err != nil {
			t.Fatalf("read %s expectation: %v", scenario, err)
		}
		return string(b)
	}
	sr, ue := read("support-refunds"), read("underpowered-eval")

	if !strings.Contains(sr, "no-effect") {
		t.Errorf("support-refunds no longer expects no-effect; the pair's contrast is gone:\n%s", sr)
	}
	if !strings.Contains(ue, "underpowered") {
		t.Errorf("underpowered-eval no longer expects underpowered, which is the only reason it exists:\n%s", ue)
	}
	if strings.Contains(ue, "no-effect") {
		t.Errorf("underpowered-eval expects no-effect somewhere: it has crossed the boundary and needs fewer Cases:\n%s", ue)
	}
}
