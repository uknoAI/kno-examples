---
verification: executed
scenario: support-refunds
stage: value
requires-stages: [baseline]
last-verified: 2026-09-03
verified-against: kno v0.2.0
---

# Value a pool of assets

What you're asking: **does this asset earn its place?** `kno value` answers it per asset — how much it moves your Goal, how sure Kno is, and what it cost to find out.

## Before you start

You need two files and one recorded run:

- **Cases** — the same eval file you scored the baseline against.
- **Assets** — a JSONL pool, one asset per line:
  ```json
  {"id":"refund-policy-faq","content":"Returns are accepted within 30 days...","tags":["billing"]}
  {"id":"pricing-tier-table","content":"Pro: $29/mo...","tags":["billing"]}
  ```
- **A baseline** — from `kno baseline` (same `--db`), whose recorded scores every delta pairs against.

This page is verified as the second stage of the `support-refunds` scenario, and the transcript
below is that scenario's real output. The command reads a store the baseline stage wrote, so
`sh scenarios/support-refunds/run.sh` is what produces it — pasted into a fresh shell against an
empty store, it has no baseline to pair against and will say so.

## Run it

```bash kno-run scenario=support-refunds stage=value
kno value --evals cases.jsonl --pool pool.jsonl --baseline-run-id sr-baseline \
  --agent fake: --goal exact-match --seed 1 --routing-seed 1 --concurrency 1 \
  --db kno.db --run-id sr-value --yes
```

`sr-baseline` is the run id the previous stage recorded. On your own data it is whatever
`kno baseline --run-id` was given, or the id kno generated and printed. Everything else on the
line is pinned so two runs produce identical bytes — `--seed` and `--routing-seed` so the draws
repeat, `--concurrency 1` so completion order cannot reorder anything. Drop them all and
`kno value --evals cases.jsonl --pool pool.jsonl --baseline-run-id <id>` does the same work.

The run routes each asset to the Cases it could plausibly affect (by tag overlap with the baseline's failures), injects it into the agent's context for the treatment arm, re-runs the same Cases without it for the control, and records one Valuation per asset.

## Read the report

```
Planning 36 measurements over 3 assets against baseline sr-baseline.
Value run sr-value (RUN_STATUS_COMPLETED)

ASSET         DELTA (95% CI, positive = goal dir)  CONTROL             NOTE
brand-guide   +0.0000  [-0.3960, +0.3960]   low -0.5815 (underpowered)
refund-policy-v3  +0.0000  [-0.3960, +0.3960]   low -0.5815 (underpowered)
ship-promise  +0.0000  [-0.3960, +0.3960]   low -0.5815 (underpowered)

Scores and traces are recorded. `kno purge` removes trace content when you no longer need it.
```

That is not an illustration. The delta, the interval and the `(underpowered)` flag above are
asserted against the real binary's output by `scenarios/support-refunds/expected/quotations.json`,
and the `--json` document behind it by `expected/value.json`; if kno reformats one of them, this
page goes red and the prose is what changes.

- **DELTA** is the mean change on the Cases the asset was routed to, with its interval. A delta without an interval is never printed — if the sample is too small or too ragged to form one, the row says so.
- **CONTROL** is the one-sided harm bound over the untouched reserved slice: how much damage could be hiding in "no regression". Read it as a limit, not a score — see [what the numbers mean](https://github.com/uknoAI/kno/blob/main/docs/what-the-numbers-mean.md).
- **routed to nothing** is a real answer: the asset matches no failure cluster, so it costs nothing and changes nothing.
- If a row says `underpowered`, the control sample could not see a regression as small as the harm margin. The number travels first; the flag travels beside it. Every row above says it, because twelve Cases is a demonstration and not an eval set.

**Every delta being exactly `+0.0000` is the honest answer, not a broken run.** `fake:` answers
each Case with what the Case expects, so it is already right, and injecting an Asset cannot change
a deterministic correct answer. An interval that crosses zero is an Asset that has not earned its
place — which is what [`kno select`](select-a-portfolio.md) then records, with a reason per Asset.

## When the budget stops the run

A run that hits `--max-cost-usd` stops resumably: everything measured stays recorded, the unfinished asset is marked `budget exhausted mid-measurement; --resume continues`, and

```sh
kno value --evals cases.jsonl --pool assets.jsonl \
  --baseline-run-id <id> --agent fake: --resume --yes
```

continues from exactly where it stopped — without paying for anything twice.

## What to watch for

- **The dropped-pairs count.** A Case whose treatment arm errored is dropped, and dropped Cases are exactly the ones where the asset was most harmful (a long injected context that times out). The delta drifts upward when this happens, and the report says how many pairs went missing.
- **Your own conditioning.** If you tagged Cases or wrote assets after reading the baseline's failures, Kno cannot see that — and the deltas can be biased by how often the agent gets those Cases right on a re-run. [ADR-0005](https://github.com/uknoAI/kno/blob/main/docs/adr/0005-value-cannot-see-user-side-conditioning.md) says why, and `validate` is the stage that catches it.

## Next

- [Choose a portfolio under budget](select-a-portfolio.md) — `kno select` turns the recorded Valuations into the Portfolio.
- [Read the whole story with `kno report`](read-the-whole-story.md) — the value page and its refusals, composed with the other stages.

Anything you repeat on every run can live in `kno.yaml` instead of the command line: `kno init` writes one, and the file can carry `agent`, `goal`, `max_cost_usd`, `max_calls`, and `key_env`. `--yes` is deliberately flags-only.
