package recipe

import (
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

// Binary is a released kno binary, interrogated for its own flag surface.
//
// The check is against the binary users can actually download, not against a
// table we maintain: a table would be a second copy of the truth, and this
// repository exists because second copies drift.
type Binary struct {
	Path string

	subcommands map[string]bool
	flags       map[string]map[string]bool
	schemes     map[string]bool
}

var (
	helpFlagRE = regexp.MustCompile(`(?m)^\s+(?:-\w, )?(--[a-z][a-z0-9-]*)`)
	helpCmdRE  = regexp.MustCompile(`(?m)^\s{2}([a-z][a-z-]*)\s{2,}\S`)
	schemeJSON = regexp.MustCompile(`"scheme":\s*"([a-z][a-z0-9_]*)"`)
)

// OpenBinary reads the subcommand list, each subcommand's flags, and the
// adapter schemes from `kno doctor --json` — all of which contact nothing.
func OpenBinary(path string) (*Binary, error) {
	b := &Binary{
		Path:        path,
		subcommands: map[string]bool{},
		flags:       map[string]map[string]bool{},
		schemes:     map[string]bool{},
	}
	root, err := b.help()
	if err != nil {
		return nil, err
	}
	for _, c := range availableCommands(root) {
		b.subcommands[c] = true
	}
	if len(b.subcommands) == 0 {
		return nil, fmt.Errorf("%s printed no subcommand list; refusing to check flags against a binary we cannot read", path)
	}
	// Every root subcommand, then every child of one. `kno eval inspect` is
	// the first two-level command in the surface and its flags live on the
	// child: `kno eval --help` lists only `-h`. Discovering children here —
	// from the binary, the same way the root list is discovered — is what
	// keeps the checker from reporting `--evals` as removed. A hard-coded
	// list of nested commands would be a second copy of the truth, and this
	// repository exists because second copies drift.
	for sub := range b.subcommands {
		if err := b.readFlags(sub); err != nil {
			return nil, err
		}
	}
	for _, sub := range b.rootSubcommands() {
		out, err := b.help(sub)
		if err != nil {
			return nil, err
		}
		for _, child := range availableCommands(out) {
			path := sub + " " + child
			b.subcommands[path] = true
			if err := b.readFlags(sub, child); err != nil {
				return nil, err
			}
		}
	}
	doctor, err := run(path, "doctor", "--json")
	if err != nil {
		return nil, fmt.Errorf("kno doctor --json: %w", err)
	}
	for _, m := range schemeJSON.FindAllStringSubmatch(doctor, -1) {
		b.schemes[m[1]] = true
	}
	return b, nil
}

// Version returns the binary's self-reported version, which is what CI writes
// into `verified-against:`.
func (b *Binary) Version() (string, error) {
	out, err := run(b.Path, "--version")
	if err != nil {
		return "", err
	}
	fields := strings.Fields(out)
	for i, f := range fields {
		if f == "version" && i+1 < len(fields) {
			return "kno v" + fields[i+1], nil
		}
	}
	return "", fmt.Errorf("could not read a version out of %q", strings.TrimSpace(out))
}

// Schemes returns every adapter scheme the binary reports.
func (b *Binary) Schemes() []string {
	out := make([]string, 0, len(b.schemes))
	for s := range b.schemes {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// rootSubcommands is the one-word command list, taken before children are
// added to the same map.
func (b *Binary) rootSubcommands() []string {
	out := make([]string, 0, len(b.subcommands))
	for s := range b.subcommands {
		if !strings.Contains(s, " ") {
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

// readFlags records the long flags of one command path, keyed by the path as a
// reader writes it ("eval inspect").
func (b *Binary) readFlags(path ...string) error {
	out, err := b.help(path...)
	if err != nil {
		return err
	}
	set := map[string]bool{}
	for _, m := range helpFlagRE.FindAllStringSubmatch(out, -1) {
		set[m[1]] = true
	}
	b.flags[strings.Join(path, " ")] = set
	return nil
}

// availableCommands parses a help page's "Available Commands:" section. It is
// the same shape at every level, so the root list and a child list are read by
// one function rather than two that can disagree.
func availableCommands(help string) []string {
	var out []string
	in := false
	for _, line := range strings.Split(help, "\n") {
		if strings.HasPrefix(line, "Available Commands:") {
			in = true
			continue
		}
		if in && strings.TrimSpace(line) == "" {
			break
		}
		if !in {
			continue
		}
		if m := helpCmdRE.FindStringSubmatch(line); m != nil {
			out = append(out, m[1])
		}
	}
	return out
}

func (b *Binary) help(args ...string) (string, error) {
	return run(b.Path, append(args, "--help")...) //nolint:gocritic // appendAssign is intended: args is a fresh slice per call
}

func run(path string, args ...string) (string, error) {
	cmd := exec.Command(path, args...) //nolint:gosec // path is an installed binary, args are literals
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%s %s: %w: %s", path, strings.Join(args, " "), err, out)
	}
	return string(out), nil
}

// FlagCheck validates every kno invocation in a recipe against the binary's
// own surface.
//
// This runs on EVERY tier, not only on `flags-only`. A flag that no longer
// exists is rot whether or not CI is allowed to execute the line, and the tier
// is a claim about the page, not about which lines get parsed.
func FlagCheck(r *Recipe, b *Binary) []Finding {
	var out []Finding
	add := func(format string, args ...any) {
		out = append(out, Finding{Path: r.Path, Message: fmt.Sprintf(format, args...)})
	}
	for _, inv := range Invocations(r.Body) {
		// Longest path first: `kno eval inspect` is checked against the child
		// when the child exists, and against `kno eval` when it does not, so
		// a word that is merely the next argument cannot invent a command.
		path := ""
		for i := len(inv.Words); i > 0; i-- {
			p := strings.Join(inv.Words[:i], " ")
			if b.subcommands[p] {
				path = p
				break
			}
		}
		if path == "" {
			add("line %d: `kno %s` is not a subcommand of this build", inv.Line, inv.Subcommand)
			continue
		}
		for _, f := range inv.Flags {
			if !b.flags[path][f] {
				add("line %d: `kno %s` has no %s flag in this build (renamed or removed)", inv.Line, path, f)
			}
		}
	}
	// Only `--agent` schemes: `doctor --json` enumerates agent adapters and
	// nothing else, so asserting an Evals or Pool scheme against it would
	// manufacture a finding about a working adapter. See AgentSchemes.
	for _, s := range AgentSchemes(r.Body) {
		if _, isNonAdapter := nonAdapterSchemes[s]; isNonAdapter {
			continue
		}
		if !b.schemes[s] {
			add("scheme %q is not reported by `kno doctor --json` in this build", s)
		}
	}
	return out
}

// nonAdapterSchemes are prefixes that look like a scheme in a `--evals` or
// `--pool` value but are file-format selectors rather than adapters, so
// `doctor` does not list them.
var nonAdapterSchemes = map[string]struct{}{
	"csv": {}, "md": {}, "http": {}, "https": {},
}
