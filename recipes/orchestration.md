---
verification: manual
owner: "@kno-maintainers"
last-manual-verification: 2026-08-31
credentials: [OPENAI_API_KEY]
---

# Schedule it: Airflow, Dagster, and what to fail the run on

Kno is a CLI over files and a SQLite store. There is no daemon, no server, and no service to
run — which means orchestrating it is orchestrating a sequence of commands with a shared
`--db`, and the interesting decisions are about **idempotency, gating and retention** rather
than about integration.

## Why this page is `manual`

The `kno` command shapes on this page *are* checked against the released binary, nightly — the
flag check runs on every page regardless of tier. What the tier declares is what the **page**
claims, and this page's subject is a DAG: tasks, retries, scheduling and secrets configured in
somebody else's system. "The flags check out" says almost nothing about whether that DAG works,
so the page does not imply that it does.

This is the same call the [n8n page](n8n.md) makes, for the same reason.

## The shape

```
              ┌─ kno mine ──────┐
sources ──────┤                 ├──→ cases.jsonl ──→ kno eval inspect   ← gate here, free
              └─ warehouse SQL ─┘                 │
                                                  ├──→ kno baseline     ← a score
                                                  ├──→ kno value        ← the expensive stage
                                                  ├──→ kno select       ← a portfolio, or a refusal
                                                  ├──→ kno export       ← a manifest ingestion consumes
                                                  └──→ kno purge        ← retention, on its own schedule
```

Three properties make this schedule cleanly, and they are worth naming because they are what an
orchestrator actually needs:

- **Every stage is a process that reads files and a store and exits.** No long-lived state, no
  service discovery, no health check.
- **Run ids are yours to choose.** `--run-id` is the correlation key for every trace, score and
  event, and a run id you named is a task instance you can find. Generated ids are a join
  against timestamps.
- **Interrupted work is resumable.** `--resume` continues from the checkpoint rather than
  starting over, and exit codes `2` and `4` mean "stopped, resumable" rather than "failed" —
  see [the exit-code contract](ci-gate.md).

## Airflow

The run id is the piece to get right. Deriving it from the logical date makes each task instance
idempotent: a re-run of the same interval writes to the same run, so a retry resumes rather than
duplicating.

```python
from airflow.decorators import dag, task
from airflow.operators.bash import BashOperator
import pendulum

KNO_DB = "/data/kno/kno.db"

@dag(
    schedule="0 4 * * *",
    start_date=pendulum.datetime(2026, 8, 1, tz="UTC"),
    catchup=False,
    default_args={"retries": 2},
)
def kno_nightly():
    mine = BashOperator(
        task_id="mine",
        bash_command=(
            "kno mine --logs /data/transcripts --out /data/kno/cases.jsonl "
            "--mode resolution --format auto --min-cases 200"
        ),
    )

    inspect = BashOperator(
        task_id="inspect",
        bash_command=(
            "kno eval inspect --evals /data/kno/cases.jsonl --json "
            "| tee /data/kno/inspect.json "
            "| jq -e '.checks_flagged <= 1'"
        ),
    )

    baseline = BashOperator(
        task_id="baseline",
        bash_command=(
            "kno baseline --evals /data/kno/cases.jsonl "
            "--agent openai:gpt-4.1 --max-output-tokens 512 --goal exact-match "
            "--holdout-frac 0.2 --max-cost-usd 10.00 "
            f"--db {KNO_DB} --run-id nightly-{{{{ ds }}}}-baseline --yes --resume"
        ),
    )

    mine >> inspect >> baseline

kno_nightly()
```

Three details that are not decoration:

- **`--min-cases 200` on `mine`.** It fails the command when the export produced fewer, which
  turns "the source was empty this week" into a red task rather than a silently shrinking eval
  set. Without it, a broken upstream export looks like a successful mining run.
- **`--resume` on the retried task.** Airflow's `retries: 2` re-runs the whole command; with a
  stable `--run-id` and `--resume`, the second attempt continues rather than paying for the
  first attempt's calls again.
- **`--max-cost-usd` on every paid stage.** A retry is a second chance to spend. See [what a run
  costs](what-it-costs.md), and note that the cap is per invocation — two retries can spend the
  cap twice, so set `KNO_MAX_COST_USD` in the worker's environment as the ceiling the command
  line cannot raise.

