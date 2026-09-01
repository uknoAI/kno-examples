#!/bin/sh
#
# The `coding-agent` scenario, end to end.
#
#   sh scenarios/coding-agent/run.sh [output-dir]
#
# Four stages against one store: baseline, value, select, export.
#
# The whole run uses the built-in `fake:` agent, so it contacts nothing, spends
# nothing, and needs no credential. It needs a released `kno` on PATH and
# nothing else.
#
# ── Why this scenario exists ────────────────────────────────────────────────
#
# `support-refunds` is a support agent answering customer questions, and its
# Pool is three documents — `kind: knowledge`, the sort of thing that gets
# injected into context.
#
# This is a coding agent answering questions about a codebase's conventions,
# and its Pool is three DEMONSTRATIONS — `kind: behavior`. Kno routes the two
# kinds differently and says so in the schema: knowledge is "facts, policies,
# documents ... valued by context or knowledge-base injection", while behavior
# is "format, tone, tool-use patterns, reasoning demonstrations. The only kind
# that faces the fine-tuning bridge at all."
#
# Until this scenario, nothing committed here exercised `behavior` at all. Every
# Asset in the repository was knowledge, so the routing decision that sends an
# Asset to tuning rather than context was asserted by nothing.
#
# It also gives the coding-shaped vendor recipes — `github` in particular — a
# scenario to be the same shape as. Before this, every vendor page deferred to
# the Zendesk recipe for its flow, including the ones about source code.
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
	    PATH="$PWD/bin:$PATH" sh scenarios/coding-agent/run.sh

	Or put an existing kno on your PATH.
	EOF
	exit 127
fi
printf 'scenario coding-agent: using %s (%s)\n' \
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
		printf 'scenario coding-agent: %s (%s)\n' "$s" "$1" >&2
		rc=0
		eval "$(marked "$s") $extra" >"$out/$s.$3" 2>"$out/$s.$3.err" || rc=$?
		if [ "$rc" -ne 0 ]; then
			printf 'scenario coding-agent: %s FAILED (exit %s)\n' "$s" "$rc" >&2
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

printf 'scenario coding-agent: complete. Artifacts in %s\n' "$out" >&2

exit 0

# ── The stages ──────────────────────────────────────────────────────────────
#
# Every source of variation is pinned, exactly as in `support-refunds`.
#
# Twelve Cases, which is above the boundary `underpowered-eval` sits below: the
# reserve is int(dev * 0.3) and an interval needs two, so this scenario reaches
# a real `no-effect` rather than an honest refusal to measure.
:
# >>> baseline
kno baseline --evals cases.jsonl --agent fake: --goal exact-match \
  --holdout-frac 0.2 --seed 1 --concurrency 1 \
  --db kno.db --run-id ca-baseline --yes
# <<< baseline
# >>> value
kno value --evals cases.jsonl --pool pool.jsonl --baseline-run-id ca-baseline \
  --agent fake: --goal exact-match --seed 1 --routing-seed 1 --concurrency 1 \
  --db kno.db --run-id ca-value --yes
# <<< value
# >>> select
kno select --value-run-id ca-value --pool pool.jsonl \
  --max-context-tokens 5000 --max-training-examples 10 --max-cost-usd 1 \
  --db kno.db --run-id ca-select
# <<< select
# >>> export
kno export --select-run-id ca-select --pool pool.jsonl \
  --destination tuning_set --out tuning.jsonl \
  --db kno.db --run-id ca-export
# <<< export
