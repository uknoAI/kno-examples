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
	inCommands := false
	for _, line := range strings.Split(root, "\n") {
		if strings.HasPrefix(line, "Available Commands:") {
			inCommands = true
			continue
		}
		if inCommands && strings.TrimSpace(line) == "" {
			inCommands = false
			continue
		}
		if !inCommands {
			continue
		}
		if m := helpCmdRE.FindStringSubmatch(line); m != nil {
			b.subcommands[m[1]] = true
		}
	}
	if len(b.subcommands) == 0 {
		return nil, fmt.Errorf("%s printed no subcommand list; refusing to check flags against a binary we cannot read", path)
	}
	for sub := range b.subcommands {
		out, err := b.help(sub)
		if err != nil {
			return nil, err
		}
		set := map[string]bool{}
		for _, m := range helpFlagRE.FindAllStringSubmatch(out, -1) {
			set[m[1]] = true
		}
		b.flags[sub] = set
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
		if !b.subcommands[inv.Subcommand] {
			add("line %d: `kno %s` is not a subcommand of this build", inv.Line, inv.Subcommand)
			continue
		}
		for _, f := range inv.Flags {
			if !b.flags[inv.Subcommand][f] {
				add("line %d: `kno %s` has no %s flag in this build (renamed or removed)", inv.Line, inv.Subcommand, f)
			}
		}
	}
	for _, s := range Schemes(r.Body) {
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
