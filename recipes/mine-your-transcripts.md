---
verification: executed
scenario: transcript-mining
stage: mine
requires-stages: []
last-verified: 2026-09-05
verified-against: kno v0.2.1
---

# Turn the logs you already have into an eval set

Every other page here starts with a `cases.jsonl` you already had. This one starts where most
teams actually are: no eval set, and several years of support threads, ticket exports and chat
logs sitting in object storage.

`kno mine` reads those and writes Cases. Wherever a human corrected the agent, that exchange
becomes a Case whose expected outcome is what the human said it should be.

It makes **no LLM call**. It is a file-in, file-out transformation — the shape a pipeline step
wants — and it costs nothing.

## One command over an export directory

```bash kno-run scenario=transcript-mining stage=mine
kno mine --logs transcripts --out mined.jsonl --mode resolution --format auto
```

```
mined 18 Cases into mined.jsonl
  transcripts/chat.jsonl: 11 agent messages, 10 human replies, 10 mined (0 filtered)
```

`--logs` takes a file or a directory and is repeatable. `transcripts/` in the verified scenario
holds a JSONL chat export **and** a CSV ticket export side by side, the way an export directory
actually looks, and `--format auto` sniffs each file — so this is one command over a directory
rather than one command per source.

**Read the pairing summary, not the headline.** `11 agent messages, 10 human replies, 10 mined
(0 filtered)` is the diagnostic. A run where "mined" is far below "human replies" is a run whose
threads do not have the shape the mode assumes, and that is a fact about your logs worth knowing
before you tune anything.

## The three formats

| `--format` | Shape | Notes |
|---|---|---|
| `jsonl-chat` | one JSON message per line: `id` (required), `role`, `content`, optional `timestamp` and `thread_id` | `role` is one of `assistant`, `agent`, `user`, `human` |
| `markdown` | `**Speaker:**` lines, H1 or `---` thread boundaries, optional ISO timestamps | needs `--agent-name` to say which speaker is the agent; prints a speaker inventory |
| `csv` | a `question,answer` header row | a missing column is fatal, not skipped |

`auto` sniffs each file, so a mixed directory works. Name the format explicitly when you are
scripting against one source and want a surprise to be an error rather than a re-interpretation.

**The most common first failure is CSV quoting.** A comma inside an answer splits the row, and
the error names the file and the line:

```
error: the command cannot use the input it was given: transcripts/tickets.csv line 6: record on line 6: wrong number of fields
  fix: fix the reported file, then re-run
```

Quote the field. Most warehouse and helpdesk exporters do this correctly; hand-assembled CSVs
usually do not.

## `--mode` is the load-bearing choice

| Mode | The expected answer is | Use it when |
|---|---|---|
| `resolution` (default) | the thread's **final human message** — the answer that closed it | threads end when someone gives the right answer |
| `immediate` | the human reply after **each** agent answer, with chit-chat filtered | you want per-turn corrections rather than outcomes |

`immediate` counts and drops gratitude, acknowledgment, escalation, quote-back, retraction and
counter-questions. That filtering is real work and it is reported. But what survives is still an
*immediate* reply, and an immediate reply was never the thing the agent had to converge to.

**That distinction decides whether an exact-match goal is honest.** Mined Cases carry no rubric.
Under a judge goal they are fine either way. Under `exact-match` they are honest only when
`--mode resolution` has shaped the expected into a short answer, because a resolution is what
the agent had to reach and an aside is not. Kno states this itself in `kno mine --help`; it is
repeated here because it is the one choice on this page that quietly changes what your score
means.

## Every mined Case says where it came from

```json
{"id":"…","input":"The VPN client says my certificate expired. What do I do?","expected":"Renew the certificate from the self-service portal, then reconnect","derived":true,"derivation_note":"mined from transcripts/chat.jsonl; mode=resolution","source_ref":"transcripts/chat.jsonl"}
```

