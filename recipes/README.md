# Recipes

Task-shaped pages. Each one declares, in front matter, how far it has actually been verified —
see [VERIFICATION.md](../VERIFICATION.md) for exactly what each tier claims and does not claim.

`make render RECIPE=recipes/<name>.md` prints a page's verification block; the website renders
the same block from the same fields.

**Read the tier before you read the page.** ✔ `executed` means CI ran these commands against the
released binary this week and compared the result to committed expectations. `•` means it did
not. `flags-only` and `manual` share one icon and one colour on purpose: with respect to the
question you are actually asking — *does this recipe work* — they are equally unverified, and the
difference between them is a sentence, not a badge.

## Start here, by what you do

Four pages answer the four questions people actually arrive with. All four are free to run —
none of them needs a key.

| If you are asking | Start here | Tier |
|---|---|---|
| **"Why not just script this myself?"** | [Why not just script this yourself?](why-not-diy.md) | ✔ `executed` |
| **"How many Cases do I need?"** | [How many Cases do I need?](power-and-sample-size.md) | ✔ `executed` |
| **"Can my eval set attribute anything?"** | [Check whether your evals can attribute anything](check-your-evals.md) | ✔ `executed` |
| **"Can I trust the judge behind these numbers?"** | [Calibrate a judge](calibrate-a-judge.md) | ✔ `executed` |
| **"What will this cost me?"** | [What a run costs](what-it-costs.md) | ✔ `executed` |
| **"I don't have an eval set."** | [Turn the logs you already have into an eval set](mine-your-transcripts.md) | ✔ `executed` |

And by role, for the rest:

**Building the agent.** [Score your agent for the first time](first-baseline.md) →
[Why not just script this yourself?](why-not-diy.md) →
[Gate a deploy on Kno in CI](ci-gate.md) → [Point Kno at your own provider](your-own-provider.md).

**Analysing the results.** [Calibrate a judge](calibrate-a-judge.md) →
[Check whether your evals can attribute anything](check-your-evals.md) →
[How many Cases do I need?](power-and-sample-size.md) →
[Value a pool of assets](value-a-pool.md) →
[Read the results in a notebook](analyze-in-a-notebook.md) →
[Choose a portfolio under budget](select-a-portfolio.md).

**Wiring the pipeline.** [Point Kno at the data you already have](from-your-warehouse.md) →
[Turn the logs you already have into an eval set](mine-your-transcripts.md) →
[Schedule it: Airflow, Dagster, and what to fail the run on](orchestration.md) →
[Delete stored conversation content](retention.md).

## Core recipes

| Recipe | Tier | What it covers |
|---|---|---|
| [Score your agent for the first time](first-baseline.md) | ✔ `executed` | Getting from an eval file to a baseline, and reading what comes back |
| [Why not just script this yourself?](why-not-diy.md) | ✔ `executed` | The hundred-line ablation, run in CI beside Kno on the same data, and the five differences |
| [How many Cases do I need?](power-and-sample-size.md) | ✔ `executed` | `kno eval inspect` at three sizes — separable effect per behaviour, before you spend anything |
| [Check whether your evals can attribute anything](check-your-evals.md) | ✔ `executed` | The same command's full surface — the five checks, the tag caveat, the `--json` contract, and the fix for each finding |
| [Calibrate a judge](calibrate-a-judge.md) | ✔ `executed` | `kno judge calibrate` — kappa against human labels, the two ways the gate refuses, and what each one blames |
| [What a run costs](what-it-costs.md) | ✔ `executed` | Calls per stage, four real runs' planned counts, and the three caps |
| [Turn the logs you already have into an eval set](mine-your-transcripts.md) | ✔ `executed` (stage 1) | `kno mine` — formats, `--mode`, weak-label provenance, and what mining does not give you |
| [Point Kno at your own provider](your-own-provider.md) | • `flags-only` | Keys, cost caps, local model servers, and the first run that can bill you |
| [Score your agent against Claude](anthropic.md) | • `flags-only` | The Anthropic agent — `ANTHROPIC_API_KEY`, the priced models, and a complete baseline-to-report run |
| [Score your agent on Bedrock](bedrock.md) | • `flags-only` | The AWS agent — env-only credentials, regional pricing, and the cross-region profile refusal |
| [Score your agent on Vertex](vertex.md) | • `flags-only` | The Google Cloud agent — service-account JWT exchange, regional pricing, and the cross-region profile refusal |
| [Gate a deploy on Kno in CI](ci-gate.md) | ✔ `executed` (stage 1) | Exit codes, `--json`, and what to fail the build on |
| [Value a pool of assets](value-a-pool.md) | ✔ `executed` (stage 2) | Deltas with their intervals, the control's harm bound, and what `underpowered` means |
| [Choose a portfolio under budget](select-a-portfolio.md) | ✔ `executed` (stage 3) | Which assets earn their place, what the corrected intervals mean, and why the rejection log is a deliverable |
| [Export a tuning set](export-a-tuning-set.md) | ✔ `executed` (stage 4) | The destination grammar, the overwrite refusal, and the byte-identical re-export contract |
| [Read the whole story with `kno report`](read-the-whole-story.md) | ✔ `executed` (stage 5) | One page across the stages — what each section means, what "no cluster data" says, and why the holdout caveat is mandatory |
| [Delete stored conversation content](retention.md) | ✔ `executed` (stage 6) | What Kno keeps, what `kno purge` removes, and why it keeps the rest |
| [Read the results in a notebook](analyze-in-a-notebook.md) | • `flags-only` | `--json` into pandas, a forest plot, and the two things not to recompute |
| [Point Kno at the data you already have](from-your-warehouse.md) | • `flags-only` | Snowflake, BigQuery, Postgres, dbt and object storage into `cases.jsonl` and a Pool |
| [Schedule it: Airflow, Dagster, and what to fail the run on](orchestration.md) | • `manual` | Idempotent run ids, resume across retries, where to gate, and where `kno.db` lives |

