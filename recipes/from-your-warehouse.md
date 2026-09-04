---
verification: flags-only
owner: "@kno-maintainers"
verified-against: kno v0.2.1
last-manual-verification: 2026-08-31
credentials: [OPENAI_API_KEY, SNOWFLAKE_PASSWORD]
---

# Point Kno at the data you already have

Kno reads two things: **Cases** (an eval set) and a **Pool** (candidate Assets). Both are files
or dataset references, and neither needs a connector — which is the answer to the first question
a data engineer asks and the reason there is no "Kno warehouse integration" to install.

| | Accepts |
|---|---|
| `--evals` | a JSONL file path, or `langsmith:<dataset>`, `langfuse:<dataset>`, `braintrust:<dataset>`, `hf:<org>/<name>/<config>/<split>` |
| `--pool` | a JSONL file path, `csv:<file>`, `md:<file-or-dir>`, or `hf:<org>/<name>/<config>/<split>:<kind>` |

Everything below is a way of producing one of those from something you already run.

## The two file contracts

**Cases**, one JSON object per line. `id` and `input` are required; `expected` is what a correct
answer looks like; `tags` name the behaviours you would fix separately.

```jsonl
{"id":"auth-01","input":"Which header carries the API key?","expected":"Authorization: Bearer <key>.","tags":["auth"]}
```

The `id` must be **stable across runs**. The dev/holdout split is keyed on it, so an id derived
from a row number moves Cases between halves whenever the query returns a different order, and
every comparison against an earlier run silently stops being a comparison. Derive it from a
primary key, or from a hash of the input — never from `ROW_NUMBER()`.

**Pool as CSV**, a header row then one Asset per row. `id` and `content` are required columns;
`kind` and `tags` are optional, and `tags` entries are separated by **semicolons** because commas
belong to the CSV. An unknown column is a fatal, named error rather than a guess.

```csv
id,content,kind,tags
auth-guide,"Every request carries Authorization: Bearer <key>.",knowledge,auth;security
```

That makes a CSV pool a `SELECT` away, which is the point.

## From Snowflake

```bash
export SNOWFLAKE_PASSWORD='…'

snowsql -o output_format=csv -o header=false -o timing=false -o friendly=false \
  -q "SELECT ticket_id, question, resolution, queue
      FROM analytics.support.resolved_tickets
      WHERE resolved_at >= DATEADD(day, -90, CURRENT_DATE())
        AND resolution IS NOT NULL" \
  | python3 -c '
import csv, json, sys
for row in csv.reader(sys.stdin):
    if len(row) != 4:
        continue
    ticket_id, question, resolution, queue = row
    print(json.dumps({
        "id": f"ticket-{ticket_id}",
        "input": question,
        "expected": resolution,
        "tags": [queue],
    }))
' > cases.jsonl
```

`ticket_id` is the stable id. `queue` becomes the behaviour tag, which is the single most
valuable column in that query — see the caveat under *Tags are the load-bearing column* below.

## From BigQuery

`bq` will emit newline-delimited JSON directly, so the transform is a `SELECT` alias away:

```bash
bq query --nouse_legacy_sql --format=prettyjson --max_rows=100000 \
  'SELECT
     CONCAT("ticket-", CAST(ticket_id AS STRING)) AS id,
     question AS input,
     resolution AS expected,
     [queue] AS tags
   FROM `analytics.support.resolved_tickets`
   WHERE resolved_at >= DATE_SUB(CURRENT_DATE(), INTERVAL 90 DAY)' \
  | jq -c '.[]' > cases.jsonl
```

Alias the columns in SQL rather than renaming them afterwards. The eval set is then a view
definition somebody can read, review and diff, instead of a shell pipeline nobody owns.

## From Postgres

```bash
psql "$DATABASE_URL" -At -c "
  SELECT json_build_object(
    'id',       'ticket-' || t.id,
    'input',    t.question,
    'expected', t.resolution,
    'tags',     json_build_array(t.queue)
  )
  FROM support.resolved_tickets t
  WHERE t.resolved_at > now() - interval '90 days'
" > cases.jsonl
```

## From object storage

A Pool is often a prefix full of Markdown rather than a table:

