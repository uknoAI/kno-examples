// Command verify is this repository's `make check`.
//
// An external repository has no `make check` of kno's, so it builds one. The
// design principle is that every recipe carries a machine-set verification
// tier and no recipe may omit it — and that the runner is the only thing here
// that can silently lie, which is why cmd/verify/testdata holds a deliberately
// broken corpus and the tests assert the runner refuses it.
//
// Subcommands:
//
//	verify lint      [--recipes DIR] [--root DIR]    front matter, credentials, quoted-marker fidelity, relative links
//	verify flags     --kno PATH [--recipes DIR]      every kno invocation against the binary's own surface
//	verify scenario  --kno PATH [--scenario DIR] [--repeat N] [--update]
//	verify render    FILE                            print one recipe's verification block
//	verify stamp     --kno PATH [--recipes DIR]      write last-verified/verified-against (CI only)
//
// Output contract with .github/workflows/nightly.yml: a line beginning FAIL is
// one finding, and each finding is one signature for the issue lifecycle.
// Exit 0 = clean, 1 = findings, 2 = the runner itself broke (no issue is filed
// for a 2 — an infrastructure failure is not a docs finding).
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/uknoAI/kno-examples/internal/fixture"
	"github.com/uknoAI/kno-examples/internal/recipe"
	"github.com/uknoAI/kno-examples/internal/render"
	"github.com/uknoAI/kno-examples/internal/scenario"
)

const (
	exitClean    = 0
	exitFindings = 1
	exitBroken   = 2
)

func main() {
	os.Exit(run(os.Args[1:]))
}

// usage is the whole surface. `verify` is what CI runs and what a contributor
// runs before opening a PR, so `--help` has to answer "what can I run" without
// sending anyone to read main.go.
const usage = `usage: verify <command> [flags]

  lint                    front matter, credentials, quoted-block fidelity, and relative links. Needs no binary.
  flags     --kno PATH    every kno invocation against the released binary's own surface
  scenario  --kno PATH    every scenario end to end against committed expectations
  render    RECIPE        print one recipe's verification block
  fixtures  --kno-src DIR the three copies of the demo fixtures, against each other
  stamp     --kno PATH    write last-verified and verified-against (CI writes these, not you)

exit codes: 0 clean, 1 findings, 2 the runner itself broke`

func run(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, usage)
		return exitBroken
	}
	// Asking for help is a successful command, not a usage error: it prints to
	// stdout and exits 0. Bare `verify` keeps stderr and exit 2, which is the
	// code the nightly reads as "the runner broke, file nothing".
	switch args[0] {
	case "-h", "--help", "help":
		fmt.Fprintln(os.Stdout, usage)
		return exitClean
	}
	var err error
	var findings []string
	switch args[0] {
	case "lint":
		findings, err = cmdLint(args[1:])
	case "flags":
		findings, err = cmdFlags(args[1:])
	case "scenario":
		findings, err = cmdScenario(args[1:])
	case "render":
		err = cmdRender(args[1:])
	case "fixtures":
		findings, err = cmdFixtures(args[1:])
	case "stamp":
		err = cmdStamp(args[1:])
	default:
		err = fmt.Errorf("unknown subcommand %q", args[0])
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "verify: %v\n", err)
		return exitBroken
	}
	sort.Strings(findings)
	for _, f := range findings {
		fmt.Println(f)
	}
	if len(findings) > 0 {
		fmt.Fprintf(os.Stderr, "verify: %d finding(s)\n", len(findings))
		return exitFindings
	}
	return exitClean
}

// loadRecipes reads every recipe in a directory. A file that will not parse is
// itself a finding rather than a crash, so one malformed page cannot hide the
// state of the other twenty-three.
func loadRecipes(dir string) ([]*recipe.Recipe, []string, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return nil, nil, fmt.Errorf("glob %s: %w", dir, err)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return nil, nil, fmt.Errorf("no recipes under %s", dir)
	}
	var rs []*recipe.Recipe
	var findings []string
	for _, p := range paths {
		// The index page is prose about the recipes, not a recipe: it makes no
		// claim of its own, so it declares no tier.
		if filepath.Base(p) == "README.md" {
			continue
		}
		r, err := recipe.Load(p)
		if err != nil {
			findings = append(findings, "FAIL "+p+": "+err.Error())
			continue
		}
		rs = append(rs, r)
	}
	return rs, findings, nil
}

