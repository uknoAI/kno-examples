---
verification: executed
scenario: diy-ablation
stage: naive
requires-stages: []
last-verified: 2026-09-02
verified-against: kno v0.2.0
---

# Why not just script this yourself?

You can. That is the honest starting point, and any answer that begins somewhere else is
selling you something.

Scoring an eval set with a candidate document in the prompt and without it, then keeping
whichever helped, is about a hundred lines of Python with no dependencies. Most engineers who
read about Kno think this within the first minute, and most of them are right that they could
write it before lunch.

So this page does not argue with that. It runs the script.

[`scenarios/diy-ablation/naive_ablation.py`](../scenarios/diy-ablation/naive_ablation.py) is
committed here in full, executed by CI on every pull request, and asserted against committed
expectations exactly like every `kno` command in this repository. It runs over the same 24
Cases and the same 3 Assets as the three `kno` stages beside it, with **the same agent** — its
`agent()` returns each Case's expected answer, which is precisely what `fake:` does — and **the
same scorer**, exact match.

Every difference in the output below is therefore a difference of method. There is no
difference of model, data, or metric left to explain it.

## 1. Run the script

```bash kno-run scenario=diy-ablation stage=naive
python3 naive_ablation.py --evals cases.jsonl --pool pool.jsonl
```

```
naive ablation over 24 cases, 3 assets
  agent   echo (returns each case's expected answer)
  scorer  exact match

ASSET                 WITH   WITHOUT    DELTA
auth-guide          1.0000    1.0000  +0.0000
limits-page         1.0000    1.0000  +0.0000
changelog-2024      1.0000    1.0000  +0.0000

winner: auth-guide (+0.0000)
scored 24 of 24 cases; 0 held back
```

Two lines to sit with.

**`winner: auth-guide (+0.0000)`.** Every delta is identical, and the script names a winner
anyway, because `max()` returns something. It returns the first element of the tie, which is the
first line of `pool.jsonl` — so the recommendation this script produces is decided by the order
of lines in a file.

**`scored 24 of 24 cases; 0 held back`.** The number it reports is the number it selected on.
Whatever it says, it is not evidence about any Case the script has not already looked at.

## 2. Run the same thing through Kno

```bash kno-run scenario=diy-ablation stage=baseline
kno baseline --evals cases.jsonl --agent fake: --goal exact-match \
  --holdout-frac 0.2 --seed 1 --concurrency 1 \
  --db kno.db --run-id diy-baseline --yes
```

```
Baseline diy-baseline
  cases      20 scored, 0 errored (of 20 dev; 4 held back)
  score      1.000
  spent      $0.00 over 20 call(s)
  status     completed

  warning: the holdout has only 4 cases, too few for a meaningful confidence interval at validate
```

```bash kno-run scenario=diy-ablation stage=value
kno value --evals cases.jsonl --pool pool.jsonl --baseline-run-id diy-baseline \
  --agent fake: --goal exact-match --seed 1 --routing-seed 1 --concurrency 1 \
  --db kno.db --run-id diy-value --yes
```

```
Planning 84 measurements over 3 assets against baseline diy-baseline.

ASSET         DELTA (95% CI, positive = goal dir)  CONTROL             NOTE
auth-guide    +0.0000  [-0.2132, +0.2132]   low -0.2908 (underpowered)
```

```bash kno-run scenario=diy-ablation stage=select
kno select --value-run-id diy-value --pool pool.jsonl \
  --max-context-tokens 5000 --max-training-examples 10 --max-cost-usd 1 \
  --db kno.db --run-id diy-select
```

```
Selected 0 — greedy on delta-per-cost, no approximation guarantee; keep/reject decisions used Bonferroni-corrected intervals

Rejected 3
  auth-guide    no-effect         delta +0.0000, CI [-0.2604, +0.2604] crosses zero
```

None of those blocks is an illustration. Every quoted line is asserted against the real binary —
and the naive script's lines against the real script — by
`scenarios/diy-ablation/expected/quotations.json`. If either reformats a line, this page goes
red and the *prose* is what changes.

## The five differences