```bash
aws s3 sync s3://acme-docs/handbook/ ./handbook/
kno value --evals cases.jsonl --pool md:handbook --split-sections \
  --baseline-run-id nightly --agent fake: --db kno.db --run-id nightly-value --yes
```

`md:` takes a file or a directory, reads `.md` recursively in lexical order, and an optional
`---` front-matter block at the top of a file carries that file's `kind` and `tags`.

**`--split-sections` is usually what you want.** Without it, one file is one Asset — so a
forty-page handbook is a single candidate and the answer you get is "the handbook helps" rather
than "the returns section helps". With it, each `## ` heading becomes an Asset, and its id is
the path plus the heading:

```
docs/api.md::Keys
docs/api.md::Rotation
```

Two identical `## ` headings in one file is a fatal error rather than a merge, because two Assets
sharing an id are indistinguishable in every measurement row and every later report.

## From dbt

The natural shape is a model whose columns are already the contract:

```sql
-- models/marts/eval_cases.sql
select
    'ticket-' || ticket_id            as id,
    question                          as input,
    resolution                        as expected,
    array_construct(queue)            as tags
from {{ ref('stg_resolved_tickets') }}
where resolution is not null
  and resolved_at >= dateadd(day, -90, current_date())
```

Then the pipeline step is an unload, and the eval set gets dbt's tests, lineage and freshness
for free — which is a better place for it than a script in someone's home directory.

## From your production transcripts, when there is no table

If the answers live in threads rather than in a resolution column, `kno mine` does the
transformation and puts provenance on every record: see [turn the logs you already
have](mine-your-transcripts.md).

## Check the eval set before it goes anywhere near an agent

```bash
kno eval inspect --evals cases.jsonl --json | jq '{flagged: .checks_flagged, cases: .cases}'
```

This makes no LLM call and writes nothing. Run it as the step immediately after the export, and
gate on `checks_flagged` or on a specific check's status; see [how many Cases do I
need](power-and-sample-size.md). Catching "the export returned 40 rows this week" here costs
nothing, and catching it after a Value run costs whatever the Value run cost.

## Tags are the load-bearing column

Everything per-behaviour that Kno reports — routing, per-cluster attribution, `separable_effect`
per tag — assumes **your tags name behaviours you would fix separately**. Kno cannot tell a
behaviour tag from a priority, a source, or a date, and it says so above every number that
depends on the assumption.

A warehouse export makes this easy to get wrong, because the columns that are *there* are the
ones that got tagged: `priority`, `channel`, `created_month`. Exporting those as tags produces a
per-behaviour table that is arithmetic about nothing. Prefer the column that names what the
question was *about* — queue, component, intent, product area — even when it is dirtier.

## Then the ordinary loop

```bash
kno baseline --evals cases.jsonl --agent openai:gpt-4.1 --max-output-tokens 512 \
  --goal exact-match --holdout-frac 0.2 --db kno.db --run-id nightly-2026-08-31 \
  --max-cost-usd 25.00 --yes
```

`--agent openai:…` bills OpenAI — that is what `OPENAI_API_KEY` in this page's front matter is
for, and it is named because a page about warehouse credentials is exactly where the *other*
credential gets forgotten. [What a run costs](what-it-costs.md) has the arithmetic and the caps.

## What this page is and is not

`flags-only`: CI checks every `kno` command here against the released binary's own flag surface,
nightly. **Nothing checks the `snowsql`, `bq`, `psql`, `aws` or `dbt` halves.** Those are other
people's tools with their own release cycles, and a nightly job holding warehouse credentials in
a public repository's CI is not a trade this project will make.

The `csv:` and `md:` behaviours described above — the required columns, the semicolon tag
separator, the `path::heading` id, the duplicate-heading refusal — were checked by hand against
the version in `verified-against`, on the date in `last-manual-verification`.

## Next

- [Turn the logs you already have into an eval set](mine-your-transcripts.md) — when there is no
  resolution column.
- [Schedule it and gate on it](orchestration.md) — Airflow, Dagster, and what to fail the run on.
- [How many Cases do I need?](power-and-sample-size.md) — before the first paid run.
