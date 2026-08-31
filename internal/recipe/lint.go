package recipe

import (
	"fmt"
	"sort"
	"strings"
)

// Finding is one lint result. The Signature is the finding with digit runs
// collapsed, so the nightly workflow can intersect a run's findings against
// the open `docs-drift` issues without matching on a count that changes.
type Finding struct {
	Path    string
	Message string
}

// String renders a finding the way the workflow's contract expects: one line,
// beginning with FAIL.
func (f Finding) String() string { return "FAIL " + f.Path + ": " + f.Message }

// SchemeCredentials maps an adapter scheme, as it appears in `--agent
// <scheme>:` and in `kno doctor --json`, to the environment variables a reader
// must have set for it to work.
//
// This table exists because of a specific, observed failure: the vendor
// recipes teach a reader about the vendor's token and never mention that
// `--agent openai:gpt-4.1` also bills OpenAI. `OPENAI_API_KEY` was named
// exactly once in the whole cookbook, in a scheme table in `your-own-provider`,
// and thereafter merely implied.
var SchemeCredentials = map[string][]string{
	"openai":    {"OPENAI_API_KEY"},
	"anthropic": {"ANTHROPIC_API_KEY"},
	"bedrock":   {"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_REGION"},
	"vertex":    {"GOOGLE_APPLICATION_CREDENTIALS"},
	// Evals and Pool schemes.
	"langsmith":   {"LANGSMITH_API_KEY"},
	"langfuse":    {"LANGFUSE_PUBLIC_KEY", "LANGFUSE_SECRET_KEY"},
	"braintrust":  {"BRAINTRUST_API_KEY"},
	"hf":          {"HF_TOKEN"},
	"fake":        nil,
	"exec":        nil,
	"tuned":       nil,
	"csv":         nil,
	"md":          nil,
	"context":     nil,
	"tuning_set":  nil,
	"http":        nil,
	"https":       nil,
	"knowledge_b": nil,
}

// Lint checks one recipe's front matter and the internal consistency of its
// body. It reports every problem it finds rather than the first, because a
// contributor fixing one field at a time is a contributor who stops
// contributing.
//
// It does NOT run any command: that is Scenario (tier A) and FlagCheck (every
// tier). Lint is the part that needs no binary.
func Lint(r *Recipe) []Finding {
	var out []Finding
	add := func(format string, args ...any) {
		out = append(out, Finding{Path: r.Path, Message: fmt.Sprintf(format, args...)})
	}
	fm := r.FrontMatter

	switch fm.Verification {
	case Executed, FlagsOnly, Manual:
	case "":
		add("no `verification:` field. Every recipe declares a tier; there is no default.")
	default:
		add("verification: %q is not one of executed, flags-only, manual", fm.Verification)
	}

	if fm.Verification == Executed {
		if fm.Scenario == "" {
			add("verification: executed requires `scenario:` — the runner has nothing to execute without it")
		}
		if fm.Stage == "" {
			add("verification: executed requires `stage:` — which stage's committed output this page's claims are checked against")
		}
		if fm.LastVerified.IsZero() {
			add("verification: executed requires `last-verified:` (written by CI)")
		}
		if fm.VerifiedAgainst == "" {
			add("verification: executed requires `verified-against:` (written by CI)")
		}
	} else {
		if fm.Owner == "" {
			add("verification: %s requires `owner:` — an unverified page with nobody to ask is not a page, it is a rumour", fm.Verification)
		}
		if fm.LastManualVerification.IsZero() {
			add("verification: %s requires `last-manual-verification:` — the 180-day staleness banner reads it", fm.Verification)
		}
		// `last-verified` is the end-to-end claim and belongs to Tier A alone.
		// `verified-against` is narrower — "which build's flag surface was
		// this checked against" — so Tier B needs it and Tier C, which checks
		// nothing mechanically, must not carry it.
		if !fm.LastVerified.IsZero() {
			add("only an `executed` recipe may carry `last-verified`: it is the end-to-end claim, and this tier does not make it")
		}
		if fm.Verification == FlagsOnly && fm.VerifiedAgainst == "" {
			add("verification: flags-only requires `verified-against:` (written by CI) — \"the flags check out\" is meaningless without naming the build they were checked against")
		}
		if fm.Verification == Manual && fm.VerifiedAgainst != "" {
			add("verification: manual must not carry `verified-against:` — nothing on the page was checked against a build")
		}
	}

	out = append(out, lintCredentials(r)...)
	return out
}

// lintCredentials asserts that every credential the recipe's own commands
// imply is named in `credentials:`.
func lintCredentials(r *Recipe) []Finding {
	declared := map[string]bool{}
	for _, c := range r.FrontMatter.Credentials {
		declared[c] = true
	}
	required := map[string]string{} // env -> the scheme that implied it
	for _, s := range Schemes(r.Body) {
		envs, known := SchemeCredentials[s]
		if !known {
			return []Finding{{
				Path:    r.Path,
				Message: fmt.Sprintf("scheme %q is not in the scheme→credential table; add it (with its required environment variables) or fix the typo", s),
			}}
		}
		for _, e := range envs {
			required[e] = s
		}
	}
	// A recipe that exports a variable itself has named it, and that counts.
	for _, e := range ExportedEnv(r.Body) {
		required[e] = "an `export` in the recipe"
	}

	missing := make([]string, 0, len(required))
	for e := range required {
		if !declared[e] {
			missing = append(missing, e)
		}
	}
	sort.Strings(missing)
	var out []Finding
	for _, e := range missing {
		out = append(out, Finding{
			Path:    r.Path,
			Message: fmt.Sprintf("%s is required (implied by %s) but is not listed in `credentials:`", e, required[e]),
		})
	}
	return out
}

// LintAll lints a set of recipes and returns the findings sorted by path so a
// run's output is stable enough to diff.
func LintAll(rs []*Recipe) []Finding {
	var out []Finding
	for _, r := range rs {
		out = append(out, Lint(r)...)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Message < out[j].Message
	})
	return out
}

// PriorStageSentence is the sentence an `executed` recipe with prior stages
// must render, generated from `requires-stages` rather than remembered.
//
// It exists because `executed` must not be read as `standalone`: a reader who
// pastes stage three without running stages one and two gets an empty store
// and a confusing failure, which is the false confidence this repository
// exists to destroy, re-created underneath a green tick.
func PriorStageSentence(fm FrontMatter) string {
	if fm.Verification != Executed || len(fm.RequiresStages) == 0 {
		return ""
	}
	return fmt.Sprintf(
		"Verified as stage %d of the `%s` scenario. These commands read a store the earlier stages wrote — run `scenarios/%s/run.sh` first, or they will find nothing.",
		len(fm.RequiresStages)+1, fm.Scenario, fm.Scenario,
	)
}

// Signature normalizes a finding line for the nightly issue lifecycle: digit
// runs collapse to N, so "3 of 12 cases" and "4 of 12 cases" are one finding
// rather than two issues.
func Signature(line string) string {
	var b strings.Builder
	inDigits := false
	for _, r := range line {
		if r >= '0' && r <= '9' {
			if !inDigits {
				b.WriteByte('N')
				inDigits = true
			}
			continue
		}
		inDigits = false
		b.WriteRune(r)
	}
	return b.String()
}