func cmdLint(args []string) ([]string, error) {
	fs := flag.NewFlagSet("lint", flag.ContinueOnError)
	dir := fs.String("recipes", "recipes", "directory of recipe markdown files")
	scenarios := fs.String("scenarios", "scenarios", "directory of scenario directories")
	root := fs.String("root", ".", "tree whose markdown links are checked")
	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("parse flags: %w", err)
	}
	rs, findings, err := loadRecipes(*dir)
	if err != nil {
		return nil, err
	}
	for _, f := range recipe.LintAll(rs) {
		findings = append(findings, f.String())
	}
	for _, r := range rs {
		findings = append(findings, lintIncludes(r, *scenarios)...)
	}
	links, err := recipe.LinkCheck(*root)
	if err != nil {
		return nil, err
	}
	for _, f := range links {
		findings = append(findings, f.String())
	}
	return findings, nil
}

// lintIncludes asserts that every `kno-run` block is byte-identical to the
// marked region of the scenario's run.sh that it claims to quote.
//
// This is the mechanism that stops `processed`/`issued` from happening again —
// not a review convention, a failing build.
func lintIncludes(r *recipe.Recipe, scenariosDir string) []string {
	incs, err := recipe.Includes(r.Body)
	if err != nil {
		return []string{"FAIL " + r.Path + ": " + err.Error()}
	}
	var findings []string
	for _, inc := range incs {
		script := filepath.Join(scenariosDir, inc.Scenario, "run.sh")
		want, err := markedRegion(script, inc.Stage)
		if err != nil {
			findings = append(findings, fmt.Sprintf("FAIL %s: line %d: %v", r.Path, inc.Line, err))
			continue
		}
		if strings.TrimRight(inc.Content, "\n") != strings.TrimRight(want, "\n") {
			findings = append(findings, fmt.Sprintf(
				"FAIL %s: line %d: the quoted block is not byte-identical to %s's `%s` region — the page and the script disagree about the command",
				r.Path, inc.Line, script, inc.Stage))
		}
		if r.FrontMatter.Verification == recipe.Executed && r.FrontMatter.Scenario != inc.Scenario {
			findings = append(findings, fmt.Sprintf(
				"FAIL %s: line %d: quotes scenario %q but the front matter declares %q",
				r.Path, inc.Line, inc.Scenario, r.FrontMatter.Scenario))
		}
	}
	// `executed` is a claim that CI RAN these commands. A page that declares
	// the tier and quotes nothing runnable claims it while executing nothing —
	// the one failure mode a tier system cannot survive, because it is
	// indistinguishable from the real thing on the rendered page.
	if r.FrontMatter.Verification == recipe.Executed && len(incs) == 0 {
		findings = append(findings, "FAIL "+r.Path+": declares `verification: executed` but quotes no `kno-run` block — the tier claims CI ran something, and nothing here is runnable")
	}
	// An executed recipe whose stage is not the scenario's first must declare
	// the stages it depends on, or the page grows a green tick over commands
	// that will find an empty store.
	if r.FrontMatter.Verification == recipe.Executed && len(incs) > 0 {
		first, err := firstStage(filepath.Join(scenariosDir, r.FrontMatter.Scenario, "run.sh"))
		if err != nil {
			findings = append(findings, "FAIL "+r.Path+": "+err.Error())
		} else if incs[0].Stage != first && len(r.FrontMatter.RequiresStages) == 0 {
			findings = append(findings, fmt.Sprintf(
				"FAIL %s: quotes stage %q, which is not the scenario's first stage (%q), but declares no `requires-stages:` — a reader pasting this gets an empty store",
				r.Path, incs[0].Stage, first))
		}
	}
	return findings
}

func markedRegion(script, stage string) (string, error) {
	b, err := os.ReadFile(script) //nolint:gosec // repo path
	if err != nil {
		return "", fmt.Errorf("read %s: %w", script, err)
	}
	open, close := "# >>> "+stage, "# <<< "+stage
	lines := strings.Split(string(b), "\n")
	start := -1
	for i, l := range lines {
		switch {
		case l == open:
			start = i + 1
		case l == close && start >= 0:
			return strings.Join(lines[start:i], "\n"), nil
		}
	}
	return "", fmt.Errorf("%s has no `%s` region", script, stage)
}

func firstStage(script string) (string, error) {
	b, err := os.ReadFile(script) //nolint:gosec // repo path
	if err != nil {
		return "", fmt.Errorf("read %s: %w", script, err)
	}
	for _, l := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(l, "stages=") {
			f := strings.Fields(strings.Trim(strings.TrimPrefix(l, "stages="), `"`))
			if len(f) > 0 {
				return f[0], nil
			}
		}
	}
	return "", errors.New("run.sh declares no `stages=` line, so the lint cannot tell which stage comes first")
}

