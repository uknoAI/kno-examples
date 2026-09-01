---
verification: executed
scenario: power-analysis
stage: inspect-160
requires-stages: []
last-verified: 2026-08-31
verified-against: kno v0.1.4
---

# Check whether your evals can attribute anything

Kno's promise is *this Asset moved this outcome by this much*. Every mechanism that delivers it
is bounded by the granularity of your eval set — and until this command existed you discovered
that bound the expensive way: after paying for a baseline and a value run.

`kno eval inspect` reports it first.

```bash kno-run scenario=power-analysis stage=inspect-160
kno eval inspect --evals cases.jsonl --holdout-frac 0.2
```

It reads no model, resolves no credential, creates no Run and writes nothing. It exits 0
whatever it finds. Run it before you spend, and again in CI.

`--holdout-frac` is worth stating even though `0.2` is the default: the fraction decides which
Cases are dev, so it changes every number below.

> One caveat the help text also carries: a remote source (`langsmith:`, `langfuse:`,
> `braintrust:`, `hf:`) *does* call the vendor's API with the vendor's credentials, because
> reading the dataset is the job. "Costs nothing" is a claim about LLM spend.

## What comes back

```
Evals  cases.jsonl
  160 Cases — 133 dev, 27 held back
  4 distinct behaviors (tags)

  Everything below reads your tags as behaviors, because that is what routing
  does. If these tags name something else — priority, source, a date — the
  per-tag numbers and suggestions below do not apply to them. Kno cannot tell
  the difference.

BEHAVIOR           DEV CASES  SEPARABLE EFFECT (two-sided 95%)  STATUS
sql-dialect               34                              0.25  ok
access-policy             33                              0.25  ok
freshness                 33                              0.25  ok
metric-definition         33                              0.25  ok

  ! flagged   ✓ ok   ? unknown   · reported, never flagged
  ✓ 4 distinct behaviors on 133 of 133 dev Cases
  ✓ every behavior has at least 5 dev Cases (core.MinClusterCases)
  ✓ the most common behavior "sql-dialect" is carried by 26% of dev Cases
  ✓ the holdout has 27 Cases (20 is the minimum for a meaningful interval at
    validate)
  ? attribution_observed is unknown: no --value-run-id given

0 of 5 checks flagged.
```

That is not an illustration. Every line above is asserted against the real binary by
`scenarios/power-analysis/expected/quotations.json`, from a 160-Case eval set CI reads nightly.
If Kno reformats one of them, this page goes red and the prose is what changes.

Read it in three passes.

**The behavior table** is what routing will see. A behavior is a normalized tag — `Refunds`,
`refunds` and `" refunds "` are one behavior, and the header line reports the collapse when it
happens. `SEPARABLE EFFECT` is the smallest effect that many dev Cases can distinguish from
zero, two-sided at 95%. It is a bound computed from the sample size alone, not an estimate from
your data, which is why it is printable before you have measured anything. Thirty-three dev
Cases separate 0.25 and nothing smaller; only you know whether 0.25 is an effect you would act
on. [How many Cases do I need?](power-and-sample-size.md) is the same command run at three sizes,
so you can see what the number does as the set grows.

**The findings** carry four markers: `!` flagged, `✓` ok, `?` unknown, and `·` for numbers that
are reported and will never be flagged. The multi-behavior share is the `·` line — there is no
principled threshold for it anywhere in Kno, and a tool built to refuse invented cut-offs cannot
flag on one.

**The headline** is a count, not a grade: `0 of 5 checks flagged`. There is no
`Attribution quality: MODERATE` line, deliberately — a single word blending five checks with
five different fixes is the anti-pattern this command exists to find.

## The one thing it cannot know

A behavior, to the engine, is a normalized tag. That is not an approximation: `cluster()` groups
failed dev Cases by tag, routing overlaps an Asset's tags against them, and `ComputeGaps` reports
one verdict per tag.

But `Case.tags` is free-form. If your tags are `p0`, `flaky`, `regression-2024` and
`source:zendesk`, this command will report four distinct behaviors, attach a specific
separable-effect number to each, and suggest splitting or merging them. It cannot tell that
taxonomy from a real one. That is why the conditional above the table is printed on every run,
and why the suggestions are introduced by "If these tags are behaviors you would fix separately".

