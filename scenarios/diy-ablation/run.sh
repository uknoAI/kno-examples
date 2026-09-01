#!/bin/sh
#
# The `diy-ablation` scenario, end to end.
#
#   sh scenarios/diy-ablation/run.sh [output-dir]
#
# Four stages against one eval set. The first is not a `kno` command at all: it
# is `naive_ablation.py`, the hundred-line context ablation an engineer writes
# in an afternoon, run over exactly the Cases and Assets the three `kno` stages
# that follow are run over. Same agent, same scorer, same data — so every
# difference between the first stage's output and the last's is a difference of
# METHOD and nothing else.
#
# That is the point of the scenario, and it is why the script is executed here
# rather than described in prose. The commonest unanswered objection to a
# measurement tool is "I could script this myself", and the honest reply is not
# an argument. It is the script, running, with its output committed next to the
# tool's.
#
# Everything runs against the built-in `fake:` agent: it contacts nothing,
# spends nothing, and needs no credential.
#
# It needs a released `kno` and a `python3` on PATH, and nothing else.
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

# The stage commands at the foot of this file are not dead code: `pass` reads
# them with `marked` and evals them, which shellcheck cannot see.
# shellcheck disable=SC2317
set -eu

here=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
out=${1:-$(mktemp -d)}
mkdir -p "$out"

# ── The two prerequisites, checked before anything runs ─────────────────────
#
# Versions are REPORTED, never checked against a floor. A minimum-version
# constant here would be a second copy of a fact `verified-against:` already
# owns, and one copy of every fact is the whole point of this repository.
kno_bin=$(command -v kno || true)
if [ -z "$kno_bin" ]; then
	cat >&2 <<-'EOF'
	error: no `kno` on PATH.

	This scenario runs against a released kno. To get one:

	    make install-kno
	    PATH="$PWD/bin:$PATH" sh scenarios/diy-ablation/run.sh

	Or put an existing kno on your PATH.
	EOF
	exit 127
fi
py_bin=$(command -v python3 || true)
if [ -z "$py_bin" ]; then
	cat >&2 <<-'EOF'
	error: no `python3` on PATH.

	This is the one scenario here that needs one. Its first stage runs
	`naive_ablation.py`, the hand-rolled ablation the rest of the scenario is
	the answer to, and a scenario that skipped it would be an argument about an
	alternative it never ran.

	Any python3 will do; the script imports only the standard library.
	EOF
	exit 127
fi
printf 'scenario diy-ablation: using %s (%s) and %s (%s)\n' \
	"$kno_bin" "$("$kno_bin" --version)" \
	"$py_bin" "$("$py_bin" --version 2>&1)" >&2

stages="naive baseline value select"

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
	# The script is copied in beside the data rather than run from `$here`, so
	# the command in the marked region is the command a reader runs after
	# copying the script next to their own Cases.
	cp "$here/naive_ablation.py" "$dir/naive_ablation.py"
	cd "$dir"
	for s in $stages; do
		printf 'scenario diy-ablation: %s (%s)\n' "$s" "$1" >&2
		rc=0
		eval "$(marked "$s") $2" >"$out/$s.$3" 2>"$out/$s.$3.err" || rc=$?
		# `|| rc=$?` rather than `if ! eval`: under `set -e` the `!` form
		# reports the negation's status, so the real exit code is lost.
		if [ "$rc" -ne 0 ]; then
			printf 'scenario diy-ablation: %s FAILED (exit %s)\n' "$s" "$rc" >&2
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

printf 'scenario diy-ablation: complete. Artifacts in %s\n' "$out" >&2

exit 0

# ── The stages ──────────────────────────────────────────────────────────────
#
# Everything below this line is data, not control flow: `pass` reads it with
# `marked` and evals it. The commands assume the working directory holds
# `cases.jsonl`, `pool.jsonl` and `naive_ablation.py`, which is what `pass`
# arranges.
#
# The naive stage takes `--evals` and `--pool` with the same spellings kno
# uses, on purpose: the two commands must differ in what they DO and not in how
# they are typed, or a reader will attribute the difference in output to the
# interface.
:
# >>> naive
python3 naive_ablation.py --evals cases.jsonl --pool pool.jsonl
# <<< naive
# >>> baseline
kno baseline --evals cases.jsonl --agent fake: --goal exact-match \
  --holdout-frac 0.2 --seed 1 --concurrency 1 \
  --db kno.db --run-id diy-baseline --yes
# <<< baseline
# >>> value
kno value --evals cases.jsonl --pool pool.jsonl --baseline-run-id diy-baseline \
  --agent fake: --goal exact-match --seed 1 --routing-seed 1 --concurrency 1 \
  --db kno.db --run-id diy-value --yes
# <<< value
# >>> select
kno select --value-run-id diy-value --pool pool.jsonl \
  --max-context-tokens 5000 --max-training-examples 10 --max-cost-usd 1 \
  --db kno.db --run-id diy-select
# <<< select