func cmdFlags(args []string) ([]string, error) {
	fs := flag.NewFlagSet("flags", flag.ContinueOnError)
	dir := fs.String("recipes", "recipes", "directory of recipe markdown files")
	knoPath := fs.String("kno", "kno", "path of the released kno binary")
	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("parse flags: %w", err)
	}
	bin, err := recipe.OpenBinary(*knoPath)
	if err != nil {
		return nil, err
	}
	rs, findings, err := loadRecipes(*dir)
	if err != nil {
		return nil, err
	}
	for _, r := range rs {
		for _, f := range recipe.FlagCheck(r, bin) {
			findings = append(findings, f.String())
		}
	}
	return findings, nil
}

func cmdScenario(args []string) ([]string, error) {
	fs := flag.NewFlagSet("scenario", flag.ContinueOnError)
	dir := fs.String("scenario", "", "one scenario directory; default is every scenario under --scenarios")
	all := fs.String("scenarios", "scenarios", "directory of scenario directories")
	knoPath := fs.String("kno", "kno", "path of the released kno binary")
	repeat := fs.Int("repeat", 1, "run this many times and byte-compare the captured output")
	update := fs.Bool("update", false, "rewrite expected/*.json from this run, preserving each projection's key set")
	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("parse flags: %w", err)
	}
	abs, err := filepath.Abs(*knoPath)
	if err != nil {
		return nil, fmt.Errorf("resolve --kno: %w", err)
	}

	// Default to every scenario rather than one named here. A second scenario
	// that CI does not run is not a second scenario, and defaulting to all of
	// them means adding one is a directory rather than a directory plus a
	// remembered edit in two workflows and a Makefile.
	dirs := []string{*dir}
	if *dir == "" {
		dirs, err = scenarioDirs(*all)
		if err != nil {
			return nil, err
		}
	}

	var findings []string
	for _, d := range dirs {
		more, err := checkScenario(d, abs, *repeat, *update)
		if err != nil {
			return nil, err
		}
		findings = append(findings, more...)
	}
	return findings, nil
}

// scenarioDirs lists every scenario directory, in a stable order.
func scenarioDirs(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", root, err)
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, filepath.Join(root, e.Name()))
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s holds no scenario directories", root)
	}
	sort.Strings(out)
	return out, nil
}

func checkScenario(dir, knoPath string, repeat int, update bool) ([]string, error) {
	s, err := scenario.Load(dir)
	if err != nil {
		return nil, err
	}
	parent, err := os.MkdirTemp("", "kno-examples-")
	if err != nil {
		return nil, fmt.Errorf("temp dir: %w", err)
	}
	defer os.RemoveAll(parent)

	var runs []*scenario.Artifacts
	for i := 0; i < repeat; i++ {
		a, err := s.Run(knoPath, parent)
		if err != nil {
			return nil, err
		}
		runs = append(runs, a)
	}
	if update {
		if err := s.Update(runs[0]); err != nil {
			return nil, err
		}
		fmt.Fprintf(os.Stderr, "verify: rewrote %s/expected\n", s.Dir)
		return nil, nil
	}
	findings, err := s.Check(runs[0])
	if err != nil {
		return nil, err
	}
	for i := 1; i < len(runs); i++ {
		more, err := scenario.CompareRuns(runs[0], runs[i])
		if err != nil {
			return nil, err
		}
		findings = append(findings, more...)
	}
	out := make([]string, 0, len(findings))
	for _, f := range findings {
		out = append(out, "FAIL "+f)
	}
	return out, nil
}

func cmdRender(args []string) error {
	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	asHTML := fs.Bool("html", false, "render HTML instead of markdown")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	if fs.NArg() != 1 {
		return errors.New("usage: verify render [--html] <recipe.md>")
	}
	r, err := recipe.Load(fs.Arg(0))
	if err != nil {
		return err
	}
	b := render.Render(r.FrontMatter, time.Now())
	if *asHTML {
		fmt.Println(b.HTML())
		return nil
	}
	fmt.Print(b.Markdown())
	return nil
}