| | The script | Kno |
|---|---|---|
| Sealed holdout | `scored 24 of 24 cases; 0 held back` | `20 scored … 4 held back` |
| An interval | `+0.0000` | `+0.0000  [-0.2132, +0.2132]` |
| Multiplicity | three questions, one threshold | `±0.2132` widened to `±0.2604` |
| A control arm | none | `low -0.2908 (underpowered)` |
| A verdict of "no" | `winner: auth-guide` | `Rejected 3 … crosses zero` |

**The holdout.** `kno baseline` seals a fraction of the Cases before the first call and reports
how many it sealed. Nothing reads them until `validate`. That is what makes the eventual number
a claim about Cases the selection never saw, and it is the difference between a measurement and
a fit.

**The interval.** `±0.2132` on a metric that lives in [0, 1] says this eval set could not have
detected a twenty-point improvement. The script reports `+0.0000` and stops; Kno reports the
same `+0.0000` and reports how much it could not see. Those are not the same claim, and only one
of them tells you whether to add Cases before believing anything. If you want that number
*before* you run anything, [`kno eval inspect`](power-and-sample-size.md) computes it for free.

**Multiplicity.** `kno value` computes `±0.2132` per Asset in isolation; `kno select` widens it
to `±0.2604` before deciding, because a decision made after looking at three candidates is not
the decision you would have made after looking at one. The two numbers differ by 22%, and
nothing about the un-widened one looks wrong. This is the step most commonly missing from a
hand-rolled ablation, and the one whose absence is least visible.

**The control arm.** Nothing in the script measures whether injecting an Asset made some *other*
behaviour worse, so a document that fixes one tag and breaks another shows up as an improvement.
Kno reserves a partition out of routing for exactly that, reports the bound — `low -0.2908` —
and then says `(underpowered)`, which is the tool declining to let you read a reassuring number
off a weak test.

**A verdict of "no".** `max` over a set of estimates has no way to return "none of these". An
empty Portfolio is a screen an ablation script does not have, and it is the answer most of the
time.

## "I would add a significance test"

Good — that is one of the five, and it is the right instinct. Two things follow.

The first is that you now need the other four, and each of them is silent when you get it wrong.
A missing holdout does not throw. An uncorrected interval prints the same shape as a corrected
one. A missing control arm reports an improvement. The table above looks exactly as convincing
with all five missing as it would with none, which is the actual problem: the failure mode of a
hand-rolled ablation is not that it breaks, it is that it keeps working and quietly overclaims.

The second is subtler. A `> 0` threshold rejects the tie above, so on *this* data the script
would agree with Kno. On real data it will not: the deltas are noise rather than zeros, the
largest of three noise draws is reliably above zero, and the threshold accepts it. That is the
winner's curse, and a threshold on the point estimate is the wrong shape of fix for it — the
point estimate is the thing that is inflated. The fix is the interval, corrected for how many
questions you asked, which is #2 and #3 rather than a fourth idea.

## The things that are not statistics

Three more, less interesting to argue about and more likely to be what actually costs you a
week:

- **A budget that stops.** `kno value` printed `Planning 84 measurements` before it ran any of
  them. `--max-cost-usd` and `--max-calls` stop a run before it spends past a ceiling rather
  than after. See [what a run costs](what-it-costs.md).
- **Resume.** `--resume` continues an interrupted run from its checkpoint. A script that dies at
  hour three of a paid ablation starts again at hour zero.
- **An exit code.** `kno select --json` plus a threshold is a build gate; see
  [gating a deploy in CI](ci-gate.md). A print statement is not.

## None of this is hard

That is the honest summary, and it is deliberately not "you couldn't build this".

Each of the eight things above is an afternoon. The claim is that they are eight, that they
compose, that six of the eight are invisible when they are wrong, and that the version you
write in an afternoon is the version you will be reading numbers off for the next year without
ever getting the signal that it is misleading you.

If you would rather build it, the eight above are a decent specification. If you would rather
not, that is what this is.

## Next

- [How many Cases do I need?](power-and-sample-size.md) — the free command that answers it
  before you spend anything.
- [What a run costs](what-it-costs.md) — the arithmetic, and the three caps.
- [`scenarios/diy-ablation/README.md`](../scenarios/diy-ablation/README.md) — the scenario, and
  what it deliberately cannot show.
