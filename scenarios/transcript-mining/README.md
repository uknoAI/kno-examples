# `transcript-mining`

The scenario that starts one step earlier than every other one here: with **transcripts**, not
with an eval set.

Every other scenario in this repository begins with a `cases.jsonl` somebody already had. That
is a fair assumption for a demonstration and a bad one for a first day, because the honest state
of most teams is: no eval set, and several years of support threads, ticket exports and chat
logs sitting in object storage. This scenario is about the step between those two facts, and
about what the step does *not* give you.

```
scenarios/transcript-mining/
  transcripts/chat.jsonl   10 synthetic helpdesk threads, JSONL chat export
  transcripts/tickets.csv  8 synthetic question/answer rows, CSV ticket export
  evals/cases.jsonl        the 18 mined Cases — an OUTPUT, committed as an expectation
  pool/pool.jsonl          3 candidate runbooks
  run.sh                   five stages: mine, inspect, baseline, value, select
  expected/                one projected JSON document per stage, plus quotations.json
  DATA-PROVENANCE.md       who wrote this data, and the assertion that it is synthetic
```

Run it:

```sh
sh scenarios/transcript-mining/run.sh /tmp/tm
```

It needs a released `kno` on `PATH` and nothing else.

## One command reads both formats

```
mined 18 Cases into mined.jsonl
  transcripts/chat.jsonl: 11 agent messages, 10 human replies, 10 mined (0 filtered)
```

`transcripts/` holds a JSONL chat export and a CSV ticket export, side by side, the way an
export directory actually looks. `--format auto` sniffs each file, so this is one command over a
directory rather than one command per source. Ten of the eighteen Cases came from the chat log;
the other eight came from the CSV.

The pairing summary is per file and it is the number to read: `11 agent messages, 10 human
replies, 10 mined (0 filtered)`. A run where "mined" is far below "human replies" is a run whose
threads do not have the shape `--mode resolution` assumes, and that is a fact about your logs
worth knowing before you tune anything.

## What `--mode resolution` decides

`resolution` says the expected answer is **the message that closed the thread** — in these
transcripts, the human engineer's final reply after the agent's weak first attempt.

That is the only reading that is honest under an exact-match goal, and the reason is worth
stating: the alternative, `--mode immediate`, takes the human's reply after *each* agent turn.
Those replies are often "that did not help" or "thanks" — not answers. `immediate` filters
gratitude, acknowledgment, escalation, quote-back, retraction and counter-questions and counts
what it dropped, but what survives is still an *immediate* reply, and an immediate reply was
never the thing the agent had to converge to. Scoring against it with exact-match asks the agent
to produce a sentence nobody claimed was correct.

## Every mined Case carries where it came from

```json
{"id":"…","input":"The VPN client says my certificate expired. What do I do?","expected":"Renew the certificate from the self-service portal, then reconnect","derived":true,"derivation_note":"mined from transcripts/chat.jsonl; mode=resolution","source_ref":"transcripts/chat.jsonl"}
```

`derived`, `derivation_note` and `source_ref` are on every record, they survive ingestion, and
`kno baseline` reports the count back:

```
  weak-label 18 of these Cases carry derived provenance (mined from transcripts or harvested from traces, not authored)
```

That line is the point of the whole provenance mechanism. A mined eval set is a **weak-label
import**, not a hand-authored one, and the difference matters when someone three months from now
reads a score off it. Nothing here can pass for authored, because the run says how much of it
was not.

## What mining does not give you, stated by the tool

`kno eval inspect` runs immediately after mining, and it flags three of five checks:

```
  18 Cases — 14 dev, 4 held back
  0 distinct behaviors (tags)

No dev Case carries a behavior tag.
…
  ! 100% of dev Cases carry no behavior tag
  ! the holdout has 4 Cases (20 is the minimum for a meaningful interval at
    validate)

3 of 5 checks flagged.
```

Two findings, and they are the two a pipeline author needs on day one:

- **Mining does not invent tags.** It cannot: a tag is a claim about which behaviour a Case
  exercises, and no transcript states one. Without tags, routing runs in all-failed mode and
  attributes nothing per behaviour, so the per-behaviour table that makes a Value run
  actionable is simply absent. Tagging is a step in *your* pipeline, and this is where you find
  out, before spending anything.
- **Eighteen Cases is not enough.** Four held back is nowhere near the twenty a meaningful
  `validate` interval needs. The fix is more transcripts through the same command, which is a
  scheduling decision rather than an authoring one — see
  [`power-analysis`](../power-analysis/README.md) for exactly how many.

Neither of those is a failure of the mining step. Both are the mining step telling you what it
did and did not do, in a command that costs nothing and constructs no agent.

## Where this sits in a pipeline

The five stages are one dependency chain, and the shape is what an orchestrator wants:

```
transcripts/  →  kno mine  →  mined.jsonl  →  kno eval inspect   (a gate: checks_flagged)
                                           →  kno baseline       (a score)
                                           →  kno value          (a table of deltas)
                                           →  kno select         (a portfolio, or a refusal)
```

`kno mine` reads files and writes a file. `kno eval inspect` reads a file, writes nothing,
constructs no agent, and exits 0 whether zero or five checks are flagged — it is a diagnostic,
so a job that wants a gate reads `checks_flagged` from `--json` and picks its own threshold.
Everything after that is the ordinary loop.

The committed `evals/cases.jsonl` is worth one sentence of explanation, because it is the one
place this scenario differs from the others: it is an **output**, not an input. The stages after
`mine` read `mined.jsonl` — the file the run just produced — so the pipeline is exercised rather
than bypassed. `evals/cases.jsonl` is the committed copy of what mining should produce, and
[a test](../../cmd/verify/scenarios_test.go) asserts the two are byte-identical. If a release
changes what `kno mine` writes, that test names it.

## What this does not show

The agent is `fake:`, so the score is 1.000 and every delta is exactly zero, and the same
ceiling applies here as everywhere else in this repository.

More specific to this scenario: **the quality of a mined Case is not verified by anything.** The
mining step is mechanical — first human message in, last human message out — and whether the
result is a good eval Case depends entirely on whether the thread was actually resolved
correctly by the human. These transcripts were written so that it was. Yours were not written at
all. `kno mine --review` exists for that reason: it presents each mined Case for keep, edit or
drop on the terminal, writes the decisions to a manifest, and reads the manifest back on the next
run so a curated drop cannot resurrect. It needs a TTY, so it is not something CI can execute,
and it is not exercised here.
