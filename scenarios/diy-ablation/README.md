# `diy-ablation`

The scenario that answers the objection every competent engineer has on first reading and
almost nobody says out loud: **why not just script this myself?**

It is a fair objection. A context ablation is not deep magic — score the eval set with each
candidate document in the prompt, score it without, keep whichever helped most. A hundred lines,
no dependencies, done in an afternoon.

So this scenario ships that script, runs it in CI, and prints its output next to Kno's on the
same data.

```
scenarios/diy-ablation/
  naive_ablation.py      the hundred lines, committed in full and executed by CI
  evals/cases.jsonl      24 Cases about an invented API, across 4 behaviors
  pool/pool.jsonl        3 candidate documentation pages
  run.sh                 four stages: naive, then baseline, value, select
  expected/              one projected JSON document per stage, plus quotations.json
  DATA-PROVENANCE.md     who wrote this data, and the assertion that it is synthetic
```

Run it:

```sh
sh scenarios/diy-ablation/run.sh /tmp/diy
```

It needs a released `kno` and a `python3` on `PATH`. It is the one scenario here that needs a
python; the script imports only the standard library.

## The two outputs, side by side

Same 24 Cases. Same 3 Assets. Same agent — the script's `agent()` is the echo agent `fake:`
implements, returning each Case's expected answer. Same scorer — exact match. **Every difference
below is a difference of method, because there is nothing else left to explain it.**

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

```
Selected 0 — greedy on delta-per-cost, no approximation guarantee; keep/reject decisions used Bonferroni-corrected intervals
  RANK  ASSET         DESTINATION       DELTA (95% CI)

Rejected 3
  auth-guide    no-effect         delta +0.0000, CI [-0.2604, +0.2604] crosses zero
  changelog-2024  no-effect         delta +0.0000, CI [-0.2604, +0.2604] crosses zero
  limits-page   no-effect         delta +0.0000, CI [-0.2604, +0.2604] crosses zero
```

Neither of those is an illustration. Both are asserted against the real binary and the real
script by `expected/quotations.json`; if either reformats a line, this page goes red and the
*prose* is what changes.

One of them names a winner. The other declines to.

## Why the script names a winner

Because every delta is identical, and `max()` returns something anyway — the first element of
the tie, which here is the first line of `pool.jsonl`. The recommendation this script produces
is decided by the order of lines in a file.

That is not a bug in the script. It is what "pick the best one" means when the evidence does not
distinguish the candidates, and no amount of care inside those hundred lines detects it, because
detecting it requires an interval and the script has no interval.

**"I would add a `> 0` threshold."** Then this tie is rejected and the script agrees with Kno
here. On real data it does not: the deltas are noise rather than zeros, the largest of three
noise draws is reliably above zero, and a threshold accepts it. That is the winner's curse, and
a threshold is exactly the wrong shape of fix for it — it filters on the point estimate, and the
point estimate is the thing that is inflated.

The reason this scenario runs on `fake:` and not on noise is that noise costs money and would
make the nightly non-deterministic. The tie is the *cheapest honest demonstration* of a property
that holds in general: `max` over a set of estimates has no way to say "none".

## The five things, and where each one is visible

| | The script | Kno, in this run |
|---|---|---|
| **Sealed holdout** | `scored 24 of 24 cases; 0 held back` | `cases      20 scored, 0 errored (of 20 dev; 4 held back)` |
| **An interval** | `+0.0000` | `+0.0000  [-0.2132, +0.2132]` |
| **Multiplicity** | three questions, one threshold | `±0.2132` widened to `±0.2604` — Bonferroni for three Assets |
| **A control arm** | none | `low -0.2908 (underpowered)` — the bound on harm, and the admission it is weak |
| **A verdict of "no"** | `winner: auth-guide` | `Rejected 3 … crosses zero` |

Read the interval column twice. `±0.2132` on a metric that lives in [0, 1] means this eval set
could not have detected a 20-point improvement. The script reports `+0.0000` and calls it a
result; Kno reports the same `+0.0000` and reports how much it could not see.

The multiplicity row is the one that is hardest to get right by hand and easiest to not notice
missing. `kno value` computes `±0.2132` for each Asset in isolation, and `kno select` widens it
to `±0.2604` before deciding, because a decision made after looking at three candidates is not
the same decision as one made after looking at one. The two numbers differ by 22%, and nothing
about the un-widened one looks wrong.

**None of the five is hard.** Each is an afternoon. The claim is not that they are difficult; it
is that they are five, that each is silent when you get it wrong, and that the naive table above
looks exactly as convincing with all five missing as it would with none.

## What this does not show

The same ceiling as every committed scenario here, and it bites this one hardest, so it is worth
being blunt: **the deltas are zero by construction.** `fake:` answers every Case with what the
Case expects, so injected context cannot move the score, and neither program is measuring
anything real.

What is real is the *shape* of each program's answer to a table of zeros — one names a winner,
one refuses — and every mechanism in the comparison table, all of which are functions of the
Case count and the number of Assets rather than of any model's behaviour. Those hold identically
against a paid provider.

What this scenario cannot show is the case that actually costs money: three noisy deltas where
the naive script picks the largest and Kno's interval rejects it. Demonstrating that needs an
agent whose answers vary, which means a paid provider, which the nightly may not run. It would
be a `manual` page against a real key — a different tier making a different promise — and it is
not this one.
