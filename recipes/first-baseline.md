---
verification: executed
scenario: support-refunds
stage: baseline
requires-stages: []
last-verified: 2026-09-05
verified-against: kno v0.2.1
---

# Score your agent for the first time

Goal: get from nothing to a baseline you can trust, and understand what it says.

## 0. Watch it happen once, for free

```bash
kno demo
```

`kno demo` writes a twelve-Case eval set and a three-Asset pool into `./kno-demo`, then runs
all five stages over them against the built-in `fake:` agent. It spends nothing, contacts
nothing, and reads no configuration — not `kno.yaml`, not `KNO_*` — so it cannot bill anyone
whatever your environment says. It finishes in well under a second, and you have seen the
shape of everything below.

Its score reads `1.000`, its deltas read `+0.0000`, and its portfolio comes back empty. All
three are honest rather than flattering: `fake:` answers every Case with what the Case
expects, injecting an Asset cannot change a deterministic answer, and an interval around zero
that crosses zero is an Asset that has not earned its place. The demo prints those three
sentences itself.

The eval set it writes is the same one this page is verified against — it is committed here as
[`scenarios/support-refunds/evals/cases.jsonl`](../scenarios/support-refunds/evals/cases.jsonl),
and the run below is [`scenarios/support-refunds/run.sh`](../scenarios/support-refunds/run.sh).

## 1. Write some Cases

A Case is one scoreable interaction: an input, and what a correct answer looks like. One JSON
object per line.

```jsonl
{"id":"refund-01","input":"How do I get a refund?","expected":"Refunds are issued within 5 business days.","tags":["refunds"]}
{"id":"ship-01","input":"When will my order arrive?","expected":"Standard shipping takes 3-5 business days.","tags":["shipping"]}
{"id":"acct-01","input":"How do I reset my password?","expected":"Use the reset link on the sign-in page.","tags":["account"]}
```

Every Case needs a stable `id`. That isn't bookkeeping: the dev/holdout split is keyed on it,
so a Case without a stable id would have its half decided by its position in the file, and
inserting a line above it would silently move it between halves.

Kno refuses a file with a missing id rather than guessing one.

**How many?** Enough that a fifth of them is a meaningful holdout. Below ~100 Cases the holdout
starts producing intervals too wide to conclude much from — Kno will run anyway and tell you
it's underpowered. The twelve-Case set below is deliberately under that line, and the output
says so, which is the point of showing it.

## 2. Run it

```bash kno-run scenario=support-refunds stage=baseline
kno baseline --evals cases.jsonl --agent fake: --goal exact-match \
  --holdout-frac 0.2 --seed 1 --concurrency 1 \
  --db kno.db --run-id sr-baseline --yes
```

The default agent is `fake:`, which answers deterministically and costs nothing. That's on
purpose: you should be able to see the whole loop work before pointing it at something that
bills you. Everything else on that line is pinned so that two runs of it produce identical
bytes — `--run-id` so the id is not generated, `--seed` and `--concurrency 1` so nothing
depends on completion order, `--holdout-frac` so the split does not move under a changed
default. Drop them all and `kno baseline --evals cases.jsonl` does the same work.

## 3. Read the output

```
Baseline sr-baseline
  cases      8 scored, 0 errored (of 8 dev; 4 held back)
  score      1.000
  spent      $0.00 over 8 call(s)
  status     completed

  warning: the holdout has only 4 cases, too few for a meaningful confidence interval at validate

Scores and traces are recorded. `kno purge` removes trace content when you no longer need it.
```

That is not an illustration. Every quoted line above is asserted against the real binary's
output by `scenarios/support-refunds/expected/quotations.json`; if kno reformats one of them,
this page goes red and the prose is what changes.

- **`4 held back`** — never scored, and nothing will read them until `validate`. This is what
  makes the eventual number honest.
- **`0 errored`** — Cases where the agent didn't answer at all. Counted separately and excluded
  from the score, because a 500 isn't a wrong answer.
- **`score      1.000`** — the mean over scored Cases. The fake echoes the expected answer, so
  of course it's perfect. Your real agent will not be.
- **The warning** — read it. A four-case holdout can't support a meaningful interval. Twelve
  Cases is a demonstration, not an eval set.

## 4. Keep the run ID

```bash
kno baseline --evals cases.jsonl --run-id nightly-2026-08-21
```

Runs are the correlation key for every trace, score, and event. Naming them yourself makes a
resume — and later, a comparison — straightforward.

Anything you repeat on every run can live in `kno.yaml` instead of the command line: `kno init`
writes one, and the file can carry `agent`, `goal`, `max_cost_usd`, `max_calls`, `concurrency`,
`holdout_frac`, and `key_env`. A committed file is a per-team default; `--yes` and the security
booleans are deliberately flags-only.

## Common problems

**"case has no id"** — every Case needs one. See step 1.

**"all N Cases landed in dev, leaving no holdout"** — your eval set is too small for the
configured fraction. Kno refuses here rather than at `validate`, so you find out before spending
anything. Add Cases — the refusal's own fix is "add more cases, or lower `--holdout-frac`", but
on a tiny set no fraction produces a usable split: below ~6 Cases the holdout collapses to one
half or the other.

**"duplicate case id"** — two Cases share an id. Since the split is keyed on id they'd land in
the same half and be indistinguishable in every later report, so this is fatal rather than
tolerated.

**The score looks too good** — check `errored`. If most Cases failed to answer, the score is
over the handful that succeeded, and Kno will have marked the run unusable as a reference.

## Next

- [Choose a portfolio under budget](select-a-portfolio.md) — which assets earn their place.
- [What the numbers mean](https://github.com/uknoAI/kno/blob/main/docs/what-the-numbers-mean.md)
  — before you act on a score.
