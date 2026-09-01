#!/bin/sh
#
# The `judge-calibration` scenario, end to end.
#
#   sh scenarios/judge-calibration/run.sh [output-dir]
#
# Seven stages in two halves. The first four calibrate the GOAL — measure how
# well it agrees with human labels, print the artifact that makes a prompt edit
# a directed act, and show the gate refusing on two different grounds. The last
# three run the ordinary loop with the goal that was just calibrated.
#
# The order is the argument. `kno judge calibrate` exists because a judged
# number is only as good as the judge that produced it, and judge error does
# not average out — it shrinks every effect you measure toward zero. So the
# calibration is not an appendix to the loop; it is the thing you do before you
# are entitled to read the loop's output. The scenario runs in that order for
# the same reason the recipe is written in it.
#
# Everything is free and offline. `kno judge calibrate --replay` is the default
# and makes no provider call, its calibration set is built into the binary, and
# the loop stages run against `fake:`. No credential, no network.
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
	    PATH="$PWD/bin:$PATH" sh scenarios/judge-calibration/run.sh

	Or put an existing kno on your PATH.
	EOF
	exit 127
fi
printf 'scenario judge-calibration: using %s (%s)\n' \
	"$kno_bin" "$("$kno_bin" --version)" >&2

stages="calibrate disagreements straddle ceiling baseline value select"

# The stages that read no store an earlier stage wrote. The four calibration
# stages read the set built into the binary; they open no database and depend
# on nothing that ran before them. `verify lint` reads this line so a recipe
# may open on any of them without declaring `requires-stages:` — which would
# generate a sentence that is false.
#
# shellcheck disable=SC2034 # read by `verify lint`, not by this script
independent_stages="calibrate disagreements straddle ceiling"

# The stages whose whole point is that they FAIL, and the code they must fail
# with. `kno judge calibrate` exits 1 on FAIL and on INDETERMINATE, and both
# are demonstrated here.
#
# Asserting the exit code in both directions matters more than it looks. A gate
# that silently stopped refusing would leave every stage green, every number
# unchanged, and the scenario would go on "passing" while demonstrating the
# opposite of what its README claims. So a 0 from one of these is a failure of
# the scenario, in the same words as any other.
expect_exit_1="straddle ceiling"

# The marked region for one stage, verbatim, with the marker lines stripped.
marked() {
	sed -n "/^# >>> $1\$/,/^# <<< $1\$/p" "$here/run.sh" | sed '1d;$d'
}

# Whether $1 appears in the space-separated list $2.
in_list() {
	for item in $2; do
		[ "$item" = "$1" ] && return 0
	done
	return 1
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
		printf 'scenario judge-calibration: %s (%s)\n' "$s" "$1" >&2
		want=0
		if in_list "$s" "$expect_exit_1"; then
			want=1
		fi
		rc=0
		eval "$(marked "$s") $2" >"$out/$s.$3" 2>"$out/$s.$3.err" || rc=$?
		# `|| rc=$?` rather than `if ! eval`: under `set -e` the `!` form
		# reports the negation's status, so the real exit code is lost.
		if [ "$rc" -ne "$want" ]; then
			if [ "$want" -ne 0 ]; then
				printf 'scenario judge-calibration: %s exited %s but MUST exit %s — the gate stopped refusing\n' \
					"$s" "$rc" "$want" >&2
			else
				printf 'scenario judge-calibration: %s FAILED (exit %s)\n' "$s" "$rc" >&2
			fi
			if [ -s "$out/$s.$3.err" ]; then
				sed -e '/./,$!d' -e 's/^/    /' "$out/$s.$3.err" >&2
			else
				printf '    (the stage wrote nothing to stderr)\n' >&2
			fi
			printf 'artifacts in %s\n' "$out" >&2
			exit 1
		fi
	done
	cd "$here"
}

pass json "--json" json
pass text "" txt

printf 'scenario judge-calibration: complete. Artifacts in %s\n' "$out" >&2

exit 0

# ── The stages ──────────────────────────────────────────────────────────────
#
# Everything below this line is data, not control flow: `pass` reads it with
# `marked` and evals it. The commands assume the working directory holds
# `cases.jsonl` and `pool.jsonl`, which is what `pass` arranges; the
# calibration stages need neither.
#
# `--set-name starter` and `--min-kappa` are stated rather than left to their
# defaults. The set name decides which records are scored and the floor decides
# the verdict, so a run whose defaults moved would change the committed numbers
# without any command here changing.
:
# >>> calibrate
kno judge calibrate --replay --goal exact-match --set-name starter --min-kappa 0.60
# <<< calibrate
# >>> disagreements
kno judge calibrate --replay --goal exact-match --set-name starter --min-kappa 0.60 \
  --show-disagreements
# <<< disagreements
# >>> straddle
kno judge calibrate --replay --goal exact-match --set-name starter --min-kappa 0.88
# <<< straddle
# >>> ceiling
kno judge calibrate --replay --goal exact-match --set-name starter --min-kappa 0.95
# <<< ceiling
# >>> baseline
kno baseline --evals cases.jsonl --agent fake: --goal exact-match \
  --holdout-frac 0.2 --seed 1 --concurrency 1 \
  --db kno.db --run-id jc-baseline --yes
# <<< baseline
# >>> value
kno value --evals cases.jsonl --pool pool.jsonl --baseline-run-id jc-baseline \
  --agent fake: --goal exact-match --seed 1 --routing-seed 1 --concurrency 1 \
  --db kno.db --run-id jc-value --yes
# <<< value
# >>> select
kno select --value-run-id jc-value --pool pool.jsonl \
  --max-context-tokens 5000 --max-training-examples 10 --max-cost-usd 1 \
  --db kno.db --run-id jc-select
# <<< select
