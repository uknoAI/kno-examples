---
verification: executed
scenario: support-refunds
stage: select
requires-stages: [baseline, value]
last-verified: 2026-09-04
verified-against: kno v0.2.1
---

# Choose a portfolio under budget

What you're asking: **which assets earn their place?** `kno value` measures each asset;
`kno select` decides — in precedence order — which ones survive your budget, and records the
Portfolio with a rejection log that says why every exclusion happened.

## Before you start

You need one recorded run and a budget:

- **A Value run** — from `kno value` (same `--db`). Select reads ONLY the recorded Valuations;
  it makes no LLM calls, reads no evals, and never touches the holdout.
- **A budget** — the carrying cost of the *selected* set, not what measurement cost. Name at
  least one cap:
  - `--max-context-tokens` — what the selected context may add per call
  - `--max-training-examples` — how many examples the tuning set may hold
  - `--max-cost-usd` — acquisition dollars for the selected assets
- **A pool** (optional but recommended) — the same `pool.jsonl` you valued. Without it, the
  content rules (`redundant`, `wrong-mechanism`) cannot run, and the report says they degraded
  rather than hiding it.

## Run it

```bash kno-run scenario=support-refunds stage=select
kno select --value-run-id sr-value --pool pool.jsonl \
  --max-context-tokens 5000 --max-training-examples 10 --max-cost-usd 1 \
  --db kno.db --run-id sr-select
```

`sr-value` is the run id the previous stage recorded. On your own data it is whatever
`kno value --run-id` was given, or the id kno generated and printed.

## Read the report

```
Select run sr-select (RUN_STATUS_COMPLETED)
  source    sr-value (completed)
  budget    context ≤ 5000 tokens; tuning ≤ 10 examples; cost ≤ $1.00

Selected 0 — greedy on delta-per-cost, no approximation guarantee; keep/reject decisions used Bonferroni-corrected intervals
  RANK  ASSET         DESTINATION       DELTA (95% CI)

Rejected 3
  brand-guide   no-effect         delta 0, CI [-0.4836569575661475, 0.4836569575661475] crosses zero
  refund-policy-v3  no-effect         delta 0, CI [-0.4836569575661475, 0.4836569575661475] crosses zero
  ship-promise  no-effect         delta 0, CI [-0.4836569575661475, 0.4836569575661475] crosses zero

Portfolio recorded. `kno export` writes the selected assets to their destinations.
```

*(The transcript is real output, captured on darwin/arm64. The trailing digits of that interval
differ by a few units in the last place on other architectures — a floating-point difference in
an iterative computation, not a change in the answer — which is why the committed assertion is
on the value rounded to four places, the same precision `kno value` renders.)*

**Selected 0 is a result, not a failure.** All three Assets were measured against a
deterministic agent, so every delta is exactly zero and every corrected interval crosses zero.
"Include nothing new" is a legal, first-class outcome, and the rejection log says why for each
Asset. That is the tool doing its job — an eval harness has no screen for this.

- **The construction is greedy, and says so.** Feasible, deterministic, and reproducible — no
  approximation guarantee. A later, cheaper asset can still fit where an earlier one did not.
- **Every decision ran at a corrected interval.** The 95% label on an individual asset is the
  family-wise error rate over everything screened, not a per-row check. Note that the interval
  in the rejection log is *wider* than the one `kno value` printed for the same Asset
  (`[-0.4837, +0.4837]` against `[-0.3960, +0.3960]`): that widening is the Bonferroni
  correction for having screened three Assets.
- **The gain line is a selection-time estimate.** `dev_estimated_gain` is winner's-curse
  inflated — the selection used up that information. The honest number arrives with `validate`,
  against the untouched holdout. An empty Portfolio has no gain line at all, which is why the
  transcript above does not show one.
- **Rejections are a deliverable.** The reason an asset was excluded is on the record:
  `regression` (harm), `no-effect` (corrected interval crosses zero), `redundant` (duplicates an
  already-selected asset), `cost-dominated` (does not fit), `wrong-mechanism` (real effect,
  wrong vehicle — e.g. knowledge destined for the tuning set).

## If the source run did not complete

`kno select` refuses a budget-stopped or interrupted Value run: ranking an incomplete
measurement set as if it were the whole answer would mislead. Either finish the Value run, or
pass `--allow-partial` to build from the recorded Valuations anyway — the source's status
travels with the Portfolio either way, so a reader cannot mistake it for a completed
measurement.

## Keep the run ID

The Portfolio is recorded under the Select run ID, and `kno export` is the next step:

```bash
kno select --json   # machine-readable: run_id, status, source_run_id, budget, selected, rejected, total_cost
```

## Next

- [Score your agent for the first time](first-baseline.md) — the stage this one reads from.
- [What the numbers mean](https://github.com/uknoAI/kno/blob/main/docs/what-the-numbers-mean.md)
  — before you act on a rejection.