This is the trap a warehouse export walks into, because the columns that exist are the ones that
got tagged — `priority`, `channel`, `created_month`. See [point Kno at the data you already
have](from-your-warehouse.md#tags-are-the-load-bearing-column).

## After a run

```bash kno-run scenario=power-analysis stage=attribute
kno eval inspect --evals cases.jsonl --holdout-frac 0.2 \
  --value-run-id pa-value --db kno.db
```

**That block is not standalone.** It reads a store the Value run wrote — run
[`scenarios/power-analysis/run.sh`](../scenarios/power-analysis/run.sh) first, or it will find
nothing. The three commands above it need no store at all.

It adds what a recorded Value run's routing actually did: the mode it ran in, each cluster's gap
verdict, how many of the cluster's Cases the baseline confirms as failed, and the control arm's
**one-sided** minimum detectable harm — printed beside the **two-sided** separable effects, both
labeled, because they answer different questions.

```
  ✓ routing ran in all-dev mode over 0 clusters

Observed  value run pa-value, against baseline pa-baseline
  routing mode all-dev, run completed
  control arm 39 Cases, minimum detectable harm 0.19 (one-sided 95%) — underpowered
```

Even at 160 Cases the harm test is underpowered, and the tool says so rather than printing a
reassuring bound.

If the eval file has changed since the run, the observed section is withheld and the check
reports `unknown` with "the eval source has changed since this run". A current tag structure
joined to a stale plan would be a page composed of two different eval sets.

## In CI

The exit code is 0 whether zero or five checks are flagged, so this is safe in a pre-commit hook.
That is deliberate: a diagnostic that failed the build on its own opinion of your eval set is a
diagnostic people stop running. To gate, pick your own threshold from `--json`:

```bash
flagged=$(kno eval inspect --evals cases.jsonl --json | jq '.checks_flagged')
[ "$flagged" -le 1 ] || { echo "eval set regressed"; exit 1; }
```

The `checks[].name` values (`behaviors_declared`, `behaviors_powered`, `behavior_concentration`,
`holdout_powered`, `attribution_observed`) are a stable contract; renaming one is a breaking
change with a CHANGELOG note. `separable_effect` and `min_detectable_harm` each carry an explicit
`sidedness` key so a `jq` pipeline cannot mistake one for the other.

Gating on `behaviors[].separable_effect` against the effect size that would actually change your
mind is more useful than gating on the count, and much less common.

## What to do about each finding

| Finding | The fix |
|---|---|
| No dev Case carries a tag | Tag them. Untagged Cases join no cluster, and if none is tagged, routing measures everything against everything — a more expensive run with no per-behavior attribution. This is what a freshly [mined eval set](mine-your-transcripts.md) looks like. |
| A behavior below 5 dev Cases | Add Cases, or merge it into a behavior you would fix in one place. Below `core.MinClusterCases` a measurement may not testify about the cluster at all. |
| One tag carries most of the set | Split it into the behaviors you would act on separately — this is [section 8's](https://github.com/uknoAI/kno/blob/main/docs/evaluation-design.md) "one giant score" as it manifests in an eval file. |
| The holdout is under 20 | Grow the eval set. `validate` needs a holdout that can carry a meaningful interval. Roughly 100 Cases at a holdout fraction of 0.2. |
| `attribution_observed` unknown | Pass `--value-run-id`, or accept that nothing is claimed about what a run attributed. |

## See also

- [How many Cases do I need?](power-and-sample-size.md) — this command at 12, 40 and 160 Cases,
  and what the extra Cases buy
- [Turn the logs you already have into an eval set](mine-your-transcripts.md) — where three of
  these five checks flag on day one
- [Evaluation design](https://github.com/uknoAI/kno/blob/main/docs/evaluation-design.md) — the
  deep version, and the separable-effect table
- [What the numbers mean](https://github.com/uknoAI/kno/blob/main/docs/what-the-numbers-mean.md)
  — what each figure claims and does not
- [Gate a deploy on Kno in CI](ci-gate.md) — exit codes and `--json` across the whole loop
