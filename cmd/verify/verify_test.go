package main

import (
	"os"
	"path/filepath"
	"testing"
)

// The runner is the only new code here, so it is the only thing that can
// silently lie. `testdata/` holds a deliberately broken corpus — one directory
// per way a recipe can rot — and these tests assert the runner refuses each
// one AND names the right file.
//
// A runner that passes everything is the failure mode. These tests are the
// defense.

func TestLintRefusesTheBrokenCorpus(t *testing.T) {
	t.Parallel()
	cases := []struct {
		dir        string
		wantInFile string
		wantSaying string
	}{
		{"no-tier", "no-tier.md", "no `verification:` field"},
		{"hand-edited", "hand-edited.md", "only an `executed` recipe may carry `last-verified`"},
		{"unnamed-credential", "unnamed-credential.md", "OPENAI_API_KEY is required"},
		// The tier says CI ran something; the page has nothing runnable on it.
		// A rendered `executed` badge over a page CI never executed is
		// indistinguishable from the real thing, which makes it the one
		// failure a tier system cannot survive.
		{"executed-runs-nothing", "executed-runs-nothing.md", "quotes no `kno-run` block"},
	}
	for _, tc := range cases {
		t.Run(tc.dir, func(t *testing.T) {
			t.Parallel()
			findings, err := cmdLint([]string{
				"--recipes", filepath.Join("testdata", tc.dir, "recipes"),
				"--scenarios", filepath.Join("testdata", tc.dir, "scenarios"),
			})
			if err != nil {
				t.Fatalf("lint returned an error rather than a finding: %v", err)
			}
			assertFinding(t, findings, tc.wantInFile, tc.wantSaying)
		})
	}
}

// TestLintNarrowsThePriorStageRuleToStagesThatReadAStore pins BOTH halves of
// the `independent_stages=` relaxation, because getting one right is the trap:
// deleting the rule would pass the first assertion and check nothing.
//
// The prior-stage rule exists so a green tick never sits over a command that
// will find an empty store. But `kno eval inspect` reads an eval FILE — no
// agent, no database, nothing an earlier stage wrote — so requiring a page
// that opens with one to declare `requires-stages:` would make it render a
// sentence ("run run.sh first, or they will find nothing") that is false.
//
// The corpus holds one recipe of each shape against one scenario: the stage
// run.sh declares independent must pass, and the stage it does not must still
// fail.
func TestLintNarrowsThePriorStageRuleToStagesThatReadAStore(t *testing.T) {
	t.Parallel()
	findings, err := cmdLint([]string{
		"--recipes", filepath.Join("testdata", "independent-stage", "recipes"),
		"--scenarios", filepath.Join("testdata", "independent-stage", "scenarios"),
		"--root", filepath.Join("testdata", "independent-stage"),
	})
	if err != nil {
		t.Fatalf("lint returned an error rather than a finding: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected exactly one finding — the store-reading stage, and NOT the declared-independent one; got:\n%v", findings)
	}
	assertFinding(t, findings, "dependent-stage.md", `quotes stage "reads-a-store"`)
}

// TestLintCatchesAQuotedBlockThatDriftedFromRunSh is the mechanism that stops
// `processed`/`issued` from happening again: not a review convention, a
// failing build.
func TestLintCatchesAQuotedBlockThatDriftedFromRunSh(t *testing.T) {
	t.Parallel()
	findings, err := cmdLint([]string{
		"--recipes", filepath.Join("testdata", "drift", "recipes"),
		"--scenarios", filepath.Join("testdata", "drift", "scenarios"),
	})
	if err != nil {
		t.Fatalf("lint: %v", err)
	}
	assertFinding(t, findings, "toy.md", "not byte-identical")
}

func TestLintPassesTheCorrectFixture(t *testing.T) {
	t.Parallel()
	findings, err := cmdLint([]string{
		"--recipes", filepath.Join("testdata", "good", "recipes"),
		"--scenarios", filepath.Join("testdata", "good", "scenarios"),
	})
	if err != nil {
		t.Fatalf("lint: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("a correct recipe must produce no findings; got %v", findings)
	}
}

// TestFlagCheckRefusesARenamedFlag needs the real binary, because the whole
// point is that the check runs against the binary users download rather than
// against a table we maintain. CI sets KNO_BIN; locally, install kno and set
// it, or the case is skipped rather than faked.
func TestFlagCheckRefusesARenamedFlag(t *testing.T) {
	t.Parallel()
	bin := knoBinary(t)
	findings, err := cmdFlags([]string{
		"--recipes", filepath.Join("testdata", "renamed-flag", "recipes"),
		"--kno", bin,
	})
	if err != nil {
		t.Fatalf("flags: %v", err)
	}
	assertFinding(t, findings, "renamed-flag.md", "--maximum-cost-usd")
}

// TestScenarioRefusesAnExpectationThatDisagrees runs the committed scenario
// against the real binary with one field of one expectation corrupted, and
// asserts the runner goes red. It is the other half of "the runner cannot
// silently lie": the lint catches shape, this catches value.
func TestScenarioRefusesAnExpectationThatDisagrees(t *testing.T) {
	bin := knoBinary(t)
	root := repoRoot(t)
	src := filepath.Join(root, "scenarios", "support-refunds")

	dir := t.TempDir()
	dst := filepath.Join(dir, "support-refunds")
	if err := copyTree(src, dst); err != nil {
		t.Fatalf("copy scenario: %v", err)
	}
	path := filepath.Join(dst, "expected", "baseline.json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read expectation: %v", err)
	}
	corrupted := replaceOnce(string(b), `"holdout_cases": 4`, `"holdout_cases": 6`)
	if corrupted == string(b) {
		t.Fatal("fixture drifted: the expectation no longer contains holdout_cases: 4")
	}
	if err := os.WriteFile(path, []byte(corrupted), 0o600); err != nil {
		t.Fatalf("write expectation: %v", err)
	}

	findings, err := cmdScenario([]string{"--scenario", dst, "--kno", bin})
	if err != nil {
		t.Fatalf("scenario: %v", err)
	}
	assertFinding(t, findings, "baseline.holdout_cases", "expected 6, got 4")
}

