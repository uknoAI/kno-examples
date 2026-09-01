# `judge-calibration`

The scenario that runs the loop in the order the loop is actually valid in: **calibrate the goal
first, then measure with it.**

Every number Kno reports through a judged goal is bounded by how well that judge agrees with a
human. Judge error does not average out — it *shrinks* every effect you measure toward zero — so
a delta read off an uncalibrated goal is not a weak measurement, it is a measurement of unknown
size in a known direction. `kno judge calibrate` is what turns that from a worry into a number.

```
scenarios/judge-calibration/
  evals/cases.jsonl      24 Cases about an invented public library, across 4 behaviors
  pool/pool.jsonl        3 candidate notices
  run.sh                 seven stages: four calibration, then baseline, value, select
  expected/              one projected JSON document per stage, plus quotations.json
  DATA-PROVENANCE.md     who wrote this data, and the assertion that it is synthetic
```

Run it:

```sh
sh scenarios/judge-calibration/run.sh /tmp/jc
```

It needs a released `kno` on `PATH` and nothing else. Nothing here spends: `--replay` is the
default and makes no provider call, the calibration set is built into the binary, and the loop
stages run against `fake:`.

## What the calibration reports

```
Calibration: exact-match against starter v1
  source     computed locally (this goal calls no model)
  records    60 scored, 0 errored

  kappa              0.867        95% CI [0.731, 0.967] (bootstrap-percentile)
  raw agreement      0.933        a constant judge scores 0.533 on this set
  sensitivity        0.875        of the records humans passed
  specificity        1.000        of the records humans failed
  inter-human kappa  0.900        the ceiling: a judge cannot beat its own labelers

  PASS
  kappa 0.867, 95% CI [0.731, 0.967], entirely at or above the 0.60 floor
```

Every one of those lines is asserted against the real binary by `expected/quotations.json`.

The reason all of it is on one screen, rather than kappa alone, is that a single scalar cannot
say *which way* a judge is wrong — and which way is the only thing a prompt edit can act on. A
judge that never says "fail" is the expensive failure mode, and it is invisible in kappa. Here
`specificity 1.000` with `sensitivity 0.875` says the opposite: this goal never wrongly passes a
record the humans failed, and sometimes wrongly fails one they passed.

`raw agreement 0.933` is printed and never gated on, with `a constant judge scores 0.533 on this
set` beside it. That pairing is the whole reason raw agreement is untrustworthy: on a set that is
85% "good", answering "good" every time scores 0.85 and kappa 0.

## Two ways to fail, and they blame different things

The scenario runs the gate three times at three floors, and **two of them are meant to exit 1**.
`run.sh` declares which in an `expect_exit_1=` line and fails the scenario if one of them exits
0 — a gate that silently stopped refusing would leave every stage green while the README went on
claiming the opposite.

| Stage | `--min-kappa` | Verdict | What it blames |
|---|---:|---|---|
| `calibrate` | 0.60 | `PASS` | — |
| `straddle` | 0.88 | `INDETERMINATE` | **the set is too small to decide** |
| `ceiling` | 0.95 | `INDETERMINATE` | **the labels, not the judge** |

Both failures print `INDETERMINATE` and both exit 1, and they mean different things — the same
shape of distinction as [`no-effect` versus
`underpowered`](../underpowered-eval/README.md), one layer up.

**`straddle`** raises the floor to 0.88. Kappa is 0.867 with an interval of [0.731, 0.967], so
the interval *contains* the floor: the judge might clear it and might not, on this many records
there is no way to tell. That fails rather than passes, and the fix says why —

```
  fix: the set is too small to decide. Add records. "We cannot tell" is not "it
       is fine", which is why this fails rather than passing
```

An implementation that passed here would be reporting the absence of evidence as evidence of
adequacy, which is the single most common way a quality gate becomes decorative.

**`ceiling`** raises the floor to 0.95, above the inter-human kappa of 0.900. Now the verdict
stops being about the judge at all:

```
  the labels do not agree with each other: inter-human kappa is 0.900, below the 0.95 floor
  fix: this is a statement about the SET, not about the judge. Adjudicate the
       records the labelers split on, or sharpen the rubric they are labeling
       against
```

A judge cannot be held to an agreement its own labelers do not reach. Demanding 0.95 of a judge
whose humans manage 0.900 is asking it to be more consistent than the thing it is copying, and
the tool says which of the two to go and fix.

## Why the loop runs afterwards

The last three stages are an ordinary `baseline` → `value` → `select` over 24 Cases and three
candidate notices, with `--goal exact-match` — the goal the four stages before them just
calibrated.

They are here because the ordering is the argument. Calibration is not an appendix to the loop;
it is what entitles you to read the loop's output. A scenario that calibrated a goal and then
never used it would be demonstrating a command. This one demonstrates a sequence.

It also means the Pool is real rather than a file present to satisfy a contract.

## What this scenario does not own

**The calibration set is inside the binary**, not in this directory. `starter v1`, 60 records,
`set_content_sha256` `8f3fcd00…`, authored by the kno maintainers.

So the numbers this scenario asserts — kappa 0.867, the interval, the four disagreements — are
facts about a released `kno`, and they will change if that set changes. That is not a weakness of
the expectations, it is the point of them: a changed calibration set is exactly the event a
reader of this page needs to be told about, and the alternative — asserting nothing about the
numbers — would leave a page full of figures nothing checks.

The `set_content_sha256` is projected in `expected/calibrate.json` for the same reason. A set
whose contents changed while its name and version stayed put would otherwise move every number
here with nothing naming the cause.

## What this does not show

`exact-match` is the only goal this build can calibrate. **This release ships the harness, the
calibration set and the gate — it does not ship a judge.** The gate arriving first is deliberate:
it means the first judge prompt lands with a threshold already pointed at it, rather than the
threshold arriving afterwards and grandfathering whatever happened to ship.

So nothing here exercises a model-backed judge, a prompt hash, a re-recorded fixture, or the
paired ratchet against a committed baseline. Those are real and documented in
[the recipe](../../recipes/calibrate-a-judge.md); they are not executed, and the recipe says
which parts are which.

The loop stages carry the usual ceiling too: `fake:` answers every Case with what the Case
expects, so the score is 1.000 and every delta is exactly zero.
