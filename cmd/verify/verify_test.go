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