Six of those are the six stages of one scenario, in order. That is not a coincidence and it is
the design: `select`, `export`, `report`, and `purge` all read a SQLite store an earlier stage
wrote, so if the *recipe* were the unit of execution none of them could ever be checked. Because
[`scenarios/support-refunds/run.sh`](../scenarios/support-refunds/run.sh) performs all six against
one store, each page asserts against its own stage — and each page whose stage is not the first
says so, next to its badge, naming the script to run first.

## The four pages that are their own scenario

`why-not-diy`, `power-and-sample-size`, `what-it-costs` and `mine-your-transcripts` are `executed`
against scenarios written for them, and each of those scenarios reaches a *different* verdict or
demonstrates a different mechanism:

| Recipe | Scenario | What its run establishes |
|---|---|---|
| [why-not-diy](why-not-diy.md) | [`diy-ablation`](../scenarios/diy-ablation/README.md) | A committed hundred-line ablation and Kno, same data, same agent, one names a winner |
| [power-and-sample-size](power-and-sample-size.md) | [`power-analysis`](../scenarios/power-analysis/README.md) | 12 → 40 → 160 Cases: separable effect 6.35 → 0.51 → 0.25, two checks clearing at different sizes |
| [what-it-costs](what-it-costs.md) | [`power-analysis`](../scenarios/power-analysis/README.md) | `Planning 567 measurements` printed before the first call |
| [mine-your-transcripts](mine-your-transcripts.md) | [`transcript-mining`](../scenarios/transcript-mining/README.md) | Transcripts in, 18 weak-label Cases out, and three of five checks flagged on what came back |
| [calibrate-a-judge](calibrate-a-judge.md) | [`judge-calibration`](../scenarios/judge-calibration/README.md) | kappa 0.867 against 60 human-labelled records, and the gate refusing at two floors for two different reasons |

## Vendor recipes

Every vendor recipe is the same shape — candidate content as Assets, real questions with vetted
answers as Cases, baseline, value, then act on the table — transplanted to a vendor's own data.
The [Zendesk recipe](zendesk.md) carries the vendor-swap table and the general explanation; the
rest are the vendor-specific export commands and read-back decisions.

Every one of them is `flags-only` or `manual`, and none of them is verified end to end. What CI
checks is that each `kno` command still exists with the flags the page names; what nobody checks
nightly is the vendor half, because doing so would mean holding long-lived credentials for eight
third-party products in a public repo's CI and spending money unattended. What checks the vendor
half is [`vendor-smoke`](../.github/workflows/vendor-smoke.yml): a human, with credentials, on an
approval-gated run, with the date written back by machine.