## Dagster

Dagster's asset graph is closer to the shape of the loop, because each stage genuinely produces
an artefact the next one consumes:

```python
import json, subprocess
from dagster import asset, AssetExecutionContext, MetadataValue, Output

KNO_DB = "/data/kno/kno.db"

def kno(context, *args):
    result = subprocess.run(["kno", *args], capture_output=True, text=True)
    context.log.info(result.stderr)
    if result.returncode in (2, 4):
        context.log.warning("stopped or interrupted; resumable")
    elif result.returncode != 0:
        raise RuntimeError(f"kno {args[0]} failed: {result.stderr}")
    return result

@asset
def eval_cases(context: AssetExecutionContext) -> Output[str]:
    path = "/data/kno/cases.jsonl"
    kno(context, "mine", "--logs", "/data/transcripts", "--out", path,
        "--mode", "resolution", "--format", "auto", "--min-cases", "200")
    doc = json.loads(kno(context, "eval", "inspect", "--evals", path, "--json").stdout)
    return Output(
        path,
        metadata={
            "cases": doc["cases"]["total"],
            "holdout": doc["cases"]["holdout"],
            "checks_flagged": doc["checks_flagged"],
            "weakest_behavior": MetadataValue.float(
                max(b["separable_effect"] for b in doc["behaviors"]) if doc["behaviors"] else 0.0
            ),
        },
    )
```

Putting `checks_flagged` and the worst `separable_effect` into asset metadata is the single
highest-value line in that file: it turns "is our eval set still good enough" into a number
plotted over time in the asset catalogue, rather than something nobody looks at until a result
is surprising.

## Gate on the free command, not the paid one

`kno eval inspect` makes no LLM call and writes nothing, so it is the cheapest possible place to
fail the run — and the failure it catches (an eval set that shrank, lost its tags, or lost its
holdout) is the one that makes every downstream number meaningless.

It **exits 0 whether zero or five checks are flagged**. That is deliberate: a diagnostic that
failed the build on its own opinion of your eval set would be a diagnostic people stop running.
A job that wants a gate picks its own threshold:

```bash
kno eval inspect --evals cases.jsonl --json | jq -e '.checks_flagged <= 1'
```

Gate on `checks_flagged`, on a named check's `status`, or — best — on
`behaviors[].separable_effect` against the effect size that would actually change your mind. See
[how many Cases do I need](power-and-sample-size.md).

## Retention is a schedule, not a cleanup script

`kno purge` removes stored agent output and judge rationales for a run, and **keeps** the
completion records, costs and scores — so a purged run is still resumable and still reportable.
That is what makes it safe to schedule aggressively:

```bash
kno purge --run-id "nightly-2026-06-01-value" --db /data/kno/kno.db --yes
```

Kno treats stored traces as customer data. A pipeline that mines production transcripts into an
eval set has moved customer content into `kno.db`, and the retention window for that content is
a decision your organisation has already made about the source system — the purge schedule is
where you honour it. [Delete stored conversation content](retention.md) is the full page.

Purge has no `--json`, so branch on its exit code rather than parsing its output.

## Two things to decide before you ship the DAG

**Where `kno.db` lives.** It is a SQLite file and it is the correlation key for everything —
resume, `--baseline-run-id`, `--value-run-id`, `report`. A store on ephemeral task-local disk
means every stage after `baseline` cannot find its predecessor. Put it on a volume that outlives
a task, and treat it as state you back up.

**Whether stages share a store across concurrent runs.** Two runs writing one SQLite file
concurrently is a question about SQLite, not about Kno, and this page does not claim an answer.
The conservative arrangement is one store per pipeline and a concurrency limit of one on the
tasks that write it; if you want parallelism, give each run its own `--db` and merge nothing.

## Next

- [Point Kno at the data you already have](from-your-warehouse.md) — producing `cases.jsonl`
  from a warehouse or object storage.
- [Gate a deploy on Kno in CI](ci-gate.md) — the exit-code contract in full.
- [Delete stored conversation content](retention.md) — what purge keeps and why.
