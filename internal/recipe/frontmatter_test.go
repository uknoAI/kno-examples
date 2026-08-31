package recipe_test

import (
	"strings"
	"testing"
	"time"

	"github.com/uknoAI/kno-examples/internal/recipe"
)

func TestParseAcceptsTheSchema(t *testing.T) {
	t.Parallel()
	src := `---
verification: executed
scenario: support-refunds
stage: select
requires-stages: [baseline, value]
last-verified: 2026-08-31
verified-against: kno v0.1.2
---

# Body
`
	r, err := recipe.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	fm := r.FrontMatter
	if fm.Verification != recipe.Executed || fm.Scenario != "support-refunds" || fm.Stage != "select" {
		t.Errorf("front matter: %+v", fm)
	}
	if len(fm.RequiresStages) != 2 || fm.RequiresStages[0] != "baseline" || fm.RequiresStages[1] != "value" {
		t.Errorf("requires-stages: %v", fm.RequiresStages)
	}
	if fm.LastVerified.Format(recipe.DateLayout) != "2026-08-31" {
		t.Errorf("last-verified: %v", fm.LastVerified)
	}
	if !strings.HasPrefix(r.Body, "\n# Body") {
		t.Errorf("body: %q", r.Body)
	}
}

// A typo in a key must be an error rather than a silently dropped claim: a
// recipe with `verifcation: executed` would otherwise lint as "no tier
// declared" in a way a reviewer could easily wave through.
func TestParseRefusesAnUnknownKey(t *testing.T) {
	t.Parallel()
	_, err := recipe.Parse("---\nverifcation: executed\n---\n")
	if err == nil || !strings.Contains(err.Error(), "unknown front-matter key") {
		t.Fatalf("expected an unknown-key error, got %v", err)
	}
}

func TestParseRefusesAFileWithNoFrontMatter(t *testing.T) {
	t.Parallel()
	if _, err := recipe.Parse("# Just a heading\n"); err == nil {
		t.Fatal("expected a refusal: every recipe declares a tier, and a file with no front matter declares nothing")
	}
}

func TestBlocksAreOptInForExecution(t *testing.T) {
	t.Parallel()
	body := "```bash\nrm -rf /tmp/something\n```\n\n```bash kno-run scenario=s stage=baseline\nkno baseline --evals cases.jsonl\n```\n"
	blocks := recipe.Blocks(body)
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	if blocks[0].Runnable() {
		t.Error("an untagged block must never be executed, however innocent it looks")
	}
	if !blocks[1].Runnable() {
		t.Error("a kno-run block must be runnable")
	}
	// But BOTH are still parsed for flag checking: missing a check is silent
	// rot, which is the failure mode this repository exists to catch.
	if got := recipe.Invocations(body); len(got) != 1 || got[0].Subcommand != "baseline" {
		t.Errorf("invocations: %+v", got)
	}
}

func TestInvocationsJoinContinuationLines(t *testing.T) {
	t.Parallel()
	body := "```bash\nkno value --evals cases.jsonl \\\n  --pool pool.jsonl --baseline-run-id x\n```\n"
	inv := recipe.Invocations(body)
	if len(inv) != 1 {
		t.Fatalf("expected one invocation, got %d", len(inv))
	}
	want := map[string]bool{"--evals": true, "--pool": true, "--baseline-run-id": true}
	for _, f := range inv[0].Flags {
		delete(want, f)
	}
	if len(want) != 0 {
		t.Errorf("a command split over lines must yield all its flags; missed %v", want)
	}
}

func TestSchemesAndCredentials(t *testing.T) {
	t.Parallel()
	body := "```bash\nkno baseline --evals langsmith:support --agent openai:gpt-4.1\n```\n"
	if got := recipe.Schemes(body); len(got) != 2 || got[0] != "langsmith" || got[1] != "openai" {
		t.Fatalf("schemes: %v", got)
	}
	r := &recipe.Recipe{
		Path: "x.md",
		FrontMatter: recipe.FrontMatter{
			Verification:           recipe.FlagsOnly,
			Owner:                  "@someone",
			VerifiedAgainst:        "kno v0.1.2",
			LastManualVerification: mustDate(t, "2026-08-31"),
			Credentials:            []string{"LANGSMITH_API_KEY"},
		},
		Body: body,
	}
	findings := recipe.Lint(r)
	if len(findings) != 1 || !strings.Contains(findings[0].Message, "OPENAI_API_KEY") {
		t.Fatalf("a recipe that bills OpenAI must say so; findings: %v", findings)
	}
}

func TestPriorStageSentenceIsGeneratedNotRemembered(t *testing.T) {
	t.Parallel()
	fm := recipe.FrontMatter{
		Verification:   recipe.Executed,
		Scenario:       "support-refunds",
		RequiresStages: []string{"baseline", "value"},
	}
	got := recipe.PriorStageSentence(fm)
	for _, want := range []string{"stage 3", "support-refunds", "scenarios/support-refunds/run.sh", "find nothing"} {
		if !strings.Contains(got, want) {
			t.Errorf("sentence %q does not name %q", got, want)
		}
	}
	if s := recipe.PriorStageSentence(recipe.FrontMatter{Verification: recipe.FlagsOnly}); s != "" {
		t.Errorf("only an executed recipe carries the sentence; got %q", s)
	}
}

func TestSignatureCollapsesDigitRuns(t *testing.T) {
	t.Parallel()
	a := recipe.Signature("FAIL x.md: line 50: expected 4, got 6")
	b := recipe.Signature("FAIL x.md: line 63: expected 4, got 8")
	if a != b {
		t.Errorf("the same finding at a moved line number must share a signature:\n%q\n%q", a, b)
	}
}

func mustDate(t *testing.T, s string) time.Time {
	t.Helper()
	r, err := recipe.Parse("---\nverification: manual\nowner: \"@x\"\nlast-manual-verification: " + s + "\n---\n")
	if err != nil {
		t.Fatalf("date fixture: %v", err)
	}
	return r.FrontMatter.LastManualVerification
}

// TestAgentSchemesExcludeEvalsAndPoolSchemes pins the split that keeps the
// flag check from manufacturing findings.
//
// `kno doctor --json` enumerates AGENT adapters and nothing else, so asserting
// an Evals or Pool scheme against it reports four working vendor recipes as
// broken. A checker that cries wolf is a checker people stop reading, so the
// doctor assertion sees only `--agent` values; Evals and Pool schemes are
// checked against the scheme→credential table by Lint instead.
func TestAgentSchemesExcludeEvalsAndPoolSchemes(t *testing.T) {
	t.Parallel()
	body := "```bash\n" +
		"kno baseline --evals langsmith:my-dataset --agent openai:gpt-4.1 --yes\n" +
		"kno value --evals hf:acme/support --pool braintrust:proj --agent anthropic:claude-opus-5\n" +
		"```\n"

	all := recipe.Schemes(body)
	want := []string{"anthropic", "braintrust", "hf", "langsmith", "openai"}
	if len(all) != len(want) {
		t.Fatalf("recipe.Schemes() = %v, want %v", all, want)
	}
	for i, w := range want {
		if all[i] != w {
			t.Fatalf("recipe.Schemes() = %v, want %v", all, want)
		}
	}

	agents := recipe.AgentSchemes(body)
	wantAgents := []string{"anthropic", "openai"}
	if len(agents) != len(wantAgents) {
		t.Fatalf("recipe.AgentSchemes() = %v, want %v", agents, wantAgents)
	}
	for i, w := range wantAgents {
		if agents[i] != w {
			t.Fatalf("recipe.AgentSchemes() = %v, want %v", agents, wantAgents)
		}
	}
}
