package recipe

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// LinkCheck asserts that every relative markdown link in the tree points at a
// file that exists.
//
// This repository's founding complaint is that no CI job had ever run a
// command from the documentation. A link between two pages is the same class
// of claim — "the thing I am pointing at is there" — and until this check
// existed nothing verified it either. `make docs` in `uknoAI/kno` skips
// `https://` targets and the website's crawl skips external hrefs, so a
// relative link that stopped resolving would have gone unnoticed in both
// repositories at once.
//
// Only RELATIVE targets are checked. An `https://` link is a claim about
// somebody else's server, and a nightly job that reddens on a third party's
// outage is a signal people learn to override — the same reasoning that keeps
// the vendor recipes at `flags-only`.
//
// The fragment is stripped and not verified: a heading anchor is a claim about
// a renderer's slug algorithm rather than about this repository, and asserting
// one would make the check depend on which markdown renderer is reading the
// page.
func LinkCheck(root string) ([]Finding, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "bin", "node_modules":
				return fs.SkipDir
			case "testdata":
				// The deliberately broken corpus lives here, and one of its
				// pages is broken in exactly this way ON PURPOSE. Walking into
				// it would make `make check` red on a fixture whose job is to
				// be red — the same reason the recipe lints load `recipes/`
				// rather than every markdown file they can find.
				//
				// A test points the walk AT one of those directories to prove
				// the check still fires; the skip is on a directory named
				// `testdata` encountered inside a walk, never on the root the
				// caller named.
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".md") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", root, err)
	}
	sort.Strings(files)

	var out []Finding
	for _, f := range files {
		b, err := os.ReadFile(f) //nolint:gosec // a repo path from the walk
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", f, err)
		}
		out = append(out, linkFindings(f, string(b))...)
	}
	return out, nil
}

var (
	markdownLinkRE = regexp.MustCompile(`\[[^\]]*\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)
	fencedBlockRE  = regexp.MustCompile("(?s)```.*?```")
)

func linkFindings(path, src string) []Finding {
	// Fenced blocks are stripped first: a code sample may legitimately contain
	// bracket-paren text that is not a link, and a finding about a line
	// nobody can click is noise.
	body := fencedBlockRE.ReplaceAllStringFunc(src, func(block string) string {
		return strings.Repeat("\n", strings.Count(block, "\n"))
	})
	dir := filepath.Dir(path)
	var out []Finding
	for _, m := range markdownLinkRE.FindAllStringSubmatchIndex(body, -1) {
		target := body[m[2]:m[3]]
		if skipTarget(target) {
			continue
		}
		file, _, _ := strings.Cut(target, "#")
		if file == "" {
			continue // a bare fragment, which is same-page and not our claim
		}
		resolved := filepath.Join(dir, file)
		if _, err := os.Stat(resolved); err != nil {
			line := 1 + strings.Count(body[:m[0]], "\n")
			out = append(out, Finding{
				Path: path,
				Message: fmt.Sprintf(
					"line %d: the link target %q does not exist (resolves to %s) — a link to a file that is not there is a claim that failed",
					line, target, resolved),
			})
		}
	}
	return out
}

func skipTarget(target string) bool {
	for _, prefix := range []string{"http://", "https://", "mailto:", "#", "/"} {
		if strings.HasPrefix(target, prefix) {
			return true
		}
	}
	// An absolute-looking target with a scheme we do not know is somebody
	// else's, not a path in this tree.
	if i := strings.Index(target, ":"); i > 0 && !strings.ContainsAny(target[:i], "/.") {
		return true
	}
	return false
}
