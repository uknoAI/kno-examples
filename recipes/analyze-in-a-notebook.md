---
verification: flags-only
owner: "@kno-maintainers"
verified-against: kno v0.2.0
last-manual-verification: 2026-08-31
---

# Read the results in a notebook

Kno is a CLI and every stage speaks `--json`. If you live in a notebook, that is the whole
integration: run the stage, load the document, and the analysis is pandas from there.

There is no Python SDK. You do not need one for this — `--json` on stdout into `pd.json_normalize`
is two lines — and pretending otherwise would be the wrong reason to add a dependency.

## Why this page is not `executed`

The `kno` commands here are checked against the released binary's own flag surface, like every
page in this repository. **The Python is not run by anything.** So the page claims what it can
claim and no more.

What is stronger than it looks: every field name below appears in a committed projection under
[`scenarios/power-analysis/expected/`](../scenarios/power-analysis/expected/), which CI compares
against real `--json` output nightly. If a release renames `separable_effect` or drops
`valuations[].low`, that scenario goes red — so the *contract* this page reads against is
verified even though this page's code is not. A field named here that CI does not assert is a
field you should treat as an implementation detail.

## The three documents worth loading

| Command | What the document holds |
|---|---|
| `kno eval inspect --evals cases.jsonl --json` | per-behaviour power, before you spend anything |
| `kno value … --json` | one row per Asset: delta, interval, control bound, sample counts |
| `kno select … --json` | the Portfolio, and the rejection log with a reason per Asset |

## Power, before you spend anything

```python
import json, subprocess
import pandas as pd

def kno(*args):
    """Run one Kno stage and parse its --json document."""
    out = subprocess.run(["kno", *args, "--json"], capture_output=True, text=True, check=True)
    return json.loads(out.stdout)

doc = kno("eval", "inspect", "--evals", "cases.jsonl", "--holdout-frac", "0.2")

power = pd.json_normalize(doc["behaviors"])[["tag", "dev_cases", "separable_effect", "status"]]
power.sort_values("separable_effect", ascending=False)
```

```
                 tag  dev_cases  separable_effect        status
3      access-policy          2            6.3531  underpowered
0          freshness          3            1.7566  underpowered
1  metric-definition          3            1.7566  underpowered
2        sql-dialect          3            1.7566  underpowered
```

Those are the real values `kno eval inspect` reports for the twelve-Case eval set in
[`scenarios/power-analysis`](../scenarios/power-analysis/README.md), which CI asserts nightly.
The pandas *formatting* around them is illustrative — your column widths and index will differ.

`separable_effect` is the smallest true improvement that behaviour's dev Cases could tell apart
from noise at 95%. A score lives in [0, 1], so **anything above 1.0 means nothing is
detectable**. That is the first thing to plot and the first thing to act on; see [how many Cases
do I need](power-and-sample-size.md).

```python
doc["checks_flagged"], [c["name"] for c in doc["checks"] if c["status"] == "flagged"]
```

## Deltas with their intervals

```python
doc = kno("value", "--evals", "cases.jsonl", "--pool", "pool.jsonl",
          "--baseline-run-id", "pa-baseline", "--agent", "fake:",
          "--db", "kno.db", "--run-id", "pa-value", "--yes")

df = pd.json_normalize(doc["valuations"])
df["width"] = df["high"] - df["low"]
df["crosses_zero"] = (df["low"] <= 0) & (df["high"] >= 0)
df[["asset_id", "delta_goal", "low", "high", "width", "crosses_zero", "n_pairs", "n_dev"]]
```

The columns that matter, in the order you should read them:

- **`low` and `high` before `delta_goal`.** The point estimate is the least informative number in
  the row. An interval that crosses zero is an Asset that has not earned its place regardless of
  which side of zero the point landed on.
- **`control_low`** is the one-sided bound on *harm*: how much worse injecting this Asset could
  have made the behaviours it was not supposed to touch. A near-zero delta with a wide
  `control_low` is not a safe Asset, it is an unmeasured one.