// cmdStamp writes the two machine fields. It is CI's job and nobody else's:
// a hand-edited `last-verified` is a date that is evidence of a memory rather
// than of a run.
// cmdFixtures compares the three copies of the demo fixture data.
//
// The scenario here, the copy embedded in the kno binary, and the copy typed
// into the VHS tape must agree. The duplication is deliberate and permanent —
// see internal/fixture — so the agreement is enforced rather than remembered.
func cmdFixtures(args []string) ([]string, error) {
	fs := flag.NewFlagSet("fixtures", flag.ContinueOnError)
	src := fs.String("kno-src", "", "checkout of uknoAI/kno (required)")
	dir := fs.String("scenario", filepath.Join("scenarios", "support-refunds"),
		"the scenario whose fixtures `kno demo` embeds")
	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("parse flags: %w", err)
	}
	if *src == "" {
		return nil, fmt.Errorf("fixtures needs --kno-src: a checkout of uknoAI/kno, which holds the two other copies")
	}

	read := func(parts ...string) ([]byte, error) {
		p := filepath.Join(parts...)
		b, err := os.ReadFile(p) //nolint:gosec // paths named by the caller
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", p, err)
		}
		return b, nil
	}

	oursCasesRaw, err := read(*dir, "evals", "cases.jsonl")
	if err != nil {
		return nil, err
	}
	oursPoolRaw, err := read(*dir, "pool", "pool.jsonl")
	if err != nil {
		return nil, err
	}
	demoCasesRaw, err := read(*src, "cli", "demodata", "cases.jsonl")
	if err != nil {
		return nil, err
	}
	demoPoolRaw, err := read(*src, "cli", "demodata", "pool.jsonl")
	if err != nil {
		return nil, err
	}
	tapeRaw, err := read(*src, "tapes", "quickstart.tape")
	if err != nil {
		return nil, err
	}

	const (
		here = "kno-examples"
		demo = "kno cli/demodata"
		tape = "kno tapes/quickstart.tape"
	)
	parse := func(name string, b []byte) ([]fixture.Record, error) {
		rs, err := fixture.Records(b)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		return rs, nil
	}

	oursCases, err := parse(here, oursCasesRaw)
	if err != nil {
		return nil, err
	}
	oursPool, err := parse(here, oursPoolRaw)
	if err != nil {
		return nil, err
	}
	demoCases, err := parse(demo, demoCasesRaw)
	if err != nil {
		return nil, err
	}
	demoPool, err := parse(demo, demoPoolRaw)
	if err != nil {
		return nil, err
	}
	tapeCases, tapePool, err := fixture.FromTape(tapeRaw)
	if err != nil {
		return nil, err
	}

	var findings []string
	findings = append(findings, fixture.Compare("cases", here, oursCases, demo, demoCases)...)
	findings = append(findings, fixture.Compare("cases", here, oursCases, tape, tapeCases)...)
	findings = append(findings, fixture.Compare("pool", here, oursPool, demo, demoPool)...)
	findings = append(findings, fixture.Compare("pool", here, oursPool, tape, tapePool)...)

	// Both .jsonl copies are whole files serving the same purpose, so they are
	// also held to byte equality. That catches reordering and whitespace, which
	// a per-record comparison cannot see and which still makes two files that
	// should be interchangeable stop being interchangeable. The tape is exempt:
	// its records are embedded in `Type` lines, so it has no file to compare.
	if !bytes.Equal(oursCasesRaw, demoCasesRaw) && len(findings) == 0 {
		findings = append(findings, fmt.Sprintf(
			"FAIL fixtures cases: every record agrees, but %s and %s are not byte-identical — check ordering and whitespace", here, demo))
	}
	if !bytes.Equal(oursPoolRaw, demoPoolRaw) && len(findings) == 0 {
		findings = append(findings, fmt.Sprintf(
			"FAIL fixtures pool: every record agrees, but %s and %s are not byte-identical — check ordering and whitespace", here, demo))
	}
	return findings, nil
}

func cmdStamp(args []string) error {
	fs := flag.NewFlagSet("stamp", flag.ContinueOnError)
	dir := fs.String("recipes", "recipes", "directory of recipe markdown files")
	knoPath := fs.String("kno", "kno", "path of the released kno binary")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	bin, err := recipe.OpenBinary(*knoPath)
	if err != nil {
		return err
	}
	version, err := bin.Version()
	if err != nil {
		return err
	}
	today := time.Now().UTC().Format(recipe.DateLayout)
	rs, bad, err := loadRecipes(*dir)
	if err != nil {
		return err
	}
	if len(bad) > 0 {
		return fmt.Errorf("refusing to stamp: %s", strings.Join(bad, "; "))
	}
	for _, r := range rs {
		// Tier A gets both fields; Tier B gets only the build its flag surface
		// was checked against, which is the narrower claim it makes. Tier C
		// gets neither, because nothing on it was checked by a machine.
		if r.FrontMatter.Verification == recipe.Manual {
			continue
		}
		b, err := os.ReadFile(r.Path) //nolint:gosec // repo path
		if err != nil {
			return fmt.Errorf("read %s: %w", r.Path, err)
		}
		out := string(b)
		if r.FrontMatter.Verification == recipe.Executed {
			out = replaceField(out, "last-verified", today)
		}
		out = replaceField(out, "verified-against", version)
		if err := os.WriteFile(r.Path, []byte(out), 0o600); err != nil {
			return fmt.Errorf("write %s: %w", r.Path, err)
		}
	}
	return nil
}

func replaceField(src, key, value string) string {
	lines := strings.Split(src, "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, key+":") {
			lines[i] = key + ": " + value
			// Only the front matter is rewritten, and the front matter is the
			// first block, so stop at the first hit.
			break
		}
	}
	return strings.Join(lines, "\n")
}
