---
verification: executed
scenario: power-analysis
stage: validate
requires-stages: [baseline, value, select]
last-verified: 2026-09-22
verified-against: kno v0.2.2
---

# Validate a portfolio against the untouched holdout

Every other stage works on the dev split. `kno validate` is the one that opens the holdout — the
Cases sealed before the first agent call, that nothing has read since — and measures the chosen
Portfolio against them.

It is the stage the whole discipline is for. A number produced by selecting on some Cases and
then reporting on those same Cases is a fit, not a measurement; the holdout is what makes the
final figure a claim about Cases the selection never saw.

```bash kno-run scenario=power-analysis stage=validate
kno validate --select-run-id pa-select --evals cases.jsonl --pool pool.jsonl \
  --agent fake: --goal exact-match --holdout-frac 0.2 --concurrency 1 \
  --db kno.db --run-id pa-validate --yes
```

**`--holdout-frac` must match the value the pipeline was measured under.** The flag's own help
says so. It decides which Cases are held back, so a validate run on a different fraction opens a
different holdout from the one the rest of the run sealed, and reports a number about a
population nothing else describes. It is the one flag here where a mismatch is silent and wrong
rather than loud.

## The most important thing it does is refuse

```
Validate run pa-validate

  The Portfolio selected nothing this stage can measure, so there is nothing to validate.
  No agent call was made and the holdout was not opened — it stays untouched for the Portfolio that earns it.
```

That is the real output of the verified run, asserted by
`scenarios/power-analysis/expected/quotations.json`. The Portfolio was empty — all three Assets
rejected as `no-effect` — so there was nothing to measure, and **the holdout was not opened.**

Read that last clause as a mechanism rather than a courtesy. The holdout is a finite resource:
every time you measure against it, it stops being a set of Cases nothing has seen. A stage that
opened it to discover there was nothing to do would burn it for no information. So it does not,
and the `--json` document says so in a field a script can gate on:

```json
{
  "nothing_to_validate": true,
  "llm_calls": 0,
  "holdout_uses": 0,
  "min_holdout": 20
}
```

`holdout_uses: 0` is the proof. Not "we did not report a number", but "the holdout has been used
zero times and is still whole".

## Using it a second time

From `kno validate --help`:

```
      --allow-repeat-holdout          measure a second portfolio against a holdout that has already been used; the count is recorded and printed
```

A second measurement against the same holdout is not forbidden — sometimes it is the only thing
available — but it is not free either, and the refusal is the default. Each use makes the set a
little less untouched, because the decisions you make after seeing the first result are informed
by it. The flag exists so that using it again is a deliberate act with a number attached, rather
than something a script does by repetition.

`holdout_uses` in the `--json` is that number. Gate on it if you care.

## Reading a real verdict

When a Portfolio has entries, the fields that carry the answer are:

| Field | What it means |
|---|---|
| `holdout_case_count` | how many sealed Cases the measurement ran on |
| `measured_case_count` | how many were actually scored, after drops |
| `holdout_underpowered` | the holdout is below `min_holdout` (20) — the interval will not support a conclusion |
| `verdict` | the outcome; `unspecified` when nothing was measured |
| `interaction_penalty_detected` | the Assets helped individually and hurt together |

`interaction_penalty_detected` is the one worth knowing exists before you need it. `kno value`
measures each Asset on its own, and a Portfolio is several of them in one prompt. Assets that
each helped can collectively help less than their sum, or hurt — a policy contradicting another
policy, two examples pulling the same answer in different directions. Nothing at the value stage
can see that, because nothing at the value stage puts them together. Validate is where it
surfaces.

## Why this page's run shows no numbers

The scenario behind it uses `fake:`, which answers every Case with exactly what the Case expects.
No injected Asset can move that score, so every delta is exactly zero, every Asset is rejected,
and the Portfolio is empty. That is the documented ceiling on every committed scenario here, and
it is why this page can verify the **refusal** end to end and cannot verify a verdict.

What is verified is worth more than it looks: that the stage runs, that it declines to open the
holdout when there is nothing to measure, and that it says so in both its rendered output and a
JSON field. Those are properties of the mechanism rather than of any model, and they hold
identically against a paid provider.

## Next

- [How many Cases do I need?](power-and-sample-size.md) — `holdout_powered` is the check that
  predicts whether this stage will be able to conclude anything, and it costs nothing to run.
- [Choose a portfolio under budget](select-a-portfolio.md) — the stage that decides what arrives
  here.
- [Read the whole story with `kno report`](read-the-whole-story.md) — where the holdout caveat is
  mandatory.
