# `coding-agent`

An agent answering questions about a codebase's conventions — review expectations, test layout,
error style, tooling — and three candidate Assets that are **demonstrations rather than
documents**.

```
scenarios/coding-agent/
  evals/cases.jsonl     12 Cases across review, testing, style, and tooling
  pool/pool.jsonl        3 candidate Assets, all `kind: behavior`
  run.sh                 four stages: baseline, value, select, export
  expected/              one projected JSON document per stage, plus quotations.json
  DATA-PROVENANCE.md     who wrote this data, and the assertion that it is synthetic
```

Run it:

```sh
sh scenarios/coding-agent/run.sh /tmp/ca
```

It needs a released `kno` on `PATH` and nothing else.

## What makes this a different shape

`support-refunds` and `underpowered-eval` both carry a Pool of `kind: knowledge` — facts and
policies, the sort of thing that gets injected into context. This scenario's Pool is
`kind: behavior`, and Kno treats the two differently on purpose. From the schema:

> **knowledge** — Facts, policies, documents, reference material. Valued by context or
> knowledge-base injection, where ICL measurement is faithful.
>
> **behavior** — Format, tone, tool-use patterns, reasoning demonstrations. The only kind that
> faces the fine-tuning bridge at all.

Until this scenario, **every committed Asset in this repository was knowledge.** The routing
decision that sends an Asset to tuning rather than to context was exercised by nothing, which
meant a regression in it would have been invisible here. `run.sh` therefore exports with
`--destination tuning_set`, the side of the bridge only behavior can reach.

The three Assets are a review-comment voice, a commit-message shape, and a table-test shape.
None is a fact about the codebase; each is a pattern to imitate. That is what makes them
behavior rather than knowledge, and getting that classification wrong is exactly the mistake the
`kind` field exists to prevent — the schema warns that an unset kind "would read as knowledge
and route the Asset to the wrong destination".

## What to look for

**Twelve Cases, so the verdict is `no-effect` and not `underpowered`.** The reserve is
`int(dev * 0.3)` and an interval needs two; see
[`underpowered-eval`](../underpowered-eval/README.md) for where that boundary sits and why it is
worth its own scenario.

**The export writes zero assets, and that is the correct output.** The Portfolio is empty, so
`tuning.jsonl` carries no rows — but the run still records the destination, the manifest, and
the promise that re-exporting is byte-identical.

## What this does not show

The same caveat as every scenario here: the agent is `fake:`, which answers each Case with
exactly what the Case expects, so no Asset can move the score. No committed scenario can
demonstrate an Asset earning its place — see the roadmap in the [repository
README](../../README.md) for why that is not expressible offline. What is asserted here is the
routing and the arithmetic, not a win.
