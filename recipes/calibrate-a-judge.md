---
verification: executed
scenario: judge-calibration
stage: calibrate
requires-stages: []
last-verified: 2026-09-04
verified-against: kno v0.2.1
---

# Calibrate a judge

A judged number is only as good as the judge that produced it, and judge error does not average
out: it *shrinks* every effect you measure toward zero. A delta read off an uncalibrated goal is
not a weak measurement — it is a measurement of unknown size, biased in a known direction.

`kno judge calibrate` measures how much.

```bash kno-run scenario=judge-calibration stage=calibrate
kno judge calibrate --replay --goal exact-match --set-name starter --min-kappa 0.60
```

Free, offline, and no API key: the calibration set is built into the binary, and `--replay` — the
default — makes no provider call at all.

```
Calibration: exact-match against starter v1
  source     computed locally (this goal calls no model)
  records    60 scored, 0 errored

  kappa              0.867        95% CI [0.731, 0.967] (bootstrap-percentile)
  raw agreement      0.933        a constant judge scores 0.533 on this set
  sensitivity        0.875        of the records humans passed
  specificity        1.000        of the records humans failed
  marginals          judge 0.467  humans 0.533
  inter-human kappa  0.900        the ceiling: a judge cannot beat its own labelers

  PASS
  kappa 0.867, 95% CI [0.731, 0.967], entirely at or above the 0.60 floor

  4 record(s) disagree. --show-disagreements prints them.
```

That is not an illustration. Every line is asserted against the real binary by
`scenarios/judge-calibration/expected/quotations.json`, and the figures behind it — kappa, the
interval, the record counts, the set's own content hash — are projected into
`expected/calibrate.json` and compared nightly.

`--set-name` and `--min-kappa` are stated rather than left to their defaults. The set decides
which records are scored and the floor decides the verdict, so a run whose defaults moved would
change every number above without any command changing.

## What this build can calibrate

`exact-match`, and nothing else. **This release ships the harness, the calibration set and the CI
gate — it does not ship a judge.** The gate is here first on purpose: it means the first judge
prompt arrives with a threshold already pointed at it, rather than the threshold arriving later
and grandfathering whatever happened to ship.

`kno doctor` prints what this build can name.

## Reading the report

Every number kappa hides is on the same screen as kappa, because a single scalar cannot say
*which way* a judge is wrong, and which way is what a prompt edit needs to know.

- **kappa** is the gated statistic, with a percentile-bootstrap interval over records. It is the
  factor by which this judge attenuates every delta you measure through it.
- **raw agreement** is printed and never gated on, with the score a constant judge would get
  beside it. On a set that is 85% "good", answering "good" every time scores 0.85 here and
  kappa 0. That pairing is the entire reason raw agreement is not the headline.
- **sensitivity and specificity** are the direction of the error. A judge that never says "fail"
  is the expensive failure mode and it is invisible in kappa alone. Above, `specificity 1.000`
  against `sensitivity 0.875` says this goal never wrongly passes a record the humans failed,
  and sometimes wrongly fails one they passed — which is the safe direction, and worth knowing
  rather than assuming.
- **inter-human kappa** is the ceiling. A judge cannot be held to an agreement its own labelers
  do not reach; when the ceiling is below the floor, the verdict blames the labels and not the
  judge — see below, where it does.

