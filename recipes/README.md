# Recipes

Task-shaped pages. Each one declares, in front matter, how far it has actually been verified —
see [VERIFICATION.md](../VERIFICATION.md) for exactly what each tier claims and does not claim.

`make render RECIPE=recipes/<name>.md` prints a page's verification block; the website renders
the same block from the same fields.

## Migrated so far

| Recipe | Tier | What it covers |
|---|---|---|
| [Score your agent for the first time](first-baseline.md) | ✔ `executed` | Getting from an eval file to a baseline, and reading what comes back |
| [Choose a portfolio under budget](select-a-portfolio.md) | ✔ `executed` (stage 3) | Which assets earn their place, what the corrected intervals mean, and why the rejection log is a deliverable |
| [Value your Zendesk knowledge](zendesk.md) | • `flags-only` | Articles, macros and policies as Assets; solved tickets as Cases |
| [Run Kno inside n8n](n8n.md) | • `manual` | Scheduled valuation with alerts, on the exit-code contract |

`n8n` is `manual` rather than `flags-only` on purpose. Its `kno` command shapes *are* checked —
the flag check runs on every page regardless of tier, because a renamed flag is rot whether or
not CI may execute the line. What the tier declares is what the *page* claims, and this page's
subject is an n8n workflow: nodes, credentials, and branches configured in a UI, with the `kno`
invocation living inside an Execute Command node. "The flags check out" says almost nothing about
whether that workflow works, so the page does not imply that it does.

## Still in `uknoAI/kno/docs/cookbook/`

Twenty-one entries have not been migrated yet, and nothing has been deleted from `uknoAI/kno` —
see the roadmap in [../README.md](../README.md). Each is a self-contained PR and they are the
designed on-ramp for a first contribution:

`anthropic` · `bedrock` · `braintrust` · `ci-gate` · `confluence` · `export-a-tuning-set` ·
`github` · `hubspot` · `huggingface` · `jira` · `langfuse` · `langsmith` · `notion` ·
`read-the-whole-story` · `retention` · `salesforce` · `shopify` · `stripe` · `value-a-pool` ·
`vertex` · `your-own-provider`

Four of those — `select-a-portfolio` (done), `export-a-tuning-set`, `read-the-whole-story`, and
`retention` — only *look* unrunnable. They read a SQLite store an earlier stage wrote, so they
are `executed` as later stages of a shared-store scenario rather than as standalone pages. That
single decision is what moves them from "cannot be checked" to "checked", and the
`requires-stages:` sentence is what keeps it honest.