`derived`, `derivation_note` and `source_ref` are on every record and they survive ingestion.
`kno baseline` reports the count back:

```bash kno-run scenario=transcript-mining stage=baseline
kno baseline --evals mined.jsonl --agent fake: --goal exact-match \
  --holdout-frac 0.2 --seed 1 --concurrency 1 \
  --db kno.db --run-id tm-baseline --yes
```

```
Baseline tm-baseline
  cases      14 scored, 0 errored (of 14 dev; 4 held back)
  weak-label 18 of these Cases carry derived provenance (mined from transcripts or harvested from traces, not authored)
  score      1.000
```

A mined eval set is a **weak-label import**, not a hand-authored one, and that matters when
someone reads a score off it in three months. Nothing here can pass for authored, because the
run says how much of it was not. That is a property of the tool rather than a convention your
team has to maintain.

## What mining does not give you

Run the diagnostic immediately after mining. It is free and it will tell you what is missing:

```bash kno-run scenario=transcript-mining stage=inspect
kno eval inspect --evals mined.jsonl --holdout-frac 0.2
```

```
Evals  mined.jsonl
  18 Cases — 14 dev, 4 held back
  0 distinct behaviors (tags)

No dev Case carries a behavior tag.

  ! 100% of dev Cases carry no behavior tag
  ! the holdout has 4 Cases (20 is the minimum for a meaningful interval at
    validate)

3 of 5 checks flagged.
```

Two findings, and both belong in your pipeline design rather than in a bug report.

**Mining does not invent tags, and cannot.** A tag is a claim about which behaviour a Case
exercises, and no transcript states one. Without tags, routing runs in all-failed mode and
attributes nothing per behaviour, so the per-behaviour table that makes a Value run actionable
is simply absent. Tagging is a step you own — a classifier, a rule over the ticket's queue or
component field, or a human pass. Whatever it is, it is a step, and this is where you find out,
before spending anything.

**Eighteen Cases is not enough**, and the fix is more transcripts through the same command,
which is a scheduling decision rather than an authoring one. [How many Cases do I
need?](power-and-sample-size.md) has the arithmetic.

## Curating what came out

```bash
kno mine --logs transcripts --out mined.jsonl --review
```

`--review` presents each mined Case for keep, edit or drop on the terminal, writes the decisions
to a manifest beside the output, and reads that manifest back on the next run — so a Case you
dropped cannot resurrect when the pipeline re-mines a wider window. It needs a TTY, so it is a
human step and not a CI one, and it is deliberately not part of the verified scenario for that
reason.

`--min-cases` is the CI-shaped counterpart: it fails the command when fewer than N Cases were
mined, which turns "the export was empty this week" into a red job rather than a silently
shrinking eval set.

```bash
kno mine --logs transcripts --out mined.jsonl --min-cases 200
```

## Where it sits in a pipeline

```
transcripts/  →  kno mine  →  mined.jsonl  →  kno eval inspect   (a gate: checks_flagged)
                                           →  kno baseline       (a score)
                                           →  kno value          (a table of deltas)
                                           →  kno select         (a portfolio, or a refusal)
```

`kno mine` reads files and writes a file, so it schedules like any other transform. Everything
downstream of `mined.jsonl` is the ordinary loop.

## The rule this page cannot enforce for you

Mining is mechanical: first human message in, last human message out. Whether the result is a
*good* eval Case depends entirely on whether the human who closed that thread was right.

Nothing checks that. `--review` is the mechanism for checking it, and a mined set that has never
been reviewed is a set of assertions your support queue made, weighted by how often people
asked. That is often a fine place to start and it is never a place to stop.

## Next

- [How many Cases do I need?](power-and-sample-size.md) — what to do about the two flagged
  checks.
- [Score your agent for the first time](first-baseline.md) — reading the baseline the mined set
  produces.
- [`scenarios/transcript-mining/README.md`](../scenarios/transcript-mining/README.md) — the
  scenario, its synthetic transcripts, and the provenance rule that governs them.
