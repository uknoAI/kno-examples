#!/bin/sh
#
# The `transcript-mining` scenario, end to end.
#
#   sh scenarios/transcript-mining/run.sh [output-dir]
#
# Five stages, starting one step earlier than every other scenario here: with
# transcripts rather than with an eval set. `kno mine` turns a directory of
# support logs into Cases, and the four stages after it are the ordinary loop
# run over what came out.
#
# The question it answers is the data engineer's, and no feature list answers
# it: WHERE DOES THIS SIT IN MY PIPELINE, and DOES IT READ WHAT I ALREADY HAVE.
# The `transcripts/` directory holds two formats — a JSONL chat export and a
# CSV ticket export — and one `kno mine` reads both, because `--format auto`
# sniffs each file. That is the shape of a real export directory.
#
# Everything runs against the built-in `fake:` agent, and `kno mine` and
# `kno eval inspect` construct no agent at all: nothing here contacts anything,
# spends anything, or needs a credential.
#
# It needs a released `kno` on PATH and nothing else.
#
# Plain POSIX sh, deliberately. No `gum`, no bashisms: the bytes CI runs are
# the bytes a reader runs.
#
# ── Why the commands below are inside markers ───────────────────────────────
#
# The `# >>> <stage>` / `# <<< <stage>` pairs delimit the ONE copy of each
# command. Recipes quote the marked region by name and `verify lint` asserts
# the quoted text is byte-identical to the source.
#
# ── Why the stages after `mine` read `mined.jsonl` ──────────────────────────
#
# `evals/cases.jsonl` is committed, and it is the OUTPUT of the mine stage
# rather than an input to it. The stages downstream deliberately read
# `mined.jsonl` — the file this run just produced — rather than the committed
# copy, so the scenario exercises the real pipeline instead of a pre-baked
# eval set that happens to sit next to it.
#
# The committed copy earns its place a different way: `internal/scenario`
# requires every scenario to carry an `evals/cases.jsonl`, and a test asserts
# that file is byte-identical to what `kno mine` writes here. So it is a
# committed expectation about mining rather than a second input that could
# drift from the first.

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
	    PATH="$PWD/bin:$PATH" sh scenarios/transcript-mining/run.sh

	Or put an existing kno on your PATH.
	EOF
	exit 127
fi
printf 'scenario transcript-mining: using %s (%s)\n' \
	"$kno_bin" "$("$kno_bin" --version)" >&2

stages="mine inspect baseline value select"

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
	cp -R "$here/transcripts" "$dir/transcripts"
	cp "$here/pool/pool.jsonl" "$dir/pool.jsonl"
	cd "$dir"
	for s in $stages; do
		extra=$2
		# mine speaks no JSON (verified against kno v0.1.5), so it runs plain
		# in both passes and is asserted by quotation only.
		if [ "$s" = mine ]; then
			extra=""
		fi
		printf 'scenario transcript-mining: %s (%s)\n' "$s" "$1" >&2
		rc=0
		eval "$(marked "$s") $extra" >"$out/$s.$3" 2>"$out/$s.$3.err" || rc=$?
		# `|| rc=$?` rather than `if ! eval`: under `set -e` the `!` form
		# reports the negation's status, so the real exit code is lost.
		if [ "$rc" -ne 0 ]; then
			printf 'scenario transcript-mining: %s FAILED (exit %s)\n' "$s" "$rc" >&2
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

printf 'scenario transcript-mining: complete. Artifacts in %s\n' "$out" >&2

exit 0

# ── The stages ──────────────────────────────────────────────────────────────
#
# Everything below this line is data, not control flow: `pass` reads it with
# `marked` and evals it. The commands assume the working directory holds
# `transcripts/` and `pool.jsonl`, which is what `pass` arranges.
#
# `--mode resolution` is stated rather than left to the default, because it is
# the load-bearing choice: it says the expected answer is what CLOSED the
# thread, which is the only reading that is honest under an exact-match goal.
:
# >>> mine
kno mine --logs transcripts --out mined.jsonl --mode resolution --format auto
# <<< mine
# >>> inspect
kno eval inspect --evals mined.jsonl --holdout-frac 0.2
# <<< inspect
# >>> baseline
kno baseline --evals mined.jsonl --agent fake: --goal exact-match \
  --holdout-frac 0.2 --seed 1 --concurrency 1 \
  --db kno.db --run-id tm-baseline --yes
# <<< baseline
# >>> value
kno value --evals mined.jsonl --pool pool.jsonl --baseline-run-id tm-baseline \
  --agent fake: --goal exact-match --seed 1 --routing-seed 1 --concurrency 1 \
  --db kno.db --run-id tm-value --yes
# <<< value
# >>> select
kno select --value-run-id tm-value --pool pool.jsonl \
  --max-context-tokens 5000 --max-training-examples 10 --max-cost-usd 1 \
  --db kno.db --run-id tm-select
# <<< select
