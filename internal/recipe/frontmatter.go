// Package recipe parses a recipe's YAML front matter and the fenced blocks in
// its body, and lints both.
//
// The front matter carries the verification tier, and no recipe may omit it.
// Everything a reader is told about how much this page has been checked is
// derived from these fields — never written by hand into the prose — so the
// parser is strict: an unknown key is an error, because a typo in
// `verification` would otherwise silently become "no tier declared".
package recipe

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// Tier is a recipe's verification tier. The three values are the whole
// vocabulary; there is no fourth and no empty default.
type Tier string

// The tiers. Only Executed carries a positive claim.
const (
	// Executed means CI ran these commands end-to-end against the released
	// binary and compared the result to committed expectations.
	Executed Tier = "executed"
	// FlagsOnly means CI checked the command shapes against the released
	// binary's own flag surface. It says nothing about whether the recipe
	// works.
	FlagsOnly Tier = "flags-only"
	// Manual means nothing on this page is machine-checked.
	Manual Tier = "manual"
)

// DateLayout is the only date format the front matter accepts.
const DateLayout = "2006-01-02"

// FrontMatter is a recipe's declared metadata.
//
// The Machine fields (LastVerified, VerifiedAgainst) are written by CI and
// never by a human; Lint recomputes them and a diff is a lint failure.
type FrontMatter struct {
	Verification Tier
	Scenario     string
	Stage        string
	// RequiresStages names the scenario stages that must have run before this
	// recipe's commands can find anything. It is what generates the
	// prior-stage sentence, so that the sentence is declared rather than
	// remembered.
	RequiresStages []string
	// LastVerified and VerifiedAgainst are written by CI.
	LastVerified    time.Time
	VerifiedAgainst string
	Owner           string
	Credentials     []string
	// LastManualVerification is written by a human (or by the vendor-smoke
	// workflow's one-line PR) and is what the 180-day banner reads.
	LastManualVerification time.Time
	Deprecated             bool
}

// Recipe is a parsed recipe file.
type Recipe struct {
	Path        string
	FrontMatter FrontMatter
	Body        string
}

var knownKeys = map[string]bool{
	"verification":             true,
	"scenario":                 true,
	"stage":                    true,
	"requires-stages":          true,
	"last-verified":            true,
	"verified-against":         true,
	"owner":                    true,
	"credentials":              true,
	"last-manual-verification": true,
	"deprecated":               true,
}

// ErrNoFrontMatter reports a file that does not open with a `---` fence.
var ErrNoFrontMatter = errors.New("no front matter: the file must open with a --- fence")

// Load reads and parses one recipe.
func Load(path string) (*Recipe, error) {
	b, err := os.ReadFile(path) //nolint:gosec // paths come from the repo tree, not from input
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	r, err := Parse(string(b))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	r.Path = path
	return r, nil
}

// Parse splits front matter from body and decodes the front matter.
//
// The decoder handles exactly the shapes this schema uses — `key: value`,
// `key: [a, b]`, and a block sequence of `- item` — rather than pulling in a
// YAML dependency for eight fixed keys. An unrecognized shape is an error,
// not a silent skip.
func Parse(src string) (*Recipe, error) {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	if !strings.HasPrefix(src, "---\n") {
		return nil, ErrNoFrontMatter
	}
	rest := src[len("---\n"):]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		return nil, errors.New("front matter is not closed by a --- line")
	}
	fm, err := parseFrontMatter(rest[:end])
	if err != nil {
		return nil, err
	}
	return &Recipe{FrontMatter: *fm, Body: rest[end+len("\n---\n"):]}, nil
}

//nolint:gocognit // one flat switch over a fixed eight-key schema; splitting it hides the schema.
func parseFrontMatter(block string) (*FrontMatter, error) {
	var fm FrontMatter
	seen := map[string]bool{}
	sc := bufio.NewScanner(strings.NewReader(block))
	var listKey string
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if strings.HasPrefix(line, "  - ") || strings.HasPrefix(line, "- ") {
			if listKey == "" {
				return nil, fmt.Errorf("list item %q has no key above it", line)
			}
			if err := appendList(&fm, listKey, unquote(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-")))); err != nil {
				return nil, err
			}
			continue
		}
		listKey = ""
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("front-matter line is not `key: value`: %q", line)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if !knownKeys[key] {
			return nil, fmt.Errorf("unknown front-matter key %q (a typo here would silently drop a claim)", key)
		}
		if seen[key] {
			return nil, fmt.Errorf("duplicate front-matter key %q", key)
		}
		seen[key] = true
		if value == "" {
			listKey = key
			continue
		}
		if err := assign(&fm, key, value); err != nil {
			return nil, err
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("scan front matter: %w", err)
	}
	return &fm, nil
}

func assign(fm *FrontMatter, key, value string) error {
	switch key {
	case "verification":
		fm.Verification = Tier(unquote(value))
	case "scenario":
		fm.Scenario = unquote(value)
	case "stage":
		fm.Stage = unquote(value)
	case "verified-against":
		fm.VerifiedAgainst = unquote(value)
	case "owner":
		fm.Owner = unquote(value)
	case "deprecated":
		fm.Deprecated = unquote(value) == "true"
	case "requires-stages", "credentials":
		items, err := parseInlineList(value)
		if err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		for _, it := range items {
			if err := appendList(fm, key, it); err != nil {
				return err
			}
		}
	case "last-verified", "last-manual-verification":
		t, err := time.Parse(DateLayout, unquote(value))
		if err != nil {
			return fmt.Errorf("%s must be YYYY-MM-DD: %w", key, err)
		}
		if key == "last-verified" {
			fm.LastVerified = t
		} else {
			fm.LastManualVerification = t
		}
	default:
		return fmt.Errorf("unhandled key %q", key)
	}
	return nil
}

func appendList(fm *FrontMatter, key, item string) error {
	if item == "" {
		return nil
	}
	switch key {
	case "requires-stages":
		fm.RequiresStages = append(fm.RequiresStages, item)
	case "credentials":
		fm.Credentials = append(fm.Credentials, item)
	default:
		return fmt.Errorf("key %q does not take a list", key)
	}
	return nil
}

func parseInlineList(value string) ([]string, error) {
	if !strings.HasPrefix(value, "[") || !strings.HasSuffix(value, "]") {
		return nil, fmt.Errorf("expected an inline list in [brackets], got %q", value)
	}
	inner := strings.TrimSpace(value[1 : len(value)-1])
	if inner == "" {
		return nil, nil
	}
	parts := strings.Split(inner, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := unquote(strings.TrimSpace(p)); s != "" {
			out = append(out, s)
		}
	}
	return out, nil
}

func unquote(s string) string {
	if len(s) >= 2 && (s[0] == '"' && s[len(s)-1] == '"' || s[0] == '\'' && s[len(s)-1] == '\'') {
		return s[1 : len(s)-1]
	}
	return s
}
