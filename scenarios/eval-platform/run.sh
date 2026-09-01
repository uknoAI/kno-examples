#!/bin/sh
#
# The `eval-platform` scenario, end to end.
#
#   sh scenarios/eval-platform/run.sh [output-dir]
#
# Four stages against one store: baseline, value, select, export.
#
# The whole run uses the built-in `fake:` agent, so it contacts nothing, spends
# nothing, and needs no credential. It needs a released `kno` on PATH and
# nothing else.
#
# ── Why this scenario exists ────────────────────────────────────────────────
#
# The Cases here are not questions to an agent. Each input is ANOTHER model's
# answer, and the expected output is a grade. This is an LLM-as-judge, which is
# the shape an eval platform's dataset actually has — a Braintrust dataset row,
# a Langfuse trace scored after the fact — and it is the shape four vendor
# recipes here describe (`braintrust`, `langfuse`, `langsmith`, `huggingface`)
# while deferring to a customer-support recipe for their flow.
#
# The Pool is MIXED, which nothing else here is: two `knowledge` Assets (the
# rubric the judge applies, and the label vocabulary it must write in) and one
# `behavior` Asset (how to grade — the disposition, not the facts). Kno routes
# the two kinds to different destinations, so a Pool that carries both is the
# case where routing has to actually decide rather than send everything one
# way.
#
# ── What it does NOT show ───────────────────────────────────────────────────
#
# The same caveat as every scenario here: `fake:` answers each Case with what
# the Case expects, so no Asset can move the score and the Portfolio comes back
# empty. See the roadmap in the repository README for why that is not fixable
# offline. What is asserted is the routing and the arithmetic, not a win.
#
# Plain POSIX sh, deliberately. The `# >>> <stage>` / `# <<< <stage>` pairs
# delimit the ONE copy of each command; `verify lint` asserts any recipe
# quoting them is byte-identical.

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
	    PATH="$PWD/bin:$PATH" sh scenarios/eval-platform/run.sh

	Or put an existing kno on your PATH.
	EOF
	exit 127
fi
printf 'scenario eval-platform: using %s (%s)\n' \
	"$kno_bin" "$("$kno_bin" --version)" >&2

stages="baseline value select export"

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
		printf 'scenario eval-platform: %s (%s)\n' "$s" "$1" >&2
		rc=0
		eval "$(marked "$s") $extra" >"$out/$s.$3" 2>"$out/$s.$3.err" || rc=$?
		if [ "$rc" -ne 0 ]; then
			printf 'scenario eval-platform: %s FAILED (exit %s)\n' "$s" "$rc" >&2
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

printf 'scenario eval-platform: complete. Artifacts in %s\n' "$out" >&2

exit 0

# ── The stages ──────────────────────────────────────────────────────────────
#
# Every source of variation is pinned, exactly as in `support-refunds`.
#
# Twelve Cases, above the boundary `underpowered-eval` sits below, so this
# scenario reaches a real `no-effect` rather than an honest refusal to measure.
#
# `--destination context` rather than `tuning_set`: this Pool is mostly
# knowledge, and exporting it as tuning data would be asserting the wrong
# routing. `coding-agent` exports the tuning side; between them the two
# destinations are both covered.
:
# >>> baseline
kno baseline --evals cases.jsonl --agent fake: --goal exact-match \
  --holdout-frac 0.2 --seed 1 --concurrency 1 \
  --db kno.db --run-id ep-baseline --yes
# <<< baseline
# >>> value
kno value --evals cases.jsonl --pool pool.jsonl --baseline-run-id ep-baseline \
  --agent fake: --goal exact-match --seed 1 --routing-seed 1 --concurrency 1 \
  --db kno.db --run-id ep-value --yes
# <<< value
# >>> select
kno select --value-run-id ep-value --pool pool.jsonl \
  --max-context-tokens 5000 --max-training-examples 10 --max-cost-usd 1 \
  --db kno.db --run-id ep-select
# <<< select
# >>> export
kno export --select-run-id ep-select --pool pool.jsonl \
  --destination context --out context.jsonl \
  --db kno.db --run-id ep-export
# <<< export
