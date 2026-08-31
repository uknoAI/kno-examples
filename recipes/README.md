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

## Core recipes

| Recipe | Tier | What it covers |
|---|---|---|
| [Score your agent for the first time](first-baseline.md) | ✔ `executed` | Getting from an eval file to a baseline, and reading what comes back |
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

Six of those are the six stages of one scenario, in order. That is not a coincidence and it is
the design: `select`, `export`, `report`, and `purge` all read a SQLite store an earlier stage
wrote, so if the *recipe* were the unit of execution none of them could ever be checked. Because
[`scenarios/support-refunds/run.sh`](../scenarios/support-refunds/run.sh) performs all six against
one store, each page asserts against its own stage — and each page whose stage is not the first
says so, next to its badge, naming the script to run first.

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
| Workflow automation | [n8n](n8n.md) — scheduled valuation with alerts, on the exit-code contract | • `manual` |

`n8n` is `manual` rather than `flags-only` on purpose. Its `kno` command shapes *are* checked —
the flag check runs on every page regardless of tier, because a renamed flag is rot whether or
not CI may execute the line. What the tier declares is what the *page* claims, and this page's
subject is an n8n workflow: nodes, credentials, and branches configured in a UI, with the `kno`
invocation living inside an Execute Command node. "The flags check out" says almost nothing about
whether that workflow works, so the page does not imply that it does.

## Every credential a page needs, named

Every recipe lists in `credentials:` every environment variable its own commands imply — and a
lint fails the build if one is missing. That exists because of a specific observed failure: the
vendor pages taught a reader about the vendor's token and never mentioned that `--agent
openai:gpt-4.1` also bills OpenAI. `OPENAI_API_KEY` appeared once in the entire cookbook, in a
scheme table on one page, and was thereafter merely implied.

## Still in `uknoAI/kno`

One page has not migrated: **`check-your-evals`**, which documents `kno eval inspect`. That
command exists on `uknoAI/kno@main` and is **not in any release** — `kno eval` is not a subcommand
of v0.1.2 — so the flag check would correctly report every command on it as broken against the
binary a reader can actually download. It stays at
[`docs/cookbook/check-your-evals.md`](https://github.com/uknoAI/kno/blob/main/docs/cookbook/check-your-evals.md)
with the code that will release it, and migrates in the release window that ships `kno eval
inspect`. There is no tier for "documents an unreleased command", and inventing one would let any
page claim verification against a binary that cannot run it.
