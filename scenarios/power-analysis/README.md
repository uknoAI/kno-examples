# `power-analysis`

The scenario that answers **how many Cases do I need**, which is the first question anybody
who has run an eval before actually asks, and the one a feature list cannot answer.

One eval set, read at three sizes, then measured end to end at the largest:

| | Cases | dev / holdout | Smallest effect a behavior could separate | Checks flagged |
|---|---:|---:|---:|---:|
| `inspect-12` | 12 | 11 / 1 | 1.76 – 6.35 | 2 of 5 |
| `inspect-40` | 40 | 36 / 4 | 0.51 – 0.59 | 1 of 5 |
| `inspect-160` | 160 | 133 / 27 | 0.25 | 0 of 5 |

A score lives in [0, 1], so the largest improvement an Asset could possibly produce is 1.00. A
separable effect of 1.76 therefore means *nothing at all is detectable* — not "small effects are
hard", but that the smallest effect this eval set could tell apart from noise is larger than the
biggest effect that can exist. Twelve Cases is not a small eval set. It is not an eval set.

Every number above comes out of `kno eval inspect`, which makes no LLM call, constructs no
agent, creates no Run and writes nothing. The sweep is free, offline, and takes about a second.

```
scenarios/power-analysis/
  evals/cases.jsonl      160 Cases across 4 behaviors, interleaved
  evals/generate.py      the program that wrote them, committed beside them
  pool/pool.jsonl        3 candidate Assets
  run.sh                 seven stages: three inspects, then baseline, value, select, attribute
  expected/              one projected JSON document per stage, plus quotations.json
  DATA-PROVENANCE.md     who wrote this data, and the assertion that it is synthetic
```

Run it:

```sh
sh scenarios/power-analysis/run.sh /tmp/pa
```

It needs a released `kno` on `PATH` and nothing else. `python3` regenerates the Cases; it is not
needed to run the scenario.

## What the sweep is for

Two questions get conflated constantly, and this scenario separates them.

**"Is my eval set big enough?"** is answerable before you spend anything, and it has an exact
answer. Kno's answer is `separable_effect`: the smallest true improvement a behavior's dev Cases
could tell apart from noise at 95%, computed from the count and the worst-case paired-binary
standard deviation. It is a bound rather than an estimate from your data, which is what lets it
be computed before there is any data.

**"Did this asset help?"** is the expensive question, and it is only worth asking once the first
one has a tolerable answer. Running it first is how teams end up with a table of deltas whose
intervals are wider than the deltas — a month of API spend that could have been ruled out by a
command that costs nothing.

## What the three checks that flip actually mean

At n=12 two checks are flagged; at n=160 none are. The two that move are not the same claim:

- **`behaviors_powered`** clears between 12 and 40. It asks whether each behavior has at least
  `core.MinClusterCases` (5) dev Cases, which is what routing needs before it will attribute a
  result to that behavior rather than to the set as a whole.
- **`holdout_powered`** clears between 40 and 160. It asks whether the *holdout* — the Cases
  never scored during development — is large enough for `validate` to form a meaningful
  interval. The floor is 20, and 20 held back at a holdout fraction of 0.2 means roughly 100
  Cases in the file.

They clear in that order and they clear at different sizes, which is the practical finding: an
eval set can be big enough to tell you which behavior is failing and still be too small to tell
you, at the end, whether you fixed it.

The holdout counts here are 1, 4 and 27 rather than 2, 8 and 32. That is not a rounding
mistake. The split is keyed on a hash of the Case id rather than on position, so the realised
fraction wanders around the requested one and wanders most when n is small. At twelve Cases,
"hold back 20%" delivered one Case. Reading the actual number rather than assuming the fraction
is exactly the habit this stage is for.

## What the measurement half adds

The three `inspect` stages are predictions. `baseline`, `value` and `select` are what actually
happened when the loop ran over all 160, and the interval they produce is the payoff:

| Scenario | Cases | Interval on every Asset |
|---|---:|---|
| [`support-refunds`](../support-refunds/README.md) | 12 | `+0.0000 [-0.3960, +0.3960]` |
| `power-analysis` | 160 | `+0.0000 [-0.0360, +0.0360]` |

Same agent, same true effect of exactly zero, same flags, two scenarios. Thirteen times the
Cases bought an interval eleven times narrower: from "this Asset is somewhere between 40 points
worse and 40 points better" to "somewhere between 3.6 worse and 3.6 better". One of those is a
finding you can act on and the other is a sentence with no content.

Do not read a rate off those two rows. They are two different scenarios with different Pools, so
the number of measured pairs behind them (5 and 75) is not simply the Case count, and the
interval's width depends on the paired standard deviation and the t-multiplier as well as on n.
What the pair establishes is the direction and the rough order of the gain. If you want the
number for *your* eval set, `kno eval inspect` computes it directly, before you spend anything —
which is the whole reason the first three stages exist.

`select` then widens `±0.0360` to `±0.0440` before deciding. That is the Bonferroni correction
for having asked about three Assets instead of one, and it is the single most commonly omitted
step in a hand-rolled ablation. It is visible here as two numbers that differ.

The last stage, `attribute`, is `kno eval inspect` again with `--value-run-id`, which is the only
way to get the fifth check off `unknown`. It reports what the run's routing *actually* did
rather than what the eval set said it could do, and it is where the control arm's harm bound
shows up: `control arm 39 Cases, minimum detectable harm 0.19 (one-sided 95%) — underpowered`.
Even at 160 Cases the harm test is underpowered, and the tool says so rather than reporting a
reassuring number. That is the honest ceiling on this scenario, printed by the tool itself.

## What this does not show

The same ceiling as every committed scenario here: the agent is `fake:`, which answers every
Case with exactly what the Case expects, so every score is 1.000 and every delta is exactly
zero. No choice of Asset content could move it.

That is why this scenario is about **widths, counts and verdicts** rather than about an Asset
winning. Everything it demonstrates — separable effect, the check flips, the interval narrowing,
the Bonferroni widening — is a function of n and of the tags, and is therefore exactly as true
against a paid provider as it is here. The point estimate is the only thing `fake:` fixes, and
the point estimate is the one number this scenario never asks you to believe.

The Cases themselves are generated rather than hand-written, and
[`evals/generate.py`](evals/generate.py) says why: nothing here depends on their wording, and
160 hand-written Cases would have bought the appearance of effort and nothing else.
