package main

import (
	"encoding/json"
	"os"
	"os/exec"
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

// `power-analysis` makes two claims that nothing else here would notice going
// wrong: that its 160 Cases are exactly what its committed generator writes,
// and that its sweep is monotone. Both are the kind of failure that stays
// green — a hand-edited Case file still runs, and a sweep that stopped
// separating still prints a table.

// TestPowerAnalysisCasesMatchTheirGenerator is what pays for committing both
// the program and its output. The generator is the scenario's provenance
// claim; a cases.jsonl that has drifted from it turns that claim into a
// comment.
func TestPowerAnalysisCasesMatchTheirGenerator(t *testing.T) {
	t.Parallel()
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("no python3 on PATH: install one to check the generator against its committed output")
	}
	root := repoRoot(t)
	gen := filepath.Join(root, "scenarios", "power-analysis", "evals", "generate.py")
	out, err := exec.Command(python, gen).Output() //nolint:gosec // a repo path
	if err != nil {
		t.Fatalf("run %s: %v", gen, err)
	}
	committed, err := os.ReadFile(filepath.Join(root, "scenarios", "power-analysis", "evals", "cases.jsonl")) //nolint:gosec // a repo path
	if err != nil {
		t.Fatalf("read cases: %v", err)
	}
	if string(out) != string(committed) {
		t.Fatalf("scenarios/power-analysis/evals/cases.jsonl is not what generate.py writes.\n"+
			"Regenerate it rather than editing it:\n"+
			"  python3 %s > scenarios/power-analysis/evals/cases.jsonl\n"+
			"generated %d bytes, committed %d bytes", gen, len(out), len(committed))
	}
}

// TestPowerAnalysisSweepIsMonotone asserts the one thing the scenario exists to
// show: more Cases separate smaller effects, and the checks clear rather than
// re-flag. If a future release made the sweep flat, every stage would still run
// clean and the scenario would still be green while demonstrating nothing.
func TestPowerAnalysisSweepIsMonotone(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	type behavior struct {
		Tag             string  `json:"tag"`
		DevCases        int     `json:"dev_cases"`
		SeparableEffect float64 `json:"separable_effect"`
	}
	type projection struct {
		Cases struct {
			Total int `json:"total"`
		} `json:"cases"`
		Behaviors     []behavior `json:"behaviors"`
		ChecksFlagged int        `json:"checks_flagged"`
	}
	read := func(stage string) projection {
		b, err := os.ReadFile(filepath.Join(root, "scenarios", "power-analysis", "expected", stage+".json")) //nolint:gosec // a repo path
		if err != nil {
			t.Fatalf("read %s: %v", stage, err)
		}
		var p projection
		if err := json.Unmarshal(b, &p); err != nil {
			t.Fatalf("%s: %v", stage, err)
		}
		if len(p.Behaviors) == 0 {
			t.Fatalf("%s projects no behaviors, so it asserts nothing about power", stage)
		}
		return p
	}
	worst := func(p projection) float64 {
		out := p.Behaviors[0].SeparableEffect
		for _, b := range p.Behaviors[1:] {
			if b.SeparableEffect > out {
				out = b.SeparableEffect
			}
		}
		return out
	}

	stages := []string{"inspect-12", "inspect-40", "inspect-160"}
	prev := read(stages[0])
	for _, stage := range stages[1:] {
		cur := read(stage)
		if cur.Cases.Total <= prev.Cases.Total {
			t.Fatalf("%s has %d Cases and the stage before it has %d: the sweep must grow",
				stage, cur.Cases.Total, prev.Cases.Total)
		}
		if worst(cur) >= worst(prev) {
			t.Fatalf("%s separates %.2f at worst and the stage before it separates %.2f: more Cases must separate smaller effects, or the scenario shows nothing",
				stage, worst(cur), worst(prev))
		}
		if cur.ChecksFlagged > prev.ChecksFlagged {
			t.Fatalf("%s flags %d checks and the stage before it flags %d: a larger eval set must not flag more",
				stage, cur.ChecksFlagged, prev.ChecksFlagged)
		}
		prev = cur
	}
	if prev.ChecksFlagged != 0 {
		t.Fatalf("the largest eval set still flags %d check(s); the sweep's endpoint is meant to be a clean one", prev.ChecksFlagged)
	}
}

// TestMinedCasesMatchTheTranscripts asserts that `transcript-mining`'s
// committed evals/cases.jsonl is byte-identical to what `kno mine` writes over
// its committed transcripts.
//
// That scenario is the one place here where evals/cases.jsonl is an OUTPUT
// rather than an input: run.sh's later stages read the file the run just
// produced, so the committed copy is never consulted by the run and nothing
// else would notice it going stale. This is what makes it an expectation
// rather than a souvenir — if a release changes what mining writes, or if
// somebody edits the Cases instead of the transcripts, the failure has a name.
func TestMinedCasesMatchTheTranscripts(t *testing.T) {
	bin := knoBinary(t)
	root := repoRoot(t)
	scenario := filepath.Join(root, "scenarios", "transcript-mining")

	// Mined in a fresh directory laid out the way run.sh lays one out, because
	// `source_ref` records the path mining was given: running this against the
	// scenario directory itself would record a different one and the
	// comparison would fail for a reason that is not a finding.
	dir := t.TempDir()
	if err := copyTree(filepath.Join(scenario, "transcripts"), filepath.Join(dir, "transcripts")); err != nil {
		t.Fatalf("copy transcripts: %v", err)
	}
	cmd := exec.Command(bin, "mine",
		"--logs", "transcripts",
		"--out", "mined.jsonl",
		"--mode", "resolution",
		"--format", "auto",
	)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("kno mine: %v: %s", err, out)
	}
	mined, err := os.ReadFile(filepath.Join(dir, "mined.jsonl"))
	if err != nil {
		t.Fatalf("read mined output: %v", err)
	}
	committed, err := os.ReadFile(filepath.Join(scenario, "evals", "cases.jsonl")) //nolint:gosec // a repo path
	if err != nil {
		t.Fatalf("read committed Cases: %v", err)
	}
	if string(mined) != string(committed) {
		t.Fatalf("scenarios/transcript-mining/evals/cases.jsonl is not what `kno mine` writes over the committed transcripts.\n"+
			"If the transcripts changed, regenerate it; if the binary changed, decide whether the scenario's claim survives.\n"+
			"mined %d bytes, committed %d bytes", len(mined), len(committed))
	}
}
