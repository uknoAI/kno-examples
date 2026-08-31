// Package render turns a recipe's declared verification tier into the badge
// and sentences a reader sees.
//
// # The rendering rule
//
// Wording is not a control. A skimming reader sees a badge shape and a colour
// and has finished forming a belief before reading the sentence underneath it.
// So the renderer carries the load, not the prose:
//
//   - Tier A (executed) is THE ONLY tier in the system with a positive
//     affordance — the only tick, the only green, the only word "Verified".
//   - Tier B (flags-only) renders in the SAME neutral register as Tier C
//     (manual): same icon, same colour token, same weight. It differs only in
//     its text.
//   - Tier C (manual) is the neutral baseline.
//
// `flags-only` therefore *looks* exactly as unverified as `manual`, because
// with respect to the question a reader is actually asking — does this recipe
// work — it is. The difference between them is a claim about what was checked,
// which is a sentence, not a colour.
//
// TestTierBAndTierCAreVisuallyIdentical is what keeps this true when a future
// stylesheet is tempted to give Tier B a colour of its own.
package render

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/uknoAI/kno-examples/internal/recipe"
)

// Colour tokens. There are two, and only Executed may use the positive one.
const (
	// TokenVerified is reserved for `executed`. Nothing else may emit it.
	TokenVerified = "kno-verify-verified"
	// TokenNeutral is what every unverified tier renders in — both of them,
	// identically.
	TokenNeutral = "kno-verify-neutral"
	// TokenStale marks the staleness banner, which is orthogonal to the tier.
	TokenStale = "kno-verify-stale"
)

// Icons. Executed owns the tick; the other two share the neutral dot.
const (
	// IconVerified is the tick, reserved for `executed`.
	IconVerified = "✔"
	// IconNeutral is shared, deliberately, by `flags-only` and `manual`.
	IconNeutral = "•"
)

// Staleness thresholds, in days. A claim with no expiry is a claim nobody
// revisits, so every claim here expires.
const (
	// ExecutedStaleAfterDays is how long a CI-written verification stays
	// current. It is short because CI runs nightly: a Tier A page older than
	// this means the nightly stopped running, which is itself the finding.
	ExecutedStaleAfterDays = 30
	// ManualStaleAfterDays is how long a human's hand-check stays current.
	ManualStaleAfterDays = 180
)

// Badge is the rendered verification block for one recipe.
type Badge struct {
	Tier recipe.Tier
	// Icon and ColorToken are the visual affordance. Tier B and Tier C MUST
	// share both.
	Icon       string
	ColorToken string
	// Text is the sentence under the badge — the only place the two
	// unverified tiers differ.
	Text string
	// PriorStages is the F4 sentence, empty unless the recipe declares
	// requires-stages. It renders adjacent to Text, never in a footnote.
	PriorStages string
	// Staleness is empty unless the recipe's claim has expired.
	Staleness string
	// Deprecated is surfaced by the renderer rather than by prose, because a
	// deprecated recipe is never deleted (external links) and so must say so
	// itself.
	Deprecated bool
}

// Stale reports whether the staleness banner renders.
func (b Badge) Stale() bool { return b.Staleness != "" }

// Render builds the badge for one recipe, as of now.
func Render(fm recipe.FrontMatter, now time.Time) Badge {
	b := Badge{
		Tier:        fm.Verification,
		Icon:        IconNeutral,
		ColorToken:  TokenNeutral,
		PriorStages: recipe.PriorStageSentence(fm),
		Deprecated:  fm.Deprecated,
	}
	switch fm.Verification {
	case recipe.Executed:
		b.Icon = IconVerified
		b.ColorToken = TokenVerified
		b.Text = fmt.Sprintf("Verified end-to-end against %s on %s.",
			fm.VerifiedAgainst, fm.LastVerified.Format(recipe.DateLayout))
		b.Staleness = staleness(fm.LastVerified, now, ExecutedStaleAfterDays,
			"This page's end-to-end verification is %d days old; the nightly run that refreshes it has not reported since %s.")
	case recipe.FlagsOnly:
		b.Text = fmt.Sprintf("Command shapes checked against %s. The vendor steps are not machine-verified; last checked by hand on %s.",
			verifiedAgainstOr(fm), fm.LastManualVerification.Format(recipe.DateLayout))
		b.Staleness = staleness(fm.LastManualVerification, now, ManualStaleAfterDays,
			"Nobody has run the vendor steps on this page for %d days (since %s).")
	case recipe.Manual:
		b.Text = fmt.Sprintf("Not machine-verified. Vendor UIs change; last checked by hand on %s by %s.",
			fm.LastManualVerification.Format(recipe.DateLayout), fm.Owner)
		b.Staleness = staleness(fm.LastManualVerification, now, ManualStaleAfterDays,
			"Nobody has walked through this page for %d days (since %s).")
	default:
		b.Text = "This page declares no verification tier, which is a lint failure — treat nothing on it as checked."
	}
	return b
}

func verifiedAgainstOr(fm recipe.FrontMatter) string {
	if fm.VerifiedAgainst != "" {
		return fm.VerifiedAgainst
	}
	return "the released kno"
}

func staleness(at, now time.Time, afterDays int, format string) string {
	if at.IsZero() {
		return ""
	}
	days := int(now.Sub(at).Hours() / 24)
	if days <= afterDays {
		return ""
	}
	return fmt.Sprintf(format, days, at.Format(recipe.DateLayout))
}

// HTML renders the badge.
//
// The colour token is a class, never an inline colour: a stylesheet that wants
// to give Tier B its own colour has to add a token, and adding a token is what
// the renderer test notices.
func (b Badge) HTML() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, `<div class="kno-verify %s" data-tier="%s">`, b.ColorToken, html.EscapeString(string(b.Tier)))
	fmt.Fprintf(&sb, `<span class="kno-verify-icon" aria-hidden="true">%s</span>`, b.Icon)
	fmt.Fprintf(&sb, `<p class="kno-verify-text">%s</p>`, html.EscapeString(b.Text))
	if b.PriorStages != "" {
		fmt.Fprintf(&sb, `<p class="kno-verify-prior-stages">%s</p>`, html.EscapeString(b.PriorStages))
	}
	if b.Deprecated {
		sb.WriteString(`<p class="kno-verify-deprecated">This recipe is deprecated. It is kept because external links point at it.</p>`)
	}
	if b.Staleness != "" {
		fmt.Fprintf(&sb, `<p class="kno-verify-staleness %s">%s</p>`, TokenStale, html.EscapeString(b.Staleness))
	}
	sb.WriteString(`</div>`)
	return sb.String()
}

// Markdown renders the badge for a plain-markdown consumer — the form that
// lands in the repository's own recipe pages, where there is no stylesheet.
//
// The same rule holds: the tick appears for `executed` and for nothing else.
func (b Badge) Markdown() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "> %s **%s** — %s\n", b.Icon, b.Tier, b.Text)
	if b.PriorStages != "" {
		fmt.Fprintf(&sb, ">\n> %s\n", b.PriorStages)
	}
	if b.Deprecated {
		sb.WriteString(">\n> This recipe is deprecated. It is kept because external links point at it.\n")
	}
	if b.Staleness != "" {
		fmt.Fprintf(&sb, ">\n> **Stale.** %s\n", b.Staleness)
	}
	return sb.String()
}
