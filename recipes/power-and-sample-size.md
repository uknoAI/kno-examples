---
verification: executed
scenario: power-analysis
stage: inspect-12
requires-stages: []
last-verified: 2026-09-01
verified-against: kno v0.1.6
---

# How many Cases do I need?

The question everyone asks second and nobody answers first, usually because the honest answer —
"more than you have" — sounds like a sales pitch.

It is not a matter of judgement. Given an eval set and a holdout fraction, the smallest effect
that set could tell apart from noise is arithmetic, and `kno eval inspect` computes it. **It
makes no LLM call, constructs no agent, creates no Run and writes nothing.** It is free, it is
offline, and it takes about a second.

Run it before you spend anything. A month of API spend on a table of deltas whose intervals are
wider than the deltas is the most common way to waste money on evaluation, and this command is
the thing that would have prevented it.

## Start with the answer nobody wants

```bash kno-run scenario=power-analysis stage=inspect-12
kno eval inspect --evals cases-12.jsonl --holdout-frac 0.2
```

```
Evals  cases-12.jsonl
  12 Cases — 11 dev, 1 held back
  4 distinct behaviors (tags)

BEHAVIOR           DEV CASES  SEPARABLE EFFECT (two-sided 95%)  STATUS
freshness                  3                              1.76  underpowered
metric-definition          3                              1.76  underpowered
sql-dialect                3                              1.76  underpowered
access-policy              2                              6.35  underpowered

  ! 4 behaviors below core.MinClusterCases (5)
  ! the holdout has 1 Cases (20 is the minimum for a meaningful interval at
    validate)

2 of 5 checks flagged.
```

**Read the separable effect first.** A score lives in [0, 1], so the largest improvement an Asset
could possibly produce is 1.00. A separable effect of 1.76 says the smallest effect this eval
set could distinguish from noise is *larger than the largest effect that can exist*. Nothing is
detectable. Not "small effects are hard" — nothing.

Twelve Cases is not a small eval set. It is not an eval set. That is a useful thing to be told in
one second and an expensive thing to discover after a week.

**Then read the holdout.** "Hold back 20%" of twelve Cases delivered *one*. The split is keyed on
a hash of each Case id rather than on position, so the realised fraction wanders around the
requested one, and it wanders most when n is small. Read the number rather than assuming the
fraction.

## The same command over the whole set

```bash kno-run scenario=power-analysis stage=inspect-160
kno eval inspect --evals cases.jsonl --holdout-frac 0.2
```

`--holdout-frac` is worth stating even though `0.2` is the default: the fraction decides which
Cases are dev, so it changes every number the command prints.

```bash kno-run scenario=power-analysis stage=inspect-40
kno eval inspect --evals cases-40.jsonl --holdout-frac 0.2
```

| Cases | dev / holdout | Separable effect | Checks flagged |
|---:|---:|---:|---:|
| 12 | 11 / 1 | 1.76 – 6.35 | 2 of 5 |
| 40 | 36 / 4 | 0.51 – 0.59 | 1 of 5 |
| 160 | 133 / 27 | 0.25 | 0 of 5 |

The three rows are the same eval set at three sizes — `cases-12.jsonl` and `cases-40.jsonl` are
`head -n` of the same file, so this is a statement about n rather than about three different
populations. Every number in the table is asserted against the real binary by
`scenarios/power-analysis/expected/`.

**The two checks clear at different sizes, and that is the practical finding.**

- `behaviors_powered` clears between 12 and 40. It asks whether each behaviour has at least five
  dev Cases, which is what routing needs before it will attribute a result to that behaviour
  rather than to the set as a whole.
- `holdout_powered` clears between 40 and 160. It asks whether the *holdout* is large enough for
  `validate` to form a meaningful interval. The floor is 20 held back, which is roughly 100
  Cases in the file at a holdout fraction of 0.2.

An eval set can therefore be big enough to tell you which behaviour is failing and still be too
small to tell you, at the end, whether you fixed it. Those are two different budgets and it is
worth knowing which one you are short of.

## The rule of thumb, and why the command is better than one

If you want a number to put in a plan: **roughly 100 Cases** gets the holdout to 20 at a holdout
fraction of 0.2, and **at least five dev Cases per behaviour you intend to attribute
separately**. Kno's own suggestion line says as much:

```
  - grow the eval set to roughly 100 Cases so the holdout reaches 20 at a
    holdout fraction of 0.20, or validate will have no meaningful interval
```

