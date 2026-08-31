package recipe

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Block is one fenced code block in a recipe body.
type Block struct {
	// Info is the fence's info string, e.g. "bash kno-run scenario=support-refunds stage=baseline".
	Info string
	// Lang is the first word of the info string.
	Lang string
	// Attrs are the key=value pairs after the language and tags.
	Attrs map[string]string
	// Tags are the bare words after the language, e.g. "kno-run".
	Tags map[string]bool
	// Content is the block's body, without the fences and without a trailing
	// newline.
	Content string
	// Line is the 1-based line number of the opening fence within the body.
	Line int
}

// Runnable reports whether CI may execute this block.
//
// Blocks are opt-in, not opt-out: only a block tagged `kno-run` is ever
// executed. Extraction for checking and extraction for execution are separate
// passes on purpose, because their failure modes are opposite — missing a
// check is silent rot, running something unintended is destructive.
func (b Block) Runnable() bool { return b.Tags["kno-run"] }

var fenceRE = regexp.MustCompile("(?m)^```([^\n]*)$")

// Blocks returns every fenced block in a markdown body.
func Blocks(body string) []Block {
	lines := strings.Split(body, "\n")
	var out []Block
	for i := 0; i < len(lines); i++ {
		if !strings.HasPrefix(lines[i], "```") {
			continue
		}
		info := strings.TrimSpace(strings.TrimPrefix(lines[i], "```"))
		var content []string
		j := i + 1
		for ; j < len(lines); j++ {
			if strings.HasPrefix(lines[j], "```") {
				break
			}
			content = append(content, lines[j])
		}
		b := Block{Info: info, Content: strings.Join(content, "\n"), Line: i + 1}
		b.Lang, b.Tags, b.Attrs = parseInfo(info)
		out = append(out, b)
		i = j
	}
	return out
}

func parseInfo(info string) (lang string, tags map[string]bool, attrs map[string]string) {
	tags = map[string]bool{}
	attrs = map[string]string{}
	fields := strings.Fields(info)
	for i, f := range fields {
		if i == 0 {
			lang = f
			continue
		}
		if k, v, ok := strings.Cut(f, "="); ok {
			attrs[k] = v
			continue
		}
		tags[f] = true
	}
	return lang, tags, attrs
}

// Invocation is one `kno` command found in a recipe.
type Invocation struct {
	Subcommand string
	Flags      []string
	Line       int
	Runnable   bool
}

var (
	knoLineRE = regexp.MustCompile(`(^|\s)kno\s+([a-z][a-z-]*)`)
	flagRE    = regexp.MustCompile(`--[a-z][a-z0-9-]*`)
	schemeRE  = regexp.MustCompile(`--(?:agent|evals|pool)[= ]+"?([a-z][a-z0-9_]*):`)
	exportRE  = regexp.MustCompile(`(?m)^\s*export\s+([A-Z][A-Z0-9_]*)=`)
)

// Invocations extracts every `kno` command from every fenced block, runnable
// or not.
//
// This is the checking pass: an untagged block containing a bare `kno`
// invocation is still parsed, because a flag that no longer exists is rot
// whether or not CI is allowed to run the line.
func Invocations(body string) []Invocation {
	var out []Invocation
	for _, b := range Blocks(body) {
		// Join continuation lines so a command split over four lines is one
		// invocation with all its flags.
		joined := strings.ReplaceAll(b.Content, "\\\n", " ")
		for i, line := range strings.Split(joined, "\n") {
			m := knoLineRE.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			out = append(out, Invocation{
				Subcommand: m[2],
				Flags:      dedupe(flagRE.FindAllString(line, -1)),
				Line:       b.Line + i + 1,
				Runnable:   b.Runnable(),
			})
		}
	}
	return out
}

// Schemes returns every adapter scheme named in a `--agent`, `--evals`, or
// `--pool` value anywhere in the body.
func Schemes(body string) []string {
	seen := map[string]bool{}
	for _, m := range schemeRE.FindAllStringSubmatch(body, -1) {
		seen[m[1]] = true
	}
	return sortedKeys(seen)
}

// ExportedEnv returns every environment variable the recipe tells the reader
// to export.
func ExportedEnv(body string) []string {
	seen := map[string]bool{}
	for _, m := range exportRE.FindAllStringSubmatch(body, -1) {
		seen[m[1]] = true
	}
	return sortedKeys(seen)
}

// Include describes a `kno-run` block that quotes a marked region of a
// scenario's run.sh.
type Include struct {
	Scenario string
	Stage    string
	Content  string
	Line     int
}

// Includes returns every runnable block that declares the scenario and stage
// it quotes.
func Includes(body string) ([]Include, error) {
	var out []Include
	for _, b := range Blocks(body) {
		if !b.Runnable() {
			continue
		}
		sc, stage := b.Attrs["scenario"], b.Attrs["stage"]
		if sc == "" || stage == "" {
			return nil, fmt.Errorf("line %d: a `kno-run` block must declare scenario=<slug> stage=<name> so the lint knows what it is quoting", b.Line)
		}
		out = append(out, Include{Scenario: sc, Stage: stage, Content: b.Content, Line: b.Line})
	}
	return out, nil
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	for _, s := range in {
		seen[s] = true
	}
	return sortedKeys(seen)
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
