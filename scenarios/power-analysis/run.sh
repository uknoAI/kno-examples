#!/bin/sh
#
# The `power-analysis` scenario, end to end.
#
#   sh scenarios/power-analysis/run.sh [output-dir]
#
# Seven stages against one eval set at three sizes: three `kno eval inspect`
# reads at n=12, n=40 and n=160, then baseline, value and select over the whole
# 160, then one more inspect that reads what the Value run actually attributed.
#
# The question it answers is the one nobody can answer from a feature list:
# HOW MANY CASES DO I NEED. Every number below is a function of n and of the
# tags, not of any model's opinion, so the whole sweep runs against the
# built-in `fake:` agent — it contacts nothing, spends nothing, needs no
# credential, and `kno eval inspect` does not construct an agent at all.
#
# It needs a released `kno` on PATH, and `python3` only to REGENERATE the
# Cases (see evals/generate.py); running this script needs no python.
#
# Plain POSIX sh, deliberately. No `gum`, no bashisms: the bytes CI runs are
# the bytes a reader runs.
#
# ── Why the commands below are inside markers ───────────────────────────────
#
# The `# >>> <stage>` / `# <<< <stage>` pairs delimit the ONE copy of each
# command. Recipes quote the marked region by name and `verify lint` asserts
# the quoted text is byte-identical to the source, so a flag can never drift
# between the script CI runs and the page a reader copies.
#
# ── Why the smaller eval sets are prefixes ──────────────────────────────────
#
# `cases-12.jsonl` and `cases-40.jsonl` are `head -n` of the committed
# `cases.jsonl` rather than two more committed files. Two reasons, and the
# first one is the important one:
#
#   1. A prefix is provably the SAME eval set, smaller. Three separately
#      written files could disagree about a tag or an id, and then the sweep
#      would be comparing three different populations while claiming to
#      compare three sizes of one. That is the exact class of bug this
#      repository exists to kill, re-created inside a single scenario.
#   2. The dev/holdout split is keyed on the Case id, not on file position, so
#      truncating the file cannot move a surviving Case between halves. A
#      prefix is a subset in the sense that matters.
#
# The Cases are interleaved by behavior (see evals/generate.py), so `head -n
# 12` takes three of each tag rather than twelve of one.

# The stage commands at the foot of this file are not dead code: `pass` reads
# them with `marked` and evals them, which shellcheck cannot see.
# shellcheck disable=SC2317
set -eu

here=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
out=${1:-$(mktemp -d)}
mkdir -p "$out"

# ── The one prerequisite, checked before anything runs ──────────────────────
#
# Reported, never checked against a floor: a minimum-version constant here
# would be a second copy of a fact `verified-against:` already owns.
kno_bin=$(command -v kno || true)
if [ -z "$kno_bin" ]; then
	cat >&2 <<-'EOF'
	error: no `kno` on PATH.

	This scenario runs against a released kno and nothing else. To get one:

	    make install-kno
	    PATH="$PWD/bin:$PATH" sh scenarios/power-analysis/run.sh

	Or put an existing kno on your PATH.
	EOF
	exit 127
fi
printf 'scenario power-analysis: using %s (%s)\n' \
	"$kno_bin" "$("$kno_bin" --version)" >&2

stages="inspect-12 inspect-40 inspect-160 baseline value select attribute"

# The marked region for one stage, verbatim, with the marker lines stripped.
marked() {
	sed -n "/^# >>> $1\$/,/^# <<< $1\$/p" "$here/run.sh" | sed '1d;$d'
}

# One pass over every stage. $1 is the working subdirectory, $2 the flags
# appended to each command (empty for the rendered pass), $3 the extension the
# captured stdout gets.
#
# stdout and stderr are captured separately on purpose: interleaving two
# streams into one file makes the result depend on buffering, which would turn
# a committed expectation into a coin flip.
pass() {
	dir="$out/$1"
	mkdir -p "$dir"
	cp "$here/evals/cases.jsonl" "$dir/cases.jsonl"
	cp "$here/pool/pool.jsonl" "$dir/pool.jsonl"
	# The two smaller sets, as prefixes of the one committed file. See the
	# note above: this is what makes the sweep a statement about n rather than
	# about three different eval sets.
	head -n 12 "$dir/cases.jsonl" >"$dir/cases-12.jsonl"
	head -n 40 "$dir/cases.jsonl" >"$dir/cases-40.jsonl"
	cd "$dir"
	for s in $stages; do
		printf 'scenario power-analysis: %s (%s)\n' "$s" "$1" >&2
		rc=0
		eval "$(marked "$s") $2" >"$out/$s.$3" 2>"$out/$s.$3.err" || rc=$?
		# `|| rc=$?` rather than `if ! eval`: under `set -e` the `!` form
		# reports the negation's status, so the real exit code is lost.
		if [ "$rc" -ne 0 ]; then
			printf 'scenario power-analysis: %s FAILED (exit %s)\n' "$s" "$rc" >&2
			if [ -s "$out/$s.$3.err" ]; then
				sed -e '/./,$!d' -e 's/^/    /' "$out/$s.$3.err" >&2
			else
				printf '    (the stage wrote nothing to stderr)\n' >&2
			fi
			printf 'artifacts in %s\n' "$out" >&2
			exit "$rc"
		fi
	done
	cd "$here"
}

pass json "--json" json
pass text "" txt

printf 'scenario power-analysis: complete. Artifacts in %s\n' "$out" >&2

exit 0

# ── The stages ──────────────────────────────────────────────────────────────
#
# Everything below this line is data, not control flow: `pass` reads it with
# `marked` and evals it. The commands assume the working directory holds
# `cases.jsonl`, `cases-12.jsonl`, `cases-40.jsonl` and `pool.jsonl`, which is
# what `pass` arranges.
#
# `--holdout-frac 0.2` is pinned on every `inspect` even though 0.2 is the
# default, because the fraction decides which Cases are dev and therefore every
# number the stage prints. A sweep whose split could move under a changed
# default is not a sweep.
:
# >>> inspect-12
kno eval inspect --evals cases-12.jsonl --holdout-frac 0.2
# <<< inspect-12
# >>> inspect-40
kno eval inspect --evals cases-40.jsonl --holdout-frac 0.2
# <<< inspect-40
# >>> inspect-160
kno eval inspect --evals cases.jsonl --holdout-frac 0.2
# <<< inspect-160
# >>> baseline
kno baseline --evals cases.jsonl --agent fake: --goal exact-match \
  --holdout-frac 0.2 --seed 1 --concurrency 1 \
  --db kno.db --run-id pa-baseline --yes
# <<< baseline
# >>> value
kno value --evals cases.jsonl --pool pool.jsonl --baseline-run-id pa-baseline \
  --agent fake: --goal exact-match --seed 1 --routing-seed 1 --concurrency 1 \
  --db kno.db --run-id pa-value --yes
# <<< value
# >>> select
kno select --value-run-id pa-value --pool pool.jsonl \
  --max-context-tokens 5000 --max-training-examples 10 --max-cost-usd 1 \
  --db kno.db --run-id pa-select
# <<< select
# >>> attribute
kno eval inspect --evals cases.jsonl --holdout-frac 0.2 \
  --value-run-id pa-value --db kno.db
# <<< attribute