- **`n_pairs` against `n_dev`.** `n_pairs` is how many Cases this Asset was actually measured on,
  which routing and sampling decide; `n_dev` is the routable pool. A small `n_pairs` explains a
  wide interval without anything being wrong.
- **`not_measured`** carries a rejection reason when an Asset was skipped entirely. A row is not
  a measurement just because it exists.

## A forest plot is the right default

One row per Asset, the interval as a horizontal bar, a line at zero. It makes the only question
that matters — does this bar cross zero — a visual one:

```python
import matplotlib.pyplot as plt

fig, ax = plt.subplots(figsize=(7, 0.5 * len(df) + 1))
ax.errorbar(
    df["delta_goal"], range(len(df)),
    xerr=[df["delta_goal"] - df["low"], df["high"] - df["delta_goal"]],
    fmt="o", capsize=4,
)
ax.axvline(0, linestyle="--", linewidth=1)
ax.set_yticks(range(len(df)), df["asset_id"])
ax.set_xlabel("delta (95% CI, positive = goal direction)")
fig.tight_layout()
```

Plot `low` and `high` from the Value document if you are describing what was measured. Plot the
**`select` document's** intervals if you are describing what was decided — they are wider,
because `select` applies a Bonferroni correction for the number of Assets you asked about, and
the corrected interval is the one the keep/reject decision used.

## The decision, and the rejection log

```python
doc = kno("select", "--value-run-id", "pa-value", "--pool", "pool.jsonl",
          "--db", "kno.db", "--run-id", "pa-select")

selected = pd.json_normalize(doc["selected"] or [])
rejected = pd.json_normalize(doc["rejected"] or [])
rejected[["asset_id", "reason", "detail"]]
```

```
         asset_id     reason                                              detail
0   access-policy  no-effect  delta +0.0000, CI [-0.0440, +0.0440] crosses zero
1    dialect-note  no-effect  delta +0.0000, CI [-0.0440, +0.0440] crosses zero
2 metrics-glossary no-effect  delta +0.0000, CI [-0.0440, +0.0440] crosses zero
```

`selected` is `null` rather than `[]` when nothing was chosen, which is why the `or []` is there.
An empty Portfolio is a result, not an error.

**`reason` is the column to group by.** `no-effect` means an interval was formed and it contained
zero — a measurement. `underpowered` means no interval could be formed at all — the refusal to
pretend a measurement happened. Charting those as one "rejected" bucket erases the distinction
that [two whole scenarios](../scenarios/underpowered-eval/README.md) exist to keep visible.

## Keeping a history

Run ids are the correlation key, so a table of runs is a `concat` over documents:

```python
frames = []
for run in ["nightly-2026-08-28", "nightly-2026-08-29", "nightly-2026-08-30"]:
    doc = kno("report", "--value-run-id", run, "--db", "kno.db")
    frame = pd.json_normalize(doc, record_path=["valuations"])
    frame["run_id"] = run
    frames.append(frame)

history = pd.concat(frames, ignore_index=True)
```

Name your runs (`--run-id nightly-2026-08-30`) rather than letting them be generated, or this
is a join against timestamps.

## Two things not to do

**Do not re-implement the statistics on top of the raw scores.** The intervals are paired, the
correction depends on how many Assets were in the Pool, and the control arm is a separate
sample. A notebook that recomputes a t-test over `delta_goal` is
[the hand-rolled ablation](why-not-diy.md) with a nicer font.

**Do not run a paid stage from a notebook cell without a cap.** A cell is easy to re-run by
accident. `--max-cost-usd` and `--max-calls` belong in the `kno()` helper's default arguments,
not in the cell you are editing; see [what a run costs](what-it-costs.md).

## Next

- [How many Cases do I need?](power-and-sample-size.md) — the numbers behind the first table.
- [Value a pool of assets](value-a-pool.md) — what each column means before you plot it.
- [What the numbers mean](https://github.com/uknoAI/kno/blob/main/docs/what-the-numbers-mean.md)
  — read before acting on a delta.