| Scenario | Vendor recipe | Tier |
|---|---|---|
| Support | [Zendesk](zendesk.md) · [HubSpot](hubspot.md) · [Salesforce](salesforce.md) | • `flags-only` |
| Coding agent | [GitHub](github.md) · [Jira + Confluence](jira.md) · [Confluence](confluence.md) | • `flags-only` |
| Eval platforms | [LangSmith](langsmith.md) · [Langfuse](langfuse.md) · [Braintrust](braintrust.md) · [Hugging Face](huggingface.md) — datasets as first-class Evals, no export step | • `flags-only` |
| E-commerce | [Shopify](shopify.md) | • `flags-only` |
| Payments | [Stripe](stripe.md) | • `flags-only` |
| Internal knowledge | [Notion](notion.md) | • `flags-only` |
| Warehouse and object storage | [Your own warehouse](from-your-warehouse.md) — Snowflake, BigQuery, Postgres, dbt, S3 | • `flags-only` |
| Workflow automation | [n8n](n8n.md) · [Airflow and Dagster](orchestration.md) | • `manual` |

`n8n` and `orchestration` are `manual` rather than `flags-only` on purpose. Their `kno` command
shapes *are* checked — the flag check runs on every page regardless of tier, because a renamed
flag is rot whether or not CI may execute the line. What the tier declares is what the *page*
claims, and both pages' subject is a workflow configured somewhere else: nodes and credentials in
a UI, or tasks and retries in a DAG. "The flags check out" says almost nothing about whether that
workflow works, so the pages do not imply that it does.

## Every credential a page needs, named

Every recipe lists in `credentials:` every environment variable its own commands imply — and a
lint fails the build if one is missing. That exists because of a specific observed failure: the
vendor pages taught a reader about the vendor's token and never mentioned that `--agent
openai:gpt-4.1` also bills OpenAI. `OPENAI_API_KEY` appeared once in the entire cookbook, in a
scheme table on one page, and was thereafter merely implied.

## Nothing is still in `uknoAI/kno`

All twenty-seven cookbook entries are here. The last two took the longest and for the same
reason: each documented a command that was on `uknoAI/kno@main` and in no release, and there is
no tier for "documents an unreleased command" — inventing one would let any page claim
verification against a binary that cannot run it.

Both reasons expired within a day of each other. v0.1.4 shipped `kno eval inspect` and
[check-your-evals](check-your-evals.md) migrated; v0.1.5 shipped `kno judge calibrate` and
[calibrate-a-judge](calibrate-a-judge.md) followed. Neither arrived as a `manual` page: both
commands are free, offline and deterministic, so both migrated as `executed` with a scenario
behind them.

Each leaves a one-line tombstone stub at its old path, which
[`scripts/cookbook-stub-check.sh`](https://github.com/uknoAI/kno/blob/main/scripts/cookbook-stub-check.sh)
holds to one line and one link. That gate now has an empty RESIDENT list, and the next page added
to it must name the release that will ship its command.

`check-your-evals` was in the same position and no longer is. `kno eval` was not a subcommand of
v0.1.2 when this repository was scaffolded; **v0.1.4 ships `kno eval inspect`**, so the page
migrated here as [Check whether your evals can attribute anything](check-your-evals.md), verified
as `executed` against the `power-analysis` scenario. Its illustrative 200-Case output was
replaced with the real 160-Case output CI asserts nightly — the page previously showed numbers
no run had produced, which is the exact thing a tier is supposed to prevent.

Both pages are declared in `uknoAI/kno`'s
[`scripts/cookbook-stub-check.sh`](https://github.com/uknoAI/kno/blob/main/scripts/cookbook-stub-check.sh),
which refuses a cookbook page that is neither a one-line tombstone stub nor a declared resident.
That gate is what makes "not migrated yet" impossible to spell the same way as "migrated and
tombstoned".

Supporting a two-level subcommand needed a change to the checker itself, which had only ever
resolved one word after `kno`: `--evals` looked up in `kno eval --help` — which lists only `-h` —
would have reported a working recipe as broken. `OpenBinary` now discovers children from the
binary the same way it discovers the root list, and
[a test corpus](../cmd/verify/testdata/nested-subcommand/) asserts that the child's real flags
pass *and* that an invented one still fails.