[What a judge's kappa claims](https://github.com/uknoAI/kno/blob/main/docs/what-the-numbers-mean.md#what-a-judges-kappa-claims)
has the derivation of the 0.60 floor, what it costs you, and the prevalence table behind the
set's balance rule.

## Fix a prompt against the disagreements

```bash kno-run scenario=judge-calibration stage=disagreements
kno judge calibrate --replay --goal exact-match --set-name starter --min-kappa 0.60 \
  --show-disagreements
```

```
  Disagreements
  RECORD  HUMAN  JUDGE  RATIONALE
  al-000  pass   fail   output did not match the expected answer
```

This is the artifact that makes a prompt edit a directed act instead of a guess: every record the
judge and the humans differ on, with the human verdict, the judge verdict, and the judge's own
rationale. Edit the prompt, re-run, and see which records moved.

## Two ways to fail, and they blame different things

The floor is a threshold you choose, and raising it produces two genuinely different refusals.
Both exit 1. Both print `INDETERMINATE`. They mean opposite things, and confusing them sends you
to fix the wrong artefact.

### The interval straddles the floor

```bash kno-run scenario=judge-calibration stage=straddle
kno judge calibrate --replay --goal exact-match --set-name starter --min-kappa 0.88
```

```
  INDETERMINATE
  kappa 0.867, 95% CI [0.731, 0.967], straddles the 0.88 floor
  fix: the set is too small to decide. Add records. "We cannot tell" is not "it
       is fine", which is why this fails rather than passing
```

Kappa is 0.867 and the floor is 0.88, but the point estimate is not the decision — the interval
is, and it *contains* the floor. The judge might clear it and might not; on sixty records there
is no way to tell.

**That fails rather than passes**, and the reason is the whole design. A gate that passed here
would be reporting the absence of evidence as evidence of adequacy, which is how a quality gate
becomes decorative. The fix is more records, not a lower floor.

### The labels do not agree with each other

```bash kno-run scenario=judge-calibration stage=ceiling
kno judge calibrate --replay --goal exact-match --set-name starter --min-kappa 0.95
```

```
  INDETERMINATE
  the labels do not agree with each other: inter-human kappa is 0.900, below the 0.95 floor
  fix: this is a statement about the SET, not about the judge. Adjudicate the
       records the labelers split on, or sharpen the rubric they are labeling
       against
```

Now the verdict has stopped being about the judge at all. Inter-human kappa is 0.900, so
demanding 0.95 asks the judge to be more consistent than the humans it is copying. No prompt edit
can deliver that, and the tool says which of the two to go and fix.

This is the check that stops a calibration gate turning into a treadmill: without it, a team that
raised the floor past its own labelers' agreement would keep editing prompts against a number
that was never reachable.

## Gate it in CI

```bash
kno judge calibrate --replay --all --baseline judge/calibration.baseline.json
```

Two things are gated at once, and a failure names which:

- **the floor** — kappa ≥ 0.60, which is derived rather than borrowed;
- **the ratchet** — kappa may not drop against `judge/calibration.baseline.json` by more than the
  *paired* bootstrap interval on the difference. Paired, because both runs judge the identical
  records; comparing two independent intervals throws that pairing away and is far too
  permissive.

Exit 0 is PASS. Exit 1 is FAIL **or INDETERMINATE** — both stages above exit 1, and the scenario
asserts that they do. Exit 2 is a live run stopped by its budget cap.

`--json` carries the same data as the page:

```bash
kno judge calibrate --json | jq -r '.verdict, .calibrations[0].kappa'
```

Every statistic in it carries at most four decimal places. That is the contract
([ADR-0006](https://github.com/uknoAI/kno/blob/main/docs/adr/0006-the-json-contract.md) rule 6),
not a formatting accident: a bootstrap over a few dozen records does not carry seventeen
significant digits, and the tail digits of one that printed them would differ between an arm64
and an amd64 machine. Round your own comparisons to the same place.

Write a baseline with `--write-baseline`, and review the diff like code.

## Change a prompt

A prompt edit changes the prompt's hash, so the recorded judge responses no longer apply and the
gate says so:

```
no recorded judge responses for prompt 4f0a2c9d1b7e — run 'make record-calibration'
```

Re-record them under a cost cap, then commit the fixtures and the regenerated baseline together:

```bash
KNO_MAX_COST_USD=2.00 make record-calibration
```

A change that *raises* kappa passes: the ratchet is one-sided. A change that lowers it
deliberately — a broader rubric, a cheaper judge model traded against agreement — passes by
committing the new baseline in the same PR and saying in the body what was traded. Same
convention as `.coverage-baseline`, and the same instruction: review the diff like code.

## Add records

Contributing calibration records is a real on-ramp, and it exists before any judge does. A record
is one line of `judge/testdata/calibration/<set>/records.jsonl` in `uknoAI/kno`:

```json
{"id":"al-000","case":{"input":"...","expected":"...","rubric":"..."},
 "response":{"output":"..."},
 "labels":[{"labeler_id":"labeler-a","value":1,"passed":true},
           {"labeler_id":"labeler-b","value":1,"passed":true}],
 "adjudicated":{"labeler_id":"adjudicator","value":1,"passed":true},
 "provenance":{"source":"synthetic"}}
```

The loader refuses a set that breaks any of these, naming which:

- **at least two independent labels** per record, plus an adjudicated reference verdict. A record
  labeled by one person is one person's judgement, which is the thing the set exists to hold a
  judge to.
- **a minority class of at least 40%**, so kappa is not depressed by prevalence.
- **provenance `authored` or `synthetic`**. The set is public and permanent, and traces are
  customer data: there is no spelling here for a record harvested from a real deployment. No PII,
  no credentials, no content under a license that forbids redistribution.

Re-attest the set in the same commit:

```bash
make update-calibration-manifest
```

One caveat, stated because it does not go away: **a public calibration set is contaminated for
training purposes the day it is published.** Any model released afterwards may have seen it. It
is a regression instrument — it detects a prompt change making things worse — not evidence that a
judge generalizes.

## What is executed here and what is not

The four calibration commands on this page are run by CI against the released binary, twice,
byte-compared, with their output and their `--json` documents checked against committed
expectations — including that the two failing stages **still exit 1**, since a gate that quietly
stopped refusing would leave this page green while saying the opposite.

Not executed: everything involving a model-backed judge. The `--baseline` ratchet against a
committed file, the prompt-hash invalidation, `make record-calibration`, and `--live` are real
and are described from the command's own surface, but this build ships no judge, so nothing here
can exercise them. When the first judge prompt lands, they become executable and this page should
grow the stages to match.

## See also

- [`scenarios/judge-calibration/README.md`](../scenarios/judge-calibration/README.md) — the
  scenario, and why the loop runs after the calibration rather than before it
- [Check whether your evals can attribute anything](check-your-evals.md) — the same posture one
  layer down: measure whether the instrument can support the claim, before making it
- [What the numbers mean](https://github.com/uknoAI/kno/blob/main/docs/what-the-numbers-mean.md)
  — what each figure claims and does not
