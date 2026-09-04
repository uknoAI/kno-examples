---
verification: executed
scenario: power-analysis
stage: value
requires-stages: []
last-verified: 2026-09-04
verified-against: kno v0.2.1
credentials: [OPENAI_API_KEY]
---

# What a run costs

The number nobody publishes and everybody needs before they can get approval to try anything.

Short version: **an agent call is the unit, and Kno tells you how many it will make before it
makes any of them.** Multiply by your provider's price. Three independent caps stop the run if
you were wrong.

## The free tier is not a trial

Everything below runs against `fake:`, the built-in local agent, and costs exactly nothing. So
does `kno demo`, and so does every scenario in this repository — the nightly job that verifies
these pages has never spent a cent.

Two commands are free even against a real setup, because they construct no agent at all:

```bash kno-run scenario=power-analysis stage=inspect-12
kno eval inspect --evals cases-12.jsonl --holdout-frac 0.2
```

`kno eval inspect` reads an eval set and reports whether it can support attribution. `kno mine`
turns transcripts into Cases. Neither makes an LLM call. Between them they cover the two things
worth doing before you spend anything: [check the eval set is big
enough](power-and-sample-size.md), and [build it out of logs you already
have](mine-your-transcripts.md).

(A remote eval source — `langsmith:`, `langfuse:`, `braintrust:`, `hf:` — still reaches its
vendor's API with the vendor's credentials, because reading the dataset is the job. "Costs
nothing" is a claim about LLM spend.)

## Baseline: one call per dev Case

```
Baseline pa-baseline
  cases      133 scored, 0 errored (of 133 dev; 27 held back)
  spent      $0.00 over 133 call(s)
```

133 dev Cases, 133 calls. The 27 held back are not called — nothing reads them until `validate`,
which is what makes them a holdout. So a baseline over N Cases at a holdout fraction *f* is
about `N × (1 − f)` calls, once.

## Value: the expensive stage, and it says so first

```bash kno-run scenario=power-analysis stage=value
kno value --evals cases.jsonl --pool pool.jsonl --baseline-run-id pa-baseline \
  --agent fake: --goal exact-match --seed 1 --routing-seed 1 --concurrency 1 \
  --db kno.db --run-id pa-value --yes
```

```
Planning 567 measurements over 3 assets against baseline pa-baseline.
…
  spent      $0.00 over 567 call(s)
```

**That first line is printed before the first call.** It is the number to budget from, and it is
the number to read rather than any formula on a documentation page — including the one below.

Four verified runs in this repository, and what they planned:

| Scenario | Cases | dev | Assets | Routed pairs | Control | Planned calls |
|---|---:|---:|---:|---:|---:|---:|
| [`support-refunds`](../scenarios/support-refunds/README.md) | 12 | 8 | 3 | 5 | 2 | 36 |
| [`transcript-mining`](../scenarios/transcript-mining/README.md) | 18 | 14 | 3 | 8 | 4 | 60 |
| [`diy-ablation`](../scenarios/diy-ablation/README.md) | 24 | 20 | 3 | 11 | 6 | 84 |
| [`power-analysis`](../scenarios/power-analysis/README.md) | 160 | 133 | 3 | 75 | 39 | 567 |

All four are consistent with:

```
calls ≈ assets × (2 × routed_pairs + control_cases)
```

Two calls per routed Case per Asset — one arm with the Asset in context, one without, which is
what makes the comparison paired — plus the control arm, measured once per Asset. Routing and
sampling decide how many of the dev Cases each Asset is actually measured on, so `routed_pairs`
is not the dev count and is usually well below it: 75 of 133 at n=160, 5 of 8 at n=12.

**Use the table to build intuition and the printed line to build a budget.** Routing, the
control reserve, and the sampling rate are the tool's arithmetic, not this page's, and
`--sample-rate`, `--control-reserve`, `--control-sample-rate` and `--trials` all move it.

`--trials` is the one to watch: it repeats every measurement, so `--trials 3` is three times the
calls. It buys precision on a noisy agent and it buys nothing on a deterministic one.

## Turning calls into dollars

```
cost ≈ planned_calls × (input_tokens × input_price + output_tokens × output_price)
```

For a priced model Kno computes this itself and shows it before the run. For a model it has no
price table row for, give it one:

```bash
kno baseline --evals cases.jsonl --agent openai:my-fine-tune \
  --price-input-per-mtok 3.00 --price-output-per-mtok 12.00 \
  --max-output-tokens 512
```

`--max-output-tokens` is not only a generation ceiling. It is what **bounds the estimate**: with
no ceiling on output length there is no upper bound on the cost of a call, so several adapters
require it. That is the mechanism that makes a pre-run estimate an estimate rather than a guess.

Injecting an Asset adds its tokens to every treatment-arm call, and those are charged on top.
`--max-prompt-bytes` bounds the Case; the injected Asset is bounded separately.

## What happens when Kno cannot price a call

It asks, and if you told it not to ask, it says so out loud:

```
Proceeding with --yes: this run's per-Case cost is unknown.
```

That line is asserted against the real binary by
`scenarios/power-analysis/expected/quotations.json`. Without `--yes`, an unpriced run prompts
for consent; with `--yes`, the estimate is still printed. `--accept-unknown-cost` is the
explicit form for a model whose per-Case cost cannot be computed at all.

## The three caps

They are independent on purpose — one is a flag, one is a flag, one is the environment — so that
a mistyped command line cannot remove all three at once.

```bash
kno value --evals cases.jsonl --pool pool.jsonl --baseline-run-id nightly \
  --agent openai:gpt-4.1 --max-output-tokens 512 \
  --max-cost-usd 25.00 --max-calls 5000
```

| Cap | Where | What it does |
|---|---|---|
| `--max-cost-usd` | flag | stops **before** spending past the ceiling, not after |
| `--max-calls` | flag | stops after N agent calls; 0 is unlimited |
| `KNO_MAX_COST_USD` | environment | a ceiling the command line cannot raise |

"Stops before" is the part that matters. A cap that notices afterwards is a report, not a cap.

A run that hits a cap is **stopped, not lost**: `kno select --allow-partial` builds a Portfolio
from a Value run that did not complete, and carries the source status onto it so nothing
downstream can forget. And `--resume` continues an interrupted run from its checkpoint rather
than starting over, which is the difference between raising a cap and paying twice.

```bash
kno value --evals cases.jsonl --pool pool.jsonl --baseline-run-id nightly \
  --agent openai:gpt-4.1 --max-output-tokens 512 --max-cost-usd 50.00 --resume
```

A `kno.yaml` written by `kno init` can carry `max_cost_usd` and `max_calls` as team defaults, so
the floor is not a thing each person has to remember. `--yes` and the security booleans are
deliberately flag-only: a committed file may lower your risk, never raise it.

## What is asserted here and what is arithmetic

Being explicit, because this page mixes the two.

**Asserted against the real binary**, nightly, by the `power-analysis` scenario: the two quoted
commands, `Planning 567 measurements over 3 assets`, `spent $0.00 over 567 call(s)`, `133
scored … 27 held back`, and the `Proceeding with --yes` line. The Cases, dev counts, routed
pairs and planned calls for all four scenarios in the table come from committed expectations
that CI compares against real output.

**Not asserted**: the `calls ≈` approximation, the dollar formula, and every price. Those are
arithmetic and market data, neither of which a green tick covers. Read the line the tool prints.

## Next

- [How many Cases do I need?](power-and-sample-size.md) — the free command to run first.
- [Point Kno at your own provider](your-own-provider.md) — keys, base URLs, and the first run
  that can bill you.
- [Why not just script this yourself?](why-not-diy.md) — including what a budget guard is for.
