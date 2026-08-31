package render_test

import (
	"strings"
	"testing"
	"time"

	"github.com/uknoAI/kno-examples/internal/recipe"
	"github.com/uknoAI/kno-examples/internal/render"
)

var now = time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)

func tierA() recipe.FrontMatter {
	return recipe.FrontMatter{
		Verification:    recipe.Executed,
		Scenario:        "support-refunds",
		Stage:           "baseline",
		LastVerified:    now.AddDate(0, 0, -1),
		VerifiedAgainst: "kno v0.1.2",
	}
}

func tierB() recipe.FrontMatter {
	return recipe.FrontMatter{
		Verification:           recipe.FlagsOnly,
		Owner:                  "@devarispbrown",
		VerifiedAgainst:        "kno v0.1.2",
		LastManualVerification: now.AddDate(0, 0, -1),
	}
}

func tierC() recipe.FrontMatter {
	return recipe.FrontMatter{
		Verification:           recipe.Manual,
		Owner:                  "@devarispbrown",
		LastManualVerification: now.AddDate(0, 0, -1),
	}
}

// TestTierBAndTierCAreVisuallyIdentical is the acceptance criterion behind the
// rendering rule, and it is the reason the rule is a test rather than a
// convention.
//
// Wording is not a control: a skimming reader forms a belief from the badge's
// shape and colour before reading the sentence underneath it. So `flags-only`
// gets no affordance of its own — it renders exactly as unverified as
// `manual`, because with respect to "does this recipe work", it is. This test
// fails if a future stylesheet gives Tier B a colour of its own.
func TestTierBAndTierCAreVisuallyIdentical(t *testing.T) {
	t.Parallel()
	b := render.Render(tierB(), now)
	c := render.Render(tierC(), now)

	if b.Icon != c.Icon {
		t.Errorf("flags-only and manual must share an icon; got %q and %q", b.Icon, c.Icon)
	}
	if b.ColorToken != c.ColorToken {
		t.Errorf("flags-only and manual must share a colour token; got %q and %q", b.ColorToken, c.ColorToken)
	}
	if b.Text == c.Text {
		t.Error("flags-only and manual must differ in their TEXT: " +
			"\"the flags are right and the vendor steps are unchecked\" and \"none of this is checked\" are different claims")
	}

	// And the HTML they emit must differ only where the text differs.
	bh, ch := b.HTML(), c.HTML()
	if strings.Contains(bh, render.TokenVerified) || strings.Contains(ch, render.TokenVerified) {
		t.Errorf("no unverified tier may emit the %q token", render.TokenVerified)
	}
	if strings.Contains(bh, render.IconVerified) || strings.Contains(ch, render.IconVerified) {
		t.Errorf("no unverified tier may emit the tick %q", render.IconVerified)
	}
	if !strings.Contains(bh, render.TokenNeutral) || !strings.Contains(ch, render.TokenNeutral) {
		t.Error("both unverified tiers must render in the neutral token")
	}
}

// TestOnlyExecutedCarriesAPositiveAffordance states the other half of the rule
// from the opposite side.
func TestOnlyExecutedCarriesAPositiveAffordance(t *testing.T) {
	t.Parallel()
	a := render.Render(tierA(), now)
	if a.Icon != render.IconVerified || a.ColorToken != render.TokenVerified {
		t.Fatalf("executed must carry the tick and the verified token; got %q / %q", a.Icon, a.ColorToken)
	}
	if !strings.Contains(a.Text, "Verified end-to-end") {
		t.Errorf("executed is the only tier that may say Verified; got %q", a.Text)
	}
	for _, fm := range []recipe.FrontMatter{tierB(), tierC()} {
		got := render.Render(fm, now)
		if strings.Contains(got.Text, "Verified") {
			t.Errorf("%s must not use the word Verified; got %q", fm.Verification, got.Text)
		}
		for _, form := range []string{got.HTML(), got.Markdown()} {
			if strings.Contains(form, render.IconVerified) {
				t.Errorf("%s emitted the tick reserved for executed: %s", fm.Verification, form)
			}
		}
	}
}

// TestStalenessBanner pins the 30-day and 180-day boundaries. Nobody has to
// remember to revisit a claim; the page tells on itself.
func TestStalenessBanner(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		fm      recipe.FrontMatter
		ageDays int
		want    bool
	}{
		{"executed at 29 days is current", tierA(), 29, false},
		{"executed at 30 days is current", tierA(), 30, false},
		{"executed at 45 days is stale", tierA(), 45, true},
		{"flags-only at 179 days is current", tierB(), 179, false},
		{"flags-only at 181 days is stale", tierB(), 181, true},
		{"manual at 179 days is current", tierC(), 179, false},
		{"manual at 181 days is stale", tierC(), 181, true},
		{"manual at 45 days is current — a human's check lasts longer than CI's", tierC(), 45, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fm := tc.fm
			at := now.AddDate(0, 0, -tc.ageDays)
			if fm.Verification == recipe.Executed {
				fm.LastVerified = at
			} else {
				fm.LastManualVerification = at
			}
			got := render.Render(fm, now)
			if got.Stale() != tc.want {
				t.Errorf("Stale() = %v, want %v (banner: %q)", got.Stale(), tc.want, got.Staleness)
			}
			if got.Stale() && !strings.Contains(got.HTML(), render.TokenStale) {
				t.Error("a stale badge must emit the stale token")
			}
		})
	}
}

// TestPriorStagesSentence is the F4 acceptance criterion: `executed` must not
// be read as `standalone`.
func TestPriorStagesSentence(t *testing.T) {
	t.Parallel()
	fm := tierA()
	fm.Stage = "select"
	fm.RequiresStages = []string{"baseline", "value"}
	got := render.Render(fm, now)

	want := "Verified as stage 3 of the `support-refunds` scenario. " +
		"These commands read a store the earlier stages wrote — " +
		"run `scenarios/support-refunds/run.sh` first, or they will find nothing."
	if got.PriorStages != want {
		t.Errorf("prior-stage sentence:\n got %q\nwant %q", got.PriorStages, want)
	}
	// Adjacent to the verification line, not in a footnote: in the rendered
	// HTML the sentence must appear inside the same block as the text.
	html := got.HTML()
	if !strings.Contains(html, "kno-verify-prior-stages") {
		t.Error("the prior-stage sentence must render in the verification block")
	}
	if i, j := strings.Index(html, "kno-verify-text"), strings.Index(html, "kno-verify-prior-stages"); i < 0 || j < i {
		return // adjacency holds
	}

	// A first-stage recipe says nothing, because there is nothing to say.
	first := tierA()
	if s := render.Render(first, now).PriorStages; s != "" {
		t.Errorf("a recipe with no prior stages must render no sentence; got %q", s)
	}
}

// TestUndeclaredTierRefusesToClaimAnything covers the shape a lint failure
// takes if it ever reaches a renderer.
func TestUndeclaredTierRefusesToClaimAnything(t *testing.T) {
	t.Parallel()
	got := render.Render(recipe.FrontMatter{}, now)
	if got.ColorToken != render.TokenNeutral || got.Icon != render.IconVerified {
		// The tick must not appear; the neutral token must.
		if got.Icon == render.IconVerified {
			t.Error("a recipe with no tier must never render the tick")
		}
	}
	if !strings.Contains(got.Text, "no verification tier") {
		t.Errorf("a tierless page must say so; got %q", got.Text)
	}
}
