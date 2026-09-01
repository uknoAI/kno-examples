# Contributing to kno-examples

This repository holds the scenarios and recipes for [Kno](https://github.com/uknoAI/kno). The
engineering process lives there —
[CONTRIBUTING.md](https://github.com/uknoAI/kno/blob/main/CONTRIBUTING.md) is the long form, and
[CLAUDE.md](https://github.com/uknoAI/kno/blob/main/CLAUDE.md) is the operating manual — and this
file covers what is different here.

[CODE_OF_CONDUCT.md](https://github.com/uknoAI/kno/blob/main/CODE_OF_CONDUCT.md) and
[SECURITY.md](https://github.com/uknoAI/kno/blob/main/SECURITY.md) apply by reference. Report a
vulnerability through Kno's private disclosure channel, never in a public issue here.

## Sign-off, not a CLA

Every commit needs a Developer Certificate of Origin sign-off:

```sh
git commit -s -m "docs: add the hubspot recipe"
```

That is the same posture as `uknoAI/kno`: DCO, no CLA, lower friction, sufficient for Apache-2.0.
The `dco` check enforces it.

Conventional Commits (`feat:`, `fix:`, `docs:`, `chore:`), squash-merge, linear history.

## Run the checks before you push

```sh
make check          # everything below
```

You need a released `kno` on your `PATH`. `make install-kno` fetches one through the project's
own `install.sh` if you have not got one.

| | What it does |
|---|---|
| `make lint` | Front matter, credentials, byte-identity between every quoted block and its `run.sh` region, and every relative markdown link in the tree |
| `make flags` | Every `kno` invocation on every page, against the released binary's `--help` and `kno doctor --json` |
| `make scenarios` | Every scenario end-to-end, twice, comparing against committed expectations and against itself |
| `make test` | The runner's own tests, including the deliberately broken corpus in `cmd/verify/testdata` |

`make update-expected` regenerates `expected/*.json` from a real run. It **preserves each
projection's key set** — it refreshes values, it never widens the projection into a full golden.
Review the diff like code.

## Adding a recipe

**Declare a tier.** A PR that adds a recipe with no `verification:` field fails the lint. It is
not a review convention.

```yaml
---
verification: flags-only
owner: "@your-handle"
verified-against: kno v0.1.2
last-manual-verification: 2026-08-31
credentials: [HUBSPOT_TOKEN, OPENAI_API_KEY]
---
```

**Name every credential.** Including the LLM one. `--agent openai:gpt-4.1` bills OpenAI, and a
reader following a vendor recipe from the top learns about the vendor's token and is never told
about the other one. The lint cross-references each `--agent`/`--evals`/`--pool` scheme against a
table and fails on an unnamed requirement, so this is enforced rather than remembered.

**`last-verified` and `verified-against` are CI's fields.** Do not hand-edit them. `manual` may
carry neither; `flags-only` carries `verified-against` only.

**Every relative link is checked.** `make lint` walks every `.md` in the tree and asserts each
relative target exists. External `https://` links are deliberately NOT checked — a nightly that
reddens on somebody else's outage is a signal people learn to override, the same reasoning that
keeps the vendor recipes at `flags-only`. Heading anchors are not checked either: a slug is a
claim about a renderer, not about this repository.

**Never re-type a command.** If your recipe shows a command a scenario runs, quote the scenario:

````markdown
```bash kno-run scenario=support-refunds stage=baseline
kno baseline --evals cases.jsonl --agent fake: --goal exact-match \
  --holdout-frac 0.2 --seed 1 --concurrency 1 \
  --db kno.db --run-id sr-baseline --yes
```
````

The lint asserts that text is byte-identical to the marked region of `scenarios/support-refunds/run.sh`.
There is exactly one copy of every flag in this repository and it is going to stay that way — a
second copy is how `uknoAI/kno`'s README and its VHS tape came to disagree about what a refund
policy says.

**Blocks are opt-in for execution.** Only a block tagged `kno-run` is ever executed. An untagged
block is inert — but it is *still parsed* for the flag check, because missing a check is silent
rot while running something unintended is destructive, and those failure modes deserve opposite
defaults.

## Adding a scenario

```
scenarios/<domain>-<task>/
  README.md            the story: what the agent does, what the Assets are, what to look for
  DATA-PROVENANCE.md   who wrote this data, and the assertion that it is synthetic
  evals/cases.jsonl    the Cases
  pool/pool.jsonl      the Pool
  run.sh               POSIX sh, the single source every recipe quotes
  expected/            one projected JSON document per stage, plus quotations.json
```

The directory names are the vocabulary — `evals/`, `pool/`, `expected/`. Not `data/`, not
`input/`, not `fixtures/`. Slugs are `<domain>-<task>`, lowercase, hyphenated.

**Anything beyond those six needs a reason in the scenario's own README.** Three exist today and
each states its reason on the page:

| | Scenario | Why |
|---|---|---|
| `transcripts/` | `transcript-mining` | `kno mine`'s input is not an eval set, and calling it `evals/` would be a lie about what it holds |
| `evals/generate.py` | `power-analysis` | 160 Cases whose wording is irrelevant; a committed program is a stronger provenance claim than a sentence about 160 lines |
| `naive_ablation.py` | `diy-ablation` | the scenario's subject is the alternative to Kno, and an alternative that is described rather than executed is not an answer |

**A generated or derived file needs a test.** `power-analysis/evals/cases.jsonl` must be exactly
what `generate.py` writes; `transcript-mining/evals/cases.jsonl` must be exactly what `kno mine`
writes over its transcripts. Both are asserted in
[`cmd/verify/scenarios_test.go`](cmd/verify/scenarios_test.go), because a committed artefact
nothing re-derives is a souvenir rather than an expectation, and it goes stale silently.

**A non-`kno` prerequisite is checked in `run.sh`, before anything runs.** `diy-ablation` needs a
`python3` and refuses with a sentence naming why the stage exists rather than exiting 127 out of
an `eval`. CI asserts the same prerequisite in its own step, so a runner image that drops it
fails as infrastructure rather than as a docs finding. Adding a prerequisite is a real cost —
weigh it against what the stage demonstrates.

**A scenario is added, never renamed.** A rename breaks every external link to it. Superseding
one means adding the successor and marking the predecessor `deprecated: true`, which the renderer
surfaces on the page.

**All scenario data must be synthetic.** This is a hard rule, and `DATA-PROVENANCE.md` naming an
author and asserting synthesis is required. A Case's `input` is an end user's words and its
`expected` is what somebody was told; Kno treats stored traces as customer data, and a public
repository of "example support tickets" is precisely where that rule gets broken by accident.
That accident is unrecoverable. This is a human gate and it is stated as one — no linter can tell
a plausible invented ticket from a real one.

**`run.sh` is plain POSIX `sh`.** No `gum`, no bashisms, no colour. CI and a reader's machine run
identical bytes, and `shellcheck --shell=sh` is part of `make check`.

**Pin every source of variation.** `--run-id`, `--seed`, `--routing-seed`, `--concurrency`,
`--holdout-frac`. CI runs each scenario twice and byte-compares; anything that varies is a Kno
bug, and the scenario is the detector.

**Budget the runtime.** A scenario that makes the nightly a forty-minute job nobody watches is
split, not tolerated.

## What CI will not do

- **Spend money.** Every nightly command runs against `fake:`, offline. The only workflow that
  can spend is `vendor-smoke`, which is `workflow_dispatch`-only, gated on a GitHub environment
  with required reviewers, and capped three independent ways (`--max-cost-usd`, `--max-calls`,
  and `KNO_MAX_COST_USD`). Nothing spends automatically. Kno's fourth prime directive applies to
  our own CI.
- **Gate `uknoAI/kno`.** This repository notices breakage; it does not block a Kno merge or
  release. A cross-repo gate that reddens on someone else's schedule becomes a gate people learn
  to override.
- **Accept a real API key.** `gitleaks` runs over the tree and the full history on every PR. A key
  that reaches this repository is compromised whether or not the commit is later removed.

## Where to start

The vendor recipes are the designed on-ramp, the same way Ring-1 adapters and judge prompts are
in `uknoAI/kno`. Twenty-one cookbook entries are still to be migrated (see the roadmap in
`README.md`); each is a self-contained PR: move the page, add its front matter, name every
credential it requires, and let the lint tell you what you missed.
