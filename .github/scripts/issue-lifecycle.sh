#!/bin/sh
#
# Decide what to do with the `docs-drift` issues, given a run's findings and
# the open issues.
#
#   issue-lifecycle.sh <report-file> <open-issues-tsv>
#
# It DECIDES and prints; it never calls `gh`. The workflow calls `gh` from the
# actions this prints. That split is what makes the lifecycle testable without
# a network, a token, or a repository — see internal/drift/lifecycle_test.go,
# which drives it against a mocked issue list. A lifecycle proven by waiting
# for reality is a lifecycle proven after it has already filed the duplicate.
#
# Inputs:
#   report-file       the runner's stdout. A line beginning FAIL is one
#                     finding. Everything else is ignored.
#   open-issues-tsv   one line per (issue, finding-line) pair, tab-separated:
#                     <number>\t<normalized finding line>. In CI this comes
#                     from `gh issue list --json number,body` through jq; in
#                     the test it is a fixture.
#
# Output, one action per line:
#   EDIT <number>     this issue still carries at least one of this run's
#                     findings; refresh its body so it never goes stale
#   CLOSE <number>    none of this issue's findings survive; close it with a
#                     dated comment. An empty run set is disjoint from
#                     everything, so close-on-green falls out of the same rule
#   CREATE <title>    at most ONE per run, for the first signature no open
#                     issue carries. The title is that finding's own line,
#                     never a generic summary — an issue titled "docs drift
#                     detected" is an issue nobody triages
#
# The signature of a finding is the line with digit runs collapsed to N, so
# "3 of 12 cases" and "4 of 12 cases" are one finding rather than two issues.
#
# The lifecycle is SET-based, and every rule falls out of set membership:
# intersect -> edit, disjoint -> close, uncovered -> create. This is
# uknoAI/kno's pricing-check.yml mechanism, deliberately not reinvented.

set -eu

report=${1:?usage: issue-lifecycle.sh <report-file> <open-issues-tsv>}
issues=${2:?usage: issue-lifecycle.sh <report-file> <open-issues-tsv>}

normalize() {
	awk '{
		norm = $0
		while (match(norm, /[0-9]+/)) {
			norm = substr(norm, 1, RSTART - 1) "N" substr(norm, RSTART + RLENGTH)
		}
		print norm "\t" $0
	}'
}

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM

# This run's signature set, in report order, deduplicated. Each signature keeps
# the first original line it came from, which becomes the title if this run
# files an issue for it.
grep '^FAIL' "$report" 2>/dev/null | normalize | awk -F'\t' '!seen[$1]++' >"$tmp/set" || true
cut -f1 "$tmp/set" >"$tmp/sigs"

# Every signature any open issue carries.
cut -f2- "$issues" 2>/dev/null | sort -u >"$tmp/covered" || : >"$tmp/covered"

# Edit the issues that intersect this run; close the ones that are disjoint.
# A just-edited issue is never closed in the same pass, because the lines that
# made it intersect are exactly the lines being checked.
cut -f1 "$issues" 2>/dev/null | sort -u -n >"$tmp/numbers" || : >"$tmp/numbers"
while read -r n; do
	[ -n "$n" ] || continue
	awk -F'\t' -v n="$n" '$1 == n {print $2}' "$issues" >"$tmp/lines"
	if [ -s "$tmp/sigs" ] && [ -s "$tmp/lines" ] && grep -qxFf "$tmp/sigs" "$tmp/lines"; then
		echo "EDIT $n"
	else
		echo "CLOSE $n"
	fi
done <"$tmp/numbers"

# At most one new issue per run: the first signature no open issue carries.
# The rest are covered on the next run, which is a deliberate rate limit — a
# detector that files nine issues on its first bad night is a detector whose
# label gets muted.
if [ -s "$tmp/sigs" ]; then
	uncovered=$(grep -vxF -f "$tmp/covered" "$tmp/sigs" 2>/dev/null | head -n 1 || true)
	if [ -n "${uncovered:-}" ]; then
		awk -F'\t' -v s="$uncovered" '$1 == s {print "CREATE " $2; exit}' "$tmp/set"
	fi
fi
