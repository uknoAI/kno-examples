# Data provenance — `eval-platform`

**All data in this scenario is synthetic.** None of it is a real model output, a real graded
trace, a real eval dataset, or a derivative of any of those. Nothing was exported from
Braintrust, Langfuse, LangSmith, Hugging Face, or any other platform. No personal data, no
confidential data, no scraped data.

## The Cases

`evals/cases.jsonl` holds twelve Cases, written by hand for this scenario. Each input is an
invented question-and-answer pair; each expected output is the grade a correct judge would
return.

The answers being graded were written to be wrong in specific, ordinary ways — vague, uncited,
dismissive, or claiming an authority the agent does not have. They are not transcripts, and no
model produced them.

Four tags, three Cases each: `correctness`, `citation`, `tone`, `scope`. Nine of the twelve
expect a failing grade, deliberately: a judge dataset made only of passes cannot distinguish a
working judge from one that always says `pass`.

The refund and shipping facts these Cases grade against are the same synthetic policies used by
[`support-refunds`](../support-refunds/DATA-PROVENANCE.md), so the two scenarios describe one
imaginary company rather than two.

## The Pool

`pool/pool.jsonl` holds three synthetic candidate Assets: a grading rubric and a label
vocabulary (`kind: knowledge`), and a grading disposition (`kind: behavior`). All three were
written for this scenario.

## Licence

Apache-2.0, the same as the rest of this repository.