The reason to run the command rather than apply the rule is that the rule assumes your Cases are
spread evenly across behaviours and yours are not. Four hundred Cases where 380 carry one tag
are 380 Cases about one behaviour and 20 about everything else, and `behavior_concentration`
says so where a headline count cannot.

Everything the per-tag half of the output says assumes **your tags name behaviours you would fix
separately**. Kno cannot tell a behaviour tag from a priority, a source, or a date, and it says
so above every number that depends on the assumption. If your tags are `p0`, `zendesk` and
`2026-Q1`, the per-tag table is arithmetic about nothing.

## What the extra Cases actually buy

The prediction above is worth checking against a run. `power-analysis` runs the whole loop over
all 160.

**The three blocks below are not standalone.** They read a store an earlier stage wrote — `kno
value` needs the recorded baseline `pa-baseline`, and `kno select` and the final `kno eval
inspect` need the Value run. Run
[`scenarios/power-analysis/run.sh`](../scenarios/power-analysis/run.sh) and they all resolve;
paste one into a fresh shell and it will find nothing. Everything above this line reads a file
and needs no store at all.

```bash kno-run scenario=power-analysis stage=value
kno value --evals cases.jsonl --pool pool.jsonl --baseline-run-id pa-baseline \
  --agent fake: --goal exact-match --seed 1 --routing-seed 1 --concurrency 1 \
  --db kno.db --run-id pa-value --yes
```

| Scenario | Cases | Interval on every Asset |
|---|---:|---|
| [`support-refunds`](../scenarios/support-refunds/README.md) | 12 | `+0.0000 [-0.3960, +0.3960]` |
| [`power-analysis`](../scenarios/power-analysis/README.md) | 160 | `+0.0000 [-0.0360, +0.0360]` |

Same agent, same true effect of exactly zero, same flags. The move is from "this Asset is
somewhere between 40 points worse and 40 points better" to "somewhere between 3.6 worse and 3.6
better". One of those is a finding and the other is a sentence with no content.

Do not read a rate off those two rows — they are two scenarios with different Pools, so the
number of measured pairs behind them is not simply the Case count. For your own set,
`kno eval inspect` gives you the number directly.

## Then correct for how many questions you asked

```bash kno-run scenario=power-analysis stage=select
kno select --value-run-id pa-value --pool pool.jsonl \
  --max-context-tokens 5000 --max-training-examples 10 --max-cost-usd 1 \
  --db kno.db --run-id pa-select
```

```
Selected 0 — greedy on delta-per-cost, no approximation guarantee; keep/reject decisions used Bonferroni-corrected intervals

Rejected 3
  access-policy  no-effect         delta +0.0000, CI [-0.0440, +0.0440] crosses zero
```

`kno value` reported `±0.0360` per Asset. `kno select` decides on `±0.0440`, because a decision
made after looking at three candidates is not the decision you would have made after looking at
one. Sample size and multiplicity pull in opposite directions: more Cases narrow the interval,
more Assets widen it, and the second is the one people forget to budget for.

## Closing the loop after a run

```bash kno-run scenario=power-analysis stage=attribute
kno eval inspect --evals cases.jsonl --holdout-frac 0.2 \
  --value-run-id pa-value --db kno.db
```

With `--value-run-id`, the fifth check comes off `unknown` and the command reports what the
run's routing *actually* did rather than what the eval set said it could do:

```
  ✓ routing ran in all-dev mode over 0 clusters
  control arm 39 Cases, minimum detectable harm 0.19 (one-sided 95%) — underpowered
```

Even at 160 Cases the harm test is underpowered, and the tool says so instead of printing a
reassuring bound. That is the honest ceiling on a 160-Case run, reported by the tool rather than
by this page.

## As a CI gate

`kno eval inspect` exits 0 whether zero or five checks are flagged — it is a diagnostic, not a
gate, and a command that failed the build on its own opinion of your eval set would be a command
people stop running. A job that wants a gate picks its own threshold:

```bash
kno eval inspect --evals cases.jsonl --json | jq '.checks_flagged'
```

Gate on `checks_flagged`, or on a specific check's `status`, or on
`behaviors[].separable_effect` against the effect size you actually care about. The last is the
most useful and the least common: if a two-point improvement is what would change your mind, a
set whose separable effect is 0.25 cannot change your mind, and no amount of running it will.

## Next

- [Why not just script this yourself?](why-not-diy.md) — what an interval is for, shown against
  a script that has none.
- [What a run costs](what-it-costs.md) — the other number you need before you start.
- [Value a pool of assets](value-a-pool.md) — reading the deltas once the set is big enough.
