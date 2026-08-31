#!/bin/sh
#
# The `underpowered-eval` scenario, end to end.
#
#   sh scenarios/underpowered-eval/run.sh [output-dir]
#
# Three stages against one store: baseline, value, select. There is no export,
# report, or purge here — `select` recommends nothing, so there is nothing
# downstream to export, and the stages that would follow are already asserted
# by `support-refunds`.
#
# The whole run uses the built-in `fake:` agent, so it contacts nothing, spends
# nothing, and needs no credential. It needs a released `kno` on PATH and
# nothing else.
#
# ── Why this scenario exists ────────────────────────────────────────────────
#
# `support-refunds` ends in `no-effect`: twelve Cases, an interval formed, and
# the honest report that injecting each Asset moved nothing. This scenario ends
# in `underpowered`: nine Cases, no interval formable, and the honest report
# that we could not tell.
#
# Those two verdicts look alike from a distance — both reject every Asset — and
# they mean opposite things. One is a measurement. The other is the refusal to
# pretend there was one. A reader who cannot see the difference will read
# "rejected" as "measured and found wanting" every time, which is the mistake
# this pair exists to make impossible.
#
# The ONLY difference between the two scenarios is the number of Cases. Same
# Pool, same Goal, same seeds, same flags. A test asserts these nine Cases are
# the first nine of `support-refunds`, so the pair stays a controlled
# comparison rather than two unrelated runs that happen to disagree.
#
# ── Why nine ────────────────────────────────────────────────────────────────
#
# With `--holdout-frac 0.2`, nine Cases land as six in dev and three held back
# (`expected/baseline.json` asserts both numbers). The control arm then reserves
# int(6 * 0.3) = one, and one is not enough to form an interval from: every
# interval comes back nil and `select` refuses on `underpowered`.
#
# The boundary is measured, not assumed. Against kno v0.1.3:
#
#     cases   dev/holdout   select
#         8         6 / 2   underpowered
#         9         6 / 3   underpowered   <- this scenario
#        10         7 / 3   no-effect
#        12         8 / 4   no-effect      <- support-refunds
#
# Ten is the smallest count that clears it, because int(7 * 0.3) = two. Nine is
# the LARGEST that fails, which is deliberately where this scenario sits: one
# Case away from the boundary is where a change in the reserve arithmetic shows
# up first.
#
# Those figures are load-bearing. If a future kno forms an interval at nine
# Cases, this run goes red and the numbers in this comment are what need
# revisiting.
#
# Plain POSIX sh, deliberately. No `gum`, no bashisms.
#
# The `# >>> <stage>` / `# <<< <stage>` pairs delimit the ONE copy of each
# command; recipes quote the marked region by name and `verify lint` asserts
# byte-identity.

# The stage commands at the foot of this file are not dead code: `pass` reads
# them with `marked` and evals them, which shellcheck cannot see.
# shellcheck disable=SC2317
set -eu

here=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
out=${1:-$(mktemp -d)}
mkdir -p "$out"

# The one prerequisite, checked before anything runs — see
# scenarios/support-refunds/run.sh for why this is not left to the shell.
kno_bin=$(command -v kno || true)
if [ -z "$kno_bin" ]; then
	cat >&2 <<-'EOF'
	error: no `kno` on PATH.

	This scenario runs against a released kno and nothing else. To get one:

	    make install-kno
	    PATH="$PWD/bin:$PATH" sh scenarios/underpowered-eval/run.sh

	Or put an existing kno on your PATH.
	EOF
	exit 127
fi
printf 'scenario underpowered-eval: using %s (%s)\n' \
	"$kno_bin" "$("$kno_bin" --version)" >&2

stages="baseline value select"

# The marked region for one stage, verbatim, with the marker lines stripped.
marked() {
	sed -n "/^# >>> $1\$/,/^# <<< $1\$/p" "$here/run.sh" | sed '1d;$d'
}

pass() {
	dir="$out/$1"
	mkdir -p "$dir"
	cp "$here/evals/cases.jsonl" "$dir/cases.jsonl"
	cp "$here/pool/pool.jsonl" "$dir/pool.jsonl"
	cd "$dir"
	for s in $stages; do
		extra=$2
		printf 'scenario underpowered-eval: %s (%s)\n' "$s" "$1" >&2
		rc=0
		eval "$(marked "$s") $extra" >"$out/$s.$3" 2>"$out/$s.$3.err" || rc=$?
		if [ "$rc" -ne 0 ]; then
			printf 'scenario underpowered-eval: %s FAILED (exit %s)\n' "$s" "$rc" >&2
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

printf 'scenario underpowered-eval: complete. Artifacts in %s\n' "$out" >&2

exit 0

# ── The stages ──────────────────────────────────────────────────────────────
#
# Every source of variation is pinned, exactly as in `support-refunds`: same
# seeds, same concurrency, same holdout fraction, same Goal. The Case count is
# the only thing that differs, which is what makes the two verdicts a
# comparison rather than a coincidence.
:
# >>> baseline
kno baseline --evals cases.jsonl --agent fake: --goal exact-match \
  --holdout-frac 0.2 --seed 1 --concurrency 1 \
  --db kno.db --run-id ue-baseline --yes
# <<< baseline
# >>> value
kno value --evals cases.jsonl --pool pool.jsonl --baseline-run-id ue-baseline \
  --agent fake: --goal exact-match --seed 1 --routing-seed 1 --concurrency 1 \
  --db kno.db --run-id ue-value --yes
# <<< value
# >>> select
kno select --value-run-id ue-value --pool pool.jsonl \
  --max-context-tokens 5000 --max-training-examples 10 --max-cost-usd 1 \
  --db kno.db --run-id ue-select
# <<< select
