// Package drift holds the test for the nightly workflow's issue lifecycle.
//
// The lifecycle itself is POSIX sh, in .github/scripts/issue-lifecycle.sh,
// because it is the mechanism uknoAI/kno's pricing-check.yml already proves in
// production and reinventing it in Go would be a second implementation of a
// working thing. What is NOT acceptable is shipping it unproven: a lifecycle
// bug files a duplicate issue every night until someone mutes the label, and
// by then the detector is decoration.
//
// So the script decides and prints, `gh` acts, and this test drives the
// deciding half against a mocked issue list — no network, no token, no
// repository.
package drift

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const script = "../../.github/scripts/issue-lifecycle.sh"

func TestIssueLifecycle(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		report string
		issues string // <number>\t<normalized line>
		want   []string
	}{
		{
			name:   "a finding no open issue carries is filed exactly once",
			report: "FAIL recipes/zendesk.md: line 50: `kno baseline` has no --max-cost-usd flag in this build\n",
			issues: "",
			want:   []string{"CREATE FAIL recipes/zendesk.md: line 50: `kno baseline` has no --max-cost-usd flag in this build"},
		},
		{
			name:   "the same finding on a second run edits rather than duplicates",
			report: "FAIL recipes/zendesk.md: line 50: `kno baseline` has no --max-cost-usd flag in this build\n",
			issues: "41\tFAIL recipes/zendesk.md: line N: `kno baseline` has no --max-cost-usd flag in this build\n",
			want:   []string{"EDIT 41"},
		},
		{
			name:   "a line number that moved is the same finding, not a new one",
			report: "FAIL recipes/zendesk.md: line 63: `kno baseline` has no --max-cost-usd flag in this build\n",
			issues: "41\tFAIL recipes/zendesk.md: line N: `kno baseline` has no --max-cost-usd flag in this build\n",
			want:   []string{"EDIT 41"},
		},
		{
			name:   "a clean run closes every open issue",
			report: "",
			issues: "41\tFAIL recipes/zendesk.md: line N: `kno baseline` has no --max-cost-usd flag in this build\n42\tFAIL support-refunds/baseline.score: expected N, got N\n",
			want:   []string{"CLOSE 41", "CLOSE 42"},
		},
		{
			name:   "an issue whose finding is gone closes while the surviving one is edited",
			report: "FAIL support-refunds/baseline.score: expected 1, got 0\n",
			issues: "41\tFAIL recipes/zendesk.md: line N: `kno baseline` has no --max-cost-usd flag in this build\n42\tFAIL support-refunds/baseline.score: expected N, got N\n",
			want:   []string{"CLOSE 41", "EDIT 42"},
		},
		{
			name: "several new findings file at most one issue per run",
			report: "FAIL a.md: first thing\n" +
				"FAIL b.md: second thing\n" +
				"FAIL c.md: third thing\n",
			issues: "",
			want:   []string{"CREATE FAIL a.md: first thing"},
		},
		{
			name:   "output that names no finding files nothing and closes nothing new",
			report: "verify: scenario support-refunds: baseline\nsome unparseable noise\n",
			issues: "",
			want:   nil,
		},
		{
			name:   "a multi-finding issue is edited when any one of its findings survives",
			report: "FAIL b.md: second thing\n",
			issues: "41\tFAIL a.md: first thing\n41\tFAIL b.md: second thing\n",
			want:   []string{"EDIT 41"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			reportPath := filepath.Join(dir, "report")
			issuesPath := filepath.Join(dir, "issues.tsv")
			write(t, reportPath, tc.report)
			write(t, issuesPath, tc.issues)

			cmd := exec.Command("sh", script, reportPath, issuesPath) //nolint:gosec // fixed repo path
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("issue-lifecycle.sh: %v\n%s", err, out)
			}
			got := lines(string(out))
			if !equal(got, tc.want) {
				t.Errorf("actions:\n got %q\nwant %q", got, tc.want)
			}
		})
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func lines(s string) []string {
	var out []string
	for _, l := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
