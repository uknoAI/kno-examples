#!/bin/sh
#
# The `support-refunds` scenario, end to end.
#
#   sh scenarios/support-refunds/run.sh [output-dir]
#
# Six stages against one store: baseline, value, select, export, report, purge.
# The whole run uses the built-in `fake:` agent, so it contacts nothing, spends
# nothing, and needs no credential. It needs a released `kno` on PATH and
# nothing else — no environment variable, not even KNO_DB (the store is
# `kno.db` inside a directory this script creates).
#
# Plain POSIX sh, deliberately. No `gum`, no bashisms: the bytes CI runs are
# the bytes a reader runs.
#
# ── Why the commands below are inside markers ───────────────────────────────
#
# The `# >>> <stage>` / `# <<< <stage>` pairs delimit the ONE copy of each
# command. Recipes quote the marked region by name and `verify lint` asserts
# the quoted text is byte-identical to the source, so a flag can never drift
# between the script CI runs and the page a reader copies. That drift is the
# specific bug this repository exists to kill: `uknoAI/kno`'s README and
# `tapes/quickstart.tape` carried the same scenario twice and disagreed about
# whether refunds are "processed" or "issued" within five business days.
#
# ── Why the commands are run twice ──────────────────────────────────────────
#
# Once plain, into `<out>/<stage>.txt` — that is the rendered output recipes
# quote phrases from. Once with `--json`, into `<out>/<stage>.json` — that is
# what `expected/*.json` is projected from. The two passes run in two separate
# working directories so `--db kno.db` resolves to two separate stores; a
# repeated flag would work too, but relying on last-flag-wins is a bet on a
# flag parser's behaviour that nothing here needs to make.
#
# Neither pass re-types a command: both `eval` the same marked bytes, read out
# of this file. There is exactly one copy of every flag in this repository.
#
# `purge` has no `--json` (verified against kno v0.1.2), so it runs plain in
# both passes and is asserted by quotation only.

# The stage commands at the foot of this file are not dead code: `pass` reads
# them with `marked` and evals them, which shellcheck cannot see.
# shellcheck disable=SC2317
set -eu

here=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
out=${1:-$(mktemp -d)}
mkdir -p "$out"

# ── The one prerequisite, checked before anything runs ──────────────────────
#
# This script needs a released `kno` and nothing else, so the two most likely
# ways for it to fail are not having one, or having one too old for the
# subcommands below. Both used to present as near-silence: every stage's
# stderr is captured to a file (see `pass`), so a missing binary printed one
# stage heading and exited 127 with the reason sitting in a file whose path is
# only announced on success.
#
# The version is REPORTED, never checked against a floor. A minimum-version
# constant here would be a second copy of a fact `verified-against:` already
# owns, and one copy of every fact is the whole point of this repository.
kno_bin=$(command -v kno || true)
if [ -z "$kno_bin" ]; then
	cat >&2 <<-'EOF'
	error: no `kno` on PATH.

	This scenario runs against a released kno and nothing else. To get one:

	    make install-kno
	    PATH="$PWD/bin:$PATH" sh scenarios/support-refunds/run.sh

	Or put an existing kno on your PATH.
	EOF
	exit 127
fi
printf 'scenario support-refunds: using %s (%s)\n' \
	"$kno_bin" "$("$kno_bin" --version)" >&2

stages="baseline value select export report purge"

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
	cd "$dir"
	for s in $stages; do
		extra=$2
		# purge speaks no JSON.
		if [ "$s" = purge ]; then
			extra=""
		fi
		printf 'scenario support-refunds: %s (%s)\n' "$s" "$1" >&2
		rc=0
		eval "$(marked "$s") $extra" >"$out/$s.$3" 2>"$out/$s.$3.err" || rc=$?
		# `|| rc=$?` rather than `if ! eval`: under `set -e` the `!` form
		# reports the negation's status, so the real exit code is lost.
		if [ "$rc" -ne 0 ]; then
			printf 'scenario support-refunds: %s FAILED (exit %s)\n' "$s" "$rc" >&2
			# The captured stderr is the diagnosis. Replaying it here is the
			# difference between "exit 1" and "unknown command \"select\"".
			# `/./,$!d` drops leading blank lines so the replay starts at the
			# first real line.
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

printf 'scenario support-refunds: complete. Artifacts in %s\n' "$out" >&2

exit 0

# ── The stages ──────────────────────────────────────────────────────────────
#
# Everything below this line is data, not control flow: `pass` reads it with
# `marked` and evals it. The commands assume the working directory holds
# `cases.jsonl` and `pool.jsonl`, which is what `pass` arranges — and what a
# reader following the recipe has.
#
# Every source of variation is pinned. `--run-id` so the ids are not generated,
# `--seed` and `--routing-seed` so the draws repeat, `--concurrency 1` so
# completion order cannot reorder anything, `--holdout-frac` so the split is
# not a default that may move. `fake:` is deterministic by construction. Two
# runs of this script on one binary that differ are a kno bug, and
# `verify scenario --repeat 2` is the detector.
#
# Twelve Cases, and the count is load-bearing rather than arbitrary — see
# `evals/cases.jsonl`'s note in this scenario's README. Eight land in dev, four
# are held back, and the control arm reserves int(8 * 0.3) = two, which is the
# minimum an interval can be formed from. At nine Cases the reserve rounds to
# one, every interval comes back nil, and `select` would reject on
# `underpowered` instead of the honest `no-effect`.
:
# >>> baseline
kno baseline --evals cases.jsonl --agent fake: --goal exact-match \
  --holdout-frac 0.2 --seed 1 --concurrency 1 \
  --db kno.db --run-id sr-baseline --yes
# <<< baseline
# >>> value
kno value --evals cases.jsonl --pool pool.jsonl --baseline-run-id sr-baseline \
  --agent fake: --goal exact-match --seed 1 --routing-seed 1 --concurrency 1 \
  --db kno.db --run-id sr-value --yes
# <<< value
# >>> select
kno select --value-run-id sr-value --pool pool.jsonl \
  --max-context-tokens 5000 --max-training-examples 10 --max-cost-usd 1 \
  --db kno.db --run-id sr-select
# <<< select
# >>> export
kno export --select-run-id sr-select --pool pool.jsonl \
  --destination tuning_set --out tuning.jsonl \
  --db kno.db --run-id sr-export
# <<< export
# >>> report
kno report --value-run-id sr-value --select-run-id sr-select \
  --export-run-id sr-export --db kno.db
# <<< report
# >>> purge
kno purge --run-id sr-baseline --db kno.db --yes
# <<< purge