// TestScenarioIsDeterministic runs the committed scenario twice and
// byte-compares. A difference is a kno determinism bug, not a docs finding,
// and the scenario is where we would find out.
func TestScenarioIsDeterministic(t *testing.T) {
	bin := knoBinary(t)
	root := repoRoot(t)
	findings, err := cmdScenario([]string{
		"--scenario", filepath.Join(root, "scenarios", "support-refunds"),
		"--kno", bin,
		"--repeat", "2",
	})
	if err != nil {
		t.Fatalf("scenario: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("the committed scenario must run clean and repeat identically; got:\n%v", findings)
	}
}

// TestFlagCheckResolvesATwoLevelSubcommand pins both halves of the nested
// lookup at once, because getting one half right is the trap: a checker that
// stopped reporting `--evals` by ignoring nested commands entirely would pass
// the first assertion and check nothing.
//
// `kno eval inspect` is the first command whose flags live on a child. The
// good block must produce no finding (its flags are real, and looking them up
// on `kno eval` — which lists only `-h` — would report a working recipe as
// broken), and the bad block must still produce one.
func TestFlagCheckResolvesATwoLevelSubcommand(t *testing.T) {
	t.Parallel()
	bin := knoBinary(t)
	findings, err := cmdFlags([]string{
		"--recipes", filepath.Join("testdata", "nested-subcommand", "recipes"),
		"--kno", bin,
	})
	if err != nil {
		t.Fatalf("flags: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected exactly one finding — the child's real flags must pass and only the invented one fail; got:\n%v", findings)
	}
	assertFinding(t, findings, "nested-subcommand.md", "`kno eval inspect` has no --maximum-holdout-frac")
}

func knoBinary(t *testing.T) string {
	t.Helper()
	bin := os.Getenv("KNO_BIN")
	if bin == "" {
		t.Skip("KNO_BIN is not set: install a released kno and point KNO_BIN at it")
	}
	abs, err := filepath.Abs(bin)
	if err != nil {
		t.Fatalf("resolve KNO_BIN: %v", err)
	}
	return abs
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// The tests run in cmd/verify.
	return filepath.Join(wd, "..", "..")
}

func assertFinding(t *testing.T, findings []string, wantFile, wantSaying string) {
	t.Helper()
	if len(findings) == 0 {
		t.Fatalf("expected a finding naming %s and saying %q; got none — a runner that passes everything is the failure mode", wantFile, wantSaying)
	}
	for _, f := range findings {
		if contains(f, wantFile) && contains(f, wantSaying) {
			return
		}
	}
	t.Fatalf("no finding named %s and said %q; got:\n%v", wantFile, wantSaying, findings)
}

func contains(haystack, needle string) bool {
	return len(needle) == 0 || indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

func replaceOnce(s, old, replacement string) string {
	i := indexOf(s, old)
	if i < 0 {
		return s
	}
	return s[:i] + replacement + s[i+len(old):]
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o750)
		}
		b, err := os.ReadFile(path) //nolint:gosec // test fixture paths
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, info.Mode().Perm())
	})
}

// TestHelpIsASuccessfulCommand pins the distinction the nightly depends on:
// asking for help succeeds, while a malformed invocation returns the code that
// means "the runner broke" rather than "the docs are wrong".
func TestHelpIsASuccessfulCommand(t *testing.T) {
	for _, arg := range []string{"-h", "--help", "help"} {
		if code := run([]string{arg}); code != exitClean {
			t.Errorf("verify %s = %d, want %d", arg, code, exitClean)
		}
	}
	if code := run(nil); code != exitBroken {
		t.Errorf("verify with no arguments = %d, want %d", code, exitBroken)
	}
	if code := run([]string{"bogus"}); code != exitBroken {
		t.Errorf("verify bogus = %d, want %d", code, exitBroken)
	}
}

// TestLintRefusesABrokenRelativeLink is the other half of "a claim on a page
// is checked by a machine". A link between two pages asserts that the thing it
// points at is there, and nothing in either repository verified that: `make
// docs` in uknoAI/kno skips https targets and the website's crawl skips
// external hrefs, so a relative link that stopped resolving would have gone
// unnoticed on both sides at once.
//
// The corpus also holds the three shapes that must NOT be findings — an
// external URL, a bare anchor, and a target inside a fenced block — because a
// checker that reports those is a checker people turn off.
func TestLintRefusesABrokenRelativeLink(t *testing.T) {
	t.Parallel()
	findings, err := cmdLint([]string{
		"--recipes", filepath.Join("testdata", "broken-link", "recipes"),
		"--scenarios", filepath.Join("testdata", "broken-link", "scenarios"),
		"--root", filepath.Join("testdata", "broken-link"),
	})
	if err != nil {
		t.Fatalf("lint returned an error rather than a finding: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected exactly one finding — the broken link, and neither the external URL, the anchor, nor the fenced block; got:\n%v", findings)
	}
	assertFinding(t, findings, "broken-link.md", `the link target "value-a-pool.md" does not exist`)
}
