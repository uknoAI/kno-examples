# `eval-platform`

An **LLM-as-judge**: each Case hands the model another model's answer and asks for a grade.

```
scenarios/eval-platform/
  evals/cases.jsonl     12 Cases across correctness, citation, tone, and scope
  pool/pool.jsonl        3 candidate Assets — two `knowledge`, one `behavior`
  run.sh                 four stages: baseline, value, select, export
  expected/              one projected JSON document per stage, plus quotations.json
  DATA-PROVENANCE.md     who wrote this data, and the assertion that it is synthetic
```

Run it:

```sh
sh scenarios/eval-platform/run.sh /tmp/ep
```

It needs a released `kno` on `PATH` and nothing else.

## What makes this a different shape

**The input is not a question.** In `support-refunds` a Case input is what a customer asked. Here
it is a complete question-and-answer pair, and the expected output is a label:

```json
{"id":"cite-02","input":"Question: What is the shipping window? Answer: Standard shipping takes 3-5 business days. Grade it.","expected":"fail: no source cited"}
```

That is the shape an eval platform's dataset actually has — a Braintrust dataset row, a Langfuse
trace scored after the fact, a LangSmith run annotated by a judge. Four vendor recipes here
([`braintrust`](../../recipes/braintrust.md), [`langfuse`](../../recipes/langfuse.md),
[`langsmith`](../../recipes/langsmith.md), [`huggingface`](../../recipes/huggingface.md))
describe exactly this and, before this scenario, had only a customer-support flow to point at.

**The Pool is mixed, which nothing else here is.** Two Assets are `knowledge` — the rubric the
judge applies, and the label vocabulary it must write in. One is `behavior` — how to grade,
which is a disposition rather than a fact. Kno routes the two kinds to different destinations,
so a Pool carrying both is the case where routing has to decide rather than send everything one
way.

`run.sh` exports with `--destination context`; [`coding-agent`](../coding-agent/README.md)
exports the tuning side. Between them both destinations are covered.

## What to look for

**The labels are terse on purpose.** `pass`, or `fail: <reason>` where the reason is a noun
phrase. A judge that writes prose cannot be scored by exact match, and a rubric that permits
prose is a rubric that cannot be measured — which is the practical reason the `label-vocabulary`
Asset exists in the Pool at all.

**Nine of the twelve Cases fail the answer being graded.** A judge dataset made only of passes
tests nothing: it cannot distinguish a working judge from one that returns `pass`
unconditionally.

## What this does not show

The same caveat as every scenario here: the agent is `fake:`, which answers each Case with
exactly what the Case expects, so no Asset can move the score. No committed scenario can
demonstrate an Asset earning its place — see the roadmap in the [repository
README](../../README.md). What is asserted here is the routing and the arithmetic, not a win.
